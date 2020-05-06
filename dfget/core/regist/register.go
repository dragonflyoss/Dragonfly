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
	"github.com/dragonflyoss/Dragonfly/dfget/locator"
	"github.com/dragonflyoss/Dragonfly/dfget/types"
	"github.com/dragonflyoss/Dragonfly/pkg/constants"
	"github.com/dragonflyoss/Dragonfly/pkg/errortypes"
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
	locator            locator.SupernodeLocator
	cfg                *config.Config
	lastRegisteredNode *locator.Supernode
}

var _ SupernodeRegister = &supernodeRegister{}

// NewSupernodeRegister creates an instance of supernodeRegister.
func NewSupernodeRegister(cfg *config.Config, api api.SupernodeAPI, locator locator.SupernodeLocator) SupernodeRegister {
	return &supernodeRegister{
		api:     api,
		locator: locator,
		cfg:     cfg,
	}
}

// Register processes the flow of register.
func (s *supernodeRegister) Register(peerPort int) (*RegisterResult, *errortypes.DfError) {
	var (
		resp       *types.RegisterResponse
		e          error
		node       *locator.Supernode
		retryTimes = 0
		start      = time.Now()
	)

	nextOrRetry := func() *locator.Supernode {
		if resp != nil && resp.Code == constants.CodeWaitAuth && retryTimes < 3 {
			retryTimes++
			logrus.Infof("sleep 1.0 s to wait auth(%d/3)...", retryTimes)
			time.Sleep(1000 * time.Millisecond)
			return s.locator.Get()
		}
		return s.locator.Next()
	}

	logrus.Infof("do register to one of %v", s.locator)
	req := s.constructRegisterRequest(peerPort)
	for node = s.locator.Next(); node != nil; node = nextOrRetry() {
		if s.lastRegisteredNode == node {
			logrus.Warnf("the last registered node is the same(%v)", s.lastRegisteredNode)
			continue
		}
		req.SupernodeIP = node.IP
		nodeHost := nodeHostStr(node)
		resp, e = s.api.Register(nodeHost, req)
		logrus.Infof("do register to %s, res:%s error:%v", nodeHost, resp, e)
		if e != nil {
			continue
		}
		if resp.Code == constants.Success || resp.Code == constants.CodeNeedAuth ||
			resp.Code == constants.CodeURLNotReachable {
			break
		}
	}

	s.setLastRegisteredNode(node)
	if err := s.checkResponse(resp, e); err != nil {
		logrus.Errorf("register fail:%v", err)
		return nil, err
	}

	result := NewRegisterResult(nodeHostStr(node), s.cfg.URL,
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

func (s *supernodeRegister) setLastRegisteredNode(node *locator.Supernode) {
	s.lastRegisteredNode = node
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

func nodeHostStr(node *locator.Supernode) string {
	if node == nil {
		return ""
	}
	return node.String()
}

func getTaskPath(taskFileName string) string {
	if !stringutils.IsEmptyStr(taskFileName) {
		return config.PeerHTTPPathPrefix + taskFileName
	}
	return ""
}

// NewRegisterResult creates an instance of RegisterResult.
func NewRegisterResult(node, url, taskID string,
	fileLen int64, pieceSize int32, cdnSource apiTypes.CdnSource) *RegisterResult {
	return &RegisterResult{
		Node:       node,
		URL:        url,
		TaskID:     taskID,
		FileLength: fileLen,
		PieceSize:  pieceSize,
		CDNSource:  cdnSource,
	}
}

// RegisterResult is the register result set.
type RegisterResult struct {
	Node       string
	URL        string
	TaskID     string
	FileLength int64
	PieceSize  int32
	CDNSource  apiTypes.CdnSource
}

func (r *RegisterResult) String() string {
	return util.JSONString(r)
}
