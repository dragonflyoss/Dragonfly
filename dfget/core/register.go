/*
 * Copyright 1999-2018 Alibaba Group.
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

package core

import (
	"os"
	"time"

	"github.com/alibaba/Dragonfly/dfget/config"
	"github.com/alibaba/Dragonfly/dfget/core/api"
	"github.com/alibaba/Dragonfly/dfget/errors"
	"github.com/alibaba/Dragonfly/dfget/types"
	"github.com/alibaba/Dragonfly/dfget/util"
	"github.com/alibaba/Dragonfly/version"
)

// SupernodeRegister encapsulates the Register steps into a struct.
type SupernodeRegister interface {
	Register(peerPort int) (*RegisterResult, *errors.DFGetError)
}

type supernodeRegister struct {
	api api.SupernodeAPI
	ctx *config.Context
}

// NewSupernodeRegister creates an instance of supernodeRegister.
func NewSupernodeRegister(ctx *config.Context, api api.SupernodeAPI) SupernodeRegister {
	return &supernodeRegister{
		api: api,
		ctx: ctx,
	}
}

// Register processes the flow of register.
func (s *supernodeRegister) Register(peerPort int) (*RegisterResult, *errors.DFGetError) {
	var (
		resp       *types.RegisterResponse
		e          error
		i          int
		retryTimes = 0
		start      = time.Now()
	)

	s.ctx.ClientLogger.Infof("do register to one of %v", s.ctx.Node)
	nodes, nLen := s.ctx.Node, len(s.ctx.Node)
	req := s.constructRegisterRequest(peerPort)
	for i = 0; i < nLen; i++ {
		req.SupernodeIP = nodes[i]
		resp, e = s.api.Register(nodes[i], req)
		s.ctx.ClientLogger.Infof("do register to %s, res:%s error:%v", nodes[i], resp, e)
		if e != nil {
			s.ctx.ClientLogger.Errorf("register to node:%s error:%v", nodes[i], e)
			continue
		}
		if resp.Code == config.Success || resp.Code == config.TaskCodeNeedAuth {
			break
		}
		if resp.Code == config.TaskCodeWaitAuth && retryTimes < 3 {
			i--
			retryTimes++
			s.ctx.ClientLogger.Infof("sleep 2.5s to wait auth(%d/3)...", retryTimes, 3)
			time.Sleep(2500 * time.Millisecond)
		}
	}
	s.setRemainderNodes(i)
	if err := s.checkResponse(resp, e); err != nil {
		s.ctx.ClientLogger.Errorf("register fail:%v", err)
		return nil, err
	}

	result := NewRegisterResult(nodes[i], s.ctx.Node, s.ctx.URL,
		resp.Data.TaskID, resp.Data.FileLength, resp.Data.PieceSize)

	s.ctx.ClientLogger.Infof("do register result:%s and cost:%.3fs", resp,
		time.Since(start).Seconds())
	return result, nil
}

func (s *supernodeRegister) checkResponse(resp *types.RegisterResponse, e error) *errors.DFGetError {
	if e != nil {
		return errors.New(config.HTTPError, e.Error())
	}
	if resp == nil {
		return errors.New(config.HTTPError, "empty response, unknown error")
	}
	if resp.Code != config.Success {
		return errors.New(resp.Code, resp.Msg)
	}
	return nil
}

func (s *supernodeRegister) setRemainderNodes(idx int) {
	nLen := len(s.ctx.Node)
	if nLen <= 0 {
		return
	}
	if idx < nLen {
		s.ctx.Node = s.ctx.Node[idx+1:]
	} else {
		s.ctx.Node = []string{}
	}
}

func (s *supernodeRegister) constructRegisterRequest(port int) *types.RegisterRequest {
	ctx := s.ctx
	hostname, _ := os.Hostname()
	req := &types.RegisterRequest{
		RawURL:     ctx.URL,
		TaskURL:    ctx.RV.TaskURL,
		Cid:        ctx.RV.Cid,
		IP:         ctx.RV.LocalIP,
		HostName:   hostname,
		Port:       port,
		Path:       getTaskPath(ctx.RV.TaskFileName),
		Version:    version.DFGetVersion,
		CallSystem: ctx.CallSystem,
		Headers:    ctx.Header,
		Dfdaemon:   ctx.DFDaemon,
	}
	if ctx.Md5 != "" {
		req.Md5 = ctx.Md5
	} else if ctx.Identifier != "" {
		req.Identifier = ctx.Identifier
	}
	return req
}

// NewRegisterResult creates a instance of RegisterResult.
func NewRegisterResult(node string, remainder []string, url string,
	taskID string, fileLen int64, pieceSize int32) *RegisterResult {
	return &RegisterResult{
		Node:           node,
		RemainderNodes: remainder,
		URL:            url,
		TaskID:         taskID,
		FileLength:     fileLen,
		PieceSize:      pieceSize,
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
}

func (r *RegisterResult) String() string {
	return util.JSONString(r)
}
