// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package goprocess reports the Go processes running on a host.
package goprocess

import (
	"sync"
	"os/user"

	"github.com/keybase/go-ps"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
)

// P represents a Go process.
type P struct {
	PID              int
	PPID             int
	Exec             string
	Path             string
	Uid              string
	Username         string
	Arguments        string
	ConnectionStates []net.ConnectionStat

	// 把用户名与参数也加到这里来
	// processInfo的结果集成进来，由FindAll统一处理完
}

// FindAll returns all the Go processes currently running on this host.
func FindAll() []P {
	pss, err := ps.Processes()
	if err != nil {
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(len(pss))
	found := make(chan P)

	for _, pr := range pss {
		pr := pr
		go func() {
			defer wg.Done()
			path, err := getPath(pr)
			if err == nil {
				uid, username, argument, connStates, err := getProcessInfo(pr.Pid())
				if err == nil {
					found <- P{
						PID:              pr.Pid(),
						PPID:             pr.PPid(),
						Exec:             pr.Executable(),
						Path:             path,
						Uid:              uid,
						Username:         username,
						Arguments:        argument,
						ConnectionStates: connStates,
					}
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(found)
	}()
	var results []P
	for p := range found {
		results = append(results, p)
	}
	return results
}

// Find finds info about the process identified with the given PID.
func Find(pid int) (p P, err error) {
	pr, err := ps.FindProcess(pid)
	path, err := getPath(pr)
	uid, username, argument, connStates, err := getProcessInfo(pid)
	return P{
		PID:              pr.Pid(),
		PPID:             pr.PPid(),
		Exec:             pr.Executable(),
		Path:             path,
		Uid:              uid,
		Username:         username,
		Arguments:        argument,
		ConnectionStates: connStates,
	}, err
}

// isGo looks up the runtime.buildVersion symbol
// in the process' binary and determines if the process
// if a Go process or not. If the process is a Go process,
// it reports PID, binary name and full path of the binary.
func getPath(pr ps.Process) (path string, err error) {
	path, err = pr.Path()
	return path, err
}

func getProcessInfo(pid int) (string, string, string, []net.ConnectionStat, error) {
	var uid string
	p, err := process.NewProcess(int32(pid))
	username, err := p.Username()
	user, err := user.Lookup(username)
	if err == nil {
		uid = user.Uid
	}
	argument, err := p.Cmdline()
	connStates, err := p.Connections()

	return uid, username, argument, connStates, err
}
