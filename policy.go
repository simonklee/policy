// Copyright 2014 Simon Zimmermann. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// pkg policy implements a TCP socket server which handles policy requests
// issues by Unity3D web players.
//
// https://docs.unity3d.com/Documentation/Manual/SecuritySandbox.html

package policy

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/simonz05/util/log"
	"github.com/simonz05/util/sig"
)

const (
	// readBufSize is scaled to fit the largest policy protocol request
	bufSize = 8 << 2
	// max bytes allocated = bufSize * poolSize
	poolSize = 8 << 11
)

var (
	protocolPolicy         = []byte("<policy-file-request/>\x00")
	protocolPolicyResponse = []byte(`<?xml version="1.0"?>
<cross-domain-policy>
   <allow-access-from domain="*" to-ports="*"/> 
</cross-domain-policy>`)
	protocolPing         = []byte("PING")
	protocolPingResponse = []byte("+OK\r\n")
	Timeout              = time.Second * 10
)

func ListenAndServe(laddr string) error {
	l, err := net.Listen("tcp", laddr)

	if err != nil {
		return err
	}

	log.Printf("Listen on %s", l.Addr())
	sig.TrapCloser(l)
	err = serve(l)
	log.Printf("Shutting down ..")
	return err
}

func serve(l net.Listener) error {
	defer l.Close()
	for {
		c, err := l.Accept()

		if err != nil {
			return err
		}

		go handle(c)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()
	buf := getBuf(bufSize)
	defer putBuf(buf)

	err := conn.SetDeadline(time.Now().Add(Timeout))

	if err != nil {
		log.Errorf("Error setting deadline on conn")
		return
	}

	n, err := conn.Read(buf)

	if err != nil {
		log.Errorf("Error reading from conn")
		return
	}

	log.Printf("Got %+q", buf[:n])
	resp, err := parseRequest(buf[:n])

	if err != nil {
		log.Error(err)
		return
	}

	n, err = conn.Write(resp)

	if err != nil || n != len(resp) {
		log.Errorf("Error writing to conn")
		return
	}
}

func parseRequest(buf []byte) (resp []byte, err error) {
	switch {
	case bytes.Equal(protocolPolicy, buf):
		log.Printf("Policy request")
		resp = protocolPolicyResponse
	case bytes.Equal(protocolPing, buf):
		log.Printf("Ping request")
		resp = protocolPingResponse
	default:
		err = fmt.Errorf("Uknown protocol request: %+q", buf)
	}

	return
}

var bufPool = make(chan []byte, poolSize)

func getBuf(size int) []byte {
	for {
		select {
		case b := <-bufPool:
			if cap(b) >= size {
				return b[:size]
			}
		default:
			return make([]byte, size)
		}
	}
}

func putBuf(b []byte) {
	select {
	case bufPool <- b:
	default:
	}
}
