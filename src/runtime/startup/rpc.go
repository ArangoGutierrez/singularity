/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package main

import "C"

import (
	"net"
	"os"

	"github.com/singularityware/singularity/src/pkg/sylog"
	"github.com/singularityware/singularity/src/runtime/workflows/rpc"
)

// RPCServer serves runtime engine requests
//export RPCServer
func RPCServer(socket C.int, sruntime *C.char) {
	runtime := C.GoString(sruntime)

	comm := os.NewFile(uintptr(socket), "unix")

	conn, err := net.FileConn(comm)
	if err != nil {
		sylog.Fatalf("socket communication error: %s\n", err)
	}
	comm.Close()

	rpc.ServeRuntimeEngineRequests(runtime, conn)
	os.Exit(0)
}