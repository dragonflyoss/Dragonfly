/*
 * Copyright The Dragonfly Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package regist

import (
	"io/ioutil"
	"os"
	"time"

	apiTypes "github.com/dragonflyoss/Dragonfly/apis/types"
	"github.com/dragonflyoss/Dragonfly/dfget/config"
	"github.com/dragonflyoss/Dragonfly/dfget/core/api"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/pkg/constants"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
	"github.com/dragonflyoss/Dragonfly/pkg/netutils"
	"github.com/dragonflyoss/Dragonfly/pkg/stringutils"
	"github.com/dragonflyoss/Dragonfly/pkg/util"
	"github.com/dragonflyoss/Dragonfly/version"

	"github.com/sirupsen/logrus"
)

// SupernodeRegister encapsulates the Register steps into a struct.
type SupernodeRegister interface {
	Register(peerPort int) (*RegisterResult, *errortypes.DfError)
}

type supernodeRegister struct {
	api                api.SupernodeAPI
	cfg                *config.Config
	lastRegisteredNode string
}

var _ SupernodeRegister = &supernodeRegister{}

// NewSupernodeRegister creates an instance of supernodeRegister.
func NewSupernodeRegister(cfg *config.Config, api api.SupernodeAPI) SupernodeRegister {
	return &supernodeRegister{
		api: api,
		cfg: cfg,
	}
}

// Register processes the flow of register.
func (s *supernodeRegister) Register(peerPort int) (*RegisterResult, *errortypes.DfError) {
	var (
		resp       *types.RegisterResponse
		e          error
		i          int
		retryTimes = 0
		start      = time.Now()
	)

	logrus.Infof("do register to one of %v", s.cfg.Nodes)
	nodes, nLen := s.cfg.Nodes, len(s.cfg.Nodes)
	req := s.constructRegisterRequest(peerPort)
	for i = 0; i < nLen; i++ {
		if s.lastRegisteredNode == nodes[i] {
			logrus.Warnf("the last registered node is the same(%s)", nodes[i])
			continue
		}
		req.SupernodeIP = netutils.ExtractHost(nodes[i])
		resp, e = s.api.Register(nodes[i], req)
		logrus.Infof("do register to %s, res:%s error:%v", nodes[i], resp, e)
		if e != nil {
			logrus.Errorf("register to node:%s error:%v", nodes[i], e)
			continue
		}
		if resp.Code == constants.Success || resp.Code == constants.CodeNeedAuth ||
			resp.Code == constants.CodeURLNotReachable {
			break
		}
		if resp.Code == constants.CodeWaitAuth && retryTimes < 3 {
			i--
			retryTimes++
			logrus.Infof("sleep 2.5s to wait auth(%d/3)...", retryTimes)
			time.Sleep(2500 * time.Millisecond)
		}
	}
	s.setLastRegisteredNode(i)
	s.setRemainderNodes(i)
	if err := s.checkResponse(resp, e); err != nil {
		logrus.Errorf("register fail:%v", err)
		return nil, err
	}

	result := NewRegisterResult(nodes[i], s.cfg.Nodes, s.cfg.URL,
		resp.Data.TaskID, resp.Data.FileLength, resp.Data.PieceSize, resp.Data.CDNSource)

	logrus.Infof("do register result:%s and cost:%.3fs", resp,
		time.Since(start).Seconds())
	return result, nil
}

func (s *supernodeRegister) checkResponse(resp *types.RegisterResponse, e error) *errortypes.DfError {
	if e != nil {
		return errortypes.New(constants.HTTPError, e.Error())
	}
	if resp == nil {
		return errortypes.New(constants.HTTPError, "empty response, unknown error")
	}
	if resp.Code != constants.Success {
		return errortypes.New(resp.Code, resp.Msg)
	}
	return nil
}

func (s *supernodeRegister) setLastRegisteredNode(idx int) {
	nLen := len(s.cfg.Nodes)
	if nLen <= 0 {
		return
	}
	if idx >= nLen {
		s.lastRegisteredNode = ""
		return
	}
	s.lastRegisteredNode = s.cfg.Nodes[idx]
}

func (s *supernodeRegister) setRemainderNodes(idx int) {
	nLen := len(s.cfg.Nodes)
	if nLen <= 0 {
		return
	}
	if idx >= nLen {
		s.cfg.Nodes = []string{}
		return
	}

	s.cfg.Nodes = s.cfg.Nodes[idx+1:]
}

func (s *supernodeRegister) constructRegisterRequest(port int) *types.RegisterRequest {
	cfg := s.cfg
	hostname, _ := os.Hostname()
	req := &types.RegisterRequest{
		RawURL:     cfg.URL,
		TaskURL:    cfg.RV.TaskURL,
		Cid:        cfg.RV.Cid,
		IP:         cfg.RV.LocalIP,
		HostName:   hostname,
		Port:       port,
		Path:       getTaskPath(cfg.RV.TaskFileName),
		Version:    version.DFGetVersion,
		CallSystem: cfg.CallSystem,
		Headers:    cfg.Header,
		Dfdaemon:   cfg.DFDaemon,
		Insecure:   cfg.Insecure,
	}
	if cfg.Md5 != "" {
		req.Md5 = cfg.Md5
	} else if cfg.Identifier != "" {
		req.Identifier = cfg.Identifier
	}

	for _, certPath := range cfg.Cacerts {
		caBytes, err := ioutil.ReadFile(certPath)
		if err != nil {
			logrus.Errorf("read cert file fail:%v", err)
			continue
		}
		req.RootCAs = append(req.RootCAs, caBytes)
	}

	return req
}

func getTaskPath(taskFileName string) string {
	if !stringutils.IsEmptyStr(taskFileName) {
		return config.PeerHTTPPathPrefix + taskFileName
	}
	return ""
}

// NewRegisterResult creates an instance of RegisterResult.
func NewRegisterResult(node string, remainder []string, url string,
	taskID string, fileLen int64, pieceSize int32, cdnSource apiTypes.CdnSource) *RegisterResult {
	return &RegisterResult{
		Node:           node,
		RemainderNodes: remainder,
		URL:            url,
		TaskID:         taskID,
		FileLength:     fileLen,
		PieceSize:      pieceSize,
		CDNSource:      cdnSource,
	}
}

// RegisterResult is the register result set.
type RegisterResult struct {
	Node           string
	RemainderNodes []string
	URL            string
	TaskID         string
	FileLength     int64
	PieceSize      int32
	CDNSource      apiTypes.CdnSource
}

func (r *RegisterResult) String() string {
	return util.JSONString(r)
}
