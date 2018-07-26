// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
)

// StartProcess starts the process
func (engine *EngineOperations) StartProcess(masterConn net.Conn) error {
	// move to the container rootfs /
	os.Chdir("/")

	args := engine.CommonConfig.OciConfig.Process.Args
	env := engine.CommonConfig.OciConfig.Process.Env

	err := syscall.Exec(args[0], args, env)
	if err != nil {
		return fmt.Errorf("exec %s failed: %s", args[0], err)
	}
	return nil
}

// MonitorContainer monitors a container process
func (engine *EngineOperations) MonitorContainer(pid int) (syscall.WaitStatus, error) {
	var status syscall.WaitStatus

	signals := make(chan os.Signal, 1)
	signal.Notify(signals)

	for {
		s := <-signals
		switch s {
		case syscall.SIGCHLD:
			if wpid, err := syscall.Wait4(pid, &status, syscall.WNOHANG, nil); err != nil {
				return status, fmt.Errorf("error while waiting child: %s", err)
			} else if wpid != pid {
				continue
			}
			return status, nil
		default:
			return status, fmt.Errorf("interrupted by signal %s", s.String())
		}
	}
}

// CleanupContainer cleans up the container
//
//Runtime's delete command is invoked with the unique identifier of the container.
//The container MUST be destroyed by undoing the steps performed during create phase
func (engine *EngineOperations) CleanupContainer() error {
	return nil
}
