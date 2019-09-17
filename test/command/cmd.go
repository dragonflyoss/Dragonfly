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

package command

import (
	"container/list"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	fp "path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dragonflyoss/Dragonfly/pkg/httputils"
	"github.com/dragonflyoss/Dragonfly/test/environment"
)

var (
	dfgetPath     string
	dfdaemonPath  string
	supernodePath string
)

func init() {
	var (
		dir string
		err error
	)

	if dir, err = os.Getwd(); err != nil {
		panic(fmt.Errorf("initialize cmd failed:%v", err))
	}
	idx := strings.LastIndex(dir, fp.Join(string(fp.Separator), "test"))
	sourceDir := dir[:idx]

	binDir := fp.Join(sourceDir, "bin", runtime.GOOS+"_"+runtime.GOARCH)

	dfgetPath = fp.Join(binDir, "dfget")
	dfdaemonPath = fp.Join(binDir, "dfdaemon")
	supernodePath = fp.Join(binDir, "supernode")
}

func checkExist(s string) {
	if _, err := os.Stat(s); err != nil {
		panic(fmt.Sprintf("please execute 'make build' before starting "+
			"integration testing:%v", err))
	}
}

// NewStarter creates an instance of Starter.
// It checks the binary files whether is existing and creates a temporary
// directory for testing.
func NewStarter(name string) *Starter {
	checkExist(dfgetPath)
	checkExist(dfdaemonPath)
	checkExist(supernodePath)

	home, err := ioutil.TempDir("/tmp", "df-"+name+"-")
	if err != nil {
		panic(err)
	}
	return &Starter{
		Name:    name,
		Home:    home,
		cmdList: list.New(),
		listMap: make(map[*exec.Cmd]*list.Element),
		fileSrv: make(map[*exec.Cmd]*http.Server),
	}
}

// Starter is a set of all components' starter.
// It provides an easy way to manager processes and recycle resources for
// integration testing. For example:
//     type ExampleSuite struct {
//         starter *Starter
//     }
//
//     func (s *ExampleSuite) SetUpSuite(c *check.C) {
//         s.starter = NewStarter("ExampleSuite")
//     }
//
//     func (s *ExampleSuite) TearDownSuite(c *check.C) {
//         s.starter.Clean()
//     }
type Starter struct {
	// Name identify the Starter instance.
	Name string

	// Home is the temporary working home.
	Home string

	// cmdList stores the created processes.
	cmdList *list.List

	// listMap maps a process to list.Element.
	// It's used for remove a corresponding element from cmdList when a process
	// is killed.
	listMap map[*exec.Cmd]*list.Element

	// fileSrv maps a supernode processes to a corresponding file server.
	// It's used for shutdown the file server when a corresponding supernode
	// is killed.
	fileSrv map[*exec.Cmd]*http.Server

	// supernodeFileServerHome is the home dir of supernode file server.
	supernodeFileServerHome string

	lock sync.Mutex
}

func (s *Starter) getCmdGo(dir string, running time.Duration, args ...string) (cmd *exec.Cmd, err error) {
	args = append([]string{
		"--home-dir=" + dir,
		"--port=" + strconv.Itoa(environment.SupernodeListenPort),
		"--advertise-ip=127.0.0.1",
		"--download-port=" + strconv.Itoa(environment.SupernodeDownloadPort),
		"--debug",
	}, args...)

	return s.execCmd(running, supernodePath, args...)
}

// Supernode starts supernode.
func (s *Starter) Supernode(running time.Duration, args ...string) (
	cmd *exec.Cmd, err error) {
	dir := fp.Join(s.Home, "supernode")
	cmd, err = s.getCmdGo(dir, running, args...)
	if err != nil {
		return nil, err
	}

	if err = check("localhost", environment.SupernodeListenPort, 10*time.Second); err != nil {
		s.Kill(cmd)
		return nil, err
	}
	s.supernodeFileServerHome = fp.Join(dir, "repo")
	if _, err = s.fileServer(cmd, s.supernodeFileServerHome, environment.SupernodeDownloadPort); err != nil {
		s.Kill(cmd)
		return nil, err
	}
	return cmd, err
}

// WriteSupernodeFileServer writes a file to the supernode file server.
func (s *Starter) WriteSupernodeFileServer(filePath string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(fp.Join(s.supernodeFileServerHome, filePath), data, perm)
}

// DFDaemon starts dfdaemon.
func (s *Starter) DFDaemon(running time.Duration, port int) (*exec.Cmd, error) {
	return s.execCmd(running, dfdaemonPath,
		"--port", fmt.Sprintf("%d", port))
}

// DFGet starts dfget.
func (s *Starter) DFGet(running time.Duration, args ...string) (*exec.Cmd, error) {
	args = append([]string{"--verbose"}, args...)
	return s.execCmd(running, dfgetPath, args...)
}

// DFGetServer starts dfget as a peer server.
func (s *Starter) DFGetServer(running time.Duration, args ...string) (*exec.Cmd, error) {
	args = append([]string{"server", "--ip", "127.0.0.1", "--verbose",
		"--home", s.Home, "--data", fp.Join(s.Home, "data")},
		args...)
	return s.execCmd(running, dfgetPath, args...)
}

// Kill shutdown one process.
func (s *Starter) Kill(cmd *exec.Cmd) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.kill(cmd)
}

// KillAll shutdown all running processes.
func (s *Starter) KillAll() {
	s.lock.Lock()
	defer s.lock.Unlock()
	for s.cmdList.Len() > 0 {
		cmd := s.cmdList.Front()
		s.cmdList.Remove(cmd)
		if cmd == nil || cmd.Value == nil {
			continue
		}
		if v, ok := cmd.Value.(*exec.Cmd); ok {
			s.kill(v)
		}
	}
}

// Clean cleans all temporary directories and processes.
func (s *Starter) Clean() {
	s.KillAll()
	if s.Home != "" {
		os.RemoveAll(s.Home)
	}
}

// kill can not be called directly, use Kill instead.
func (s *Starter) kill(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if cmd.ProcessState == nil {
		cmd.Process.Signal(os.Interrupt)
	}
	if v, ok := s.listMap[cmd]; ok {
		s.cmdList.Remove(v)
		delete(s.listMap, cmd)
	}
	if v, ok := s.fileSrv[cmd]; ok {
		v.Shutdown(context.Background())
		delete(s.fileSrv, cmd)
	}
}

// execCmd executes a command.
// param running indicates that how much time the process can run at most,
// after the running duration, the process will be killed automatically.
// When the value of running is less than 0, it will not be killed automatically,
// the caller should take the response to release it.
func (s *Starter) execCmd(running time.Duration, name string, args ...string) (
	cmd *exec.Cmd, err error) {
	cmd = exec.Command(name, args...)
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	s.addCmd(cmd)
	if running > 0 {
		time.AfterFunc(running, func() { s.Kill(cmd) })
	}
	return
}

func (s *Starter) addCmd(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	el := s.cmdList.PushBack(cmd)
	s.listMap[cmd] = el
}

func (s *Starter) fileServer(cmd *exec.Cmd, root string, port int) (*http.Server, error) {
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: http.FileServer(http.Dir(root)),
	}
	err := make(chan error)
	go func() {
		if e := server.ListenAndServe(); err != nil {
			err <- e
		}
	}()

	select {
	case e := <-err:
		return nil, e
	case <-time.After(100 * time.Millisecond):
		s.lock.Lock()
		s.fileSrv[cmd] = server
		s.lock.Unlock()
		return server, nil
	}
}

func check(ip string, port int, timeout time.Duration) (err error) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	end := make(chan error)
	time.AfterFunc(timeout, func() {
		end <- fmt.Errorf("wait timeout:%v", timeout)
	})

	for {
		select {
		case err = <-end:
			return err
		case <-ticker.C:
			if _, err = httputils.CheckConnect(ip, port, 50); err == nil {
				return nil
			}
		}
	}
}
