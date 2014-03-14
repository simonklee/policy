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
	"net"
	"time"

	"github.com/simonz05/util/log"
	"github.com/simonz05/util/sig"
)

const BufSize = 8 << 5

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
	buf := getBuf(BufSize)
	defer putBuf(buf)

	err := conn.SetReadDeadline(time.Now().Add(Timeout))
	if err != nil {
		log.Errorf("Error setting read deadline on conn: %v", err)
		return
	}

	n, err := conn.Read(buf)

	if err != nil {
		log.Errorf("Error reading from conn: %v", err)
		return
	}

	log.Printf("Got %+q", buf[:n])
	var resp []byte

	if bytes.Equal(protocolPolicy, buf[:n]) {
		log.Printf("Policy request")
		resp = protocolPolicyResponse
	} else if bytes.Equal(protocolPing, buf[:n]) {
		log.Printf("Ping request")
		resp = protocolPingResponse
	} else {
		log.Errorf("Uknown protocol request: %+q", buf[:n])
		return
	}

	err = conn.SetWriteDeadline(time.Now().Add(Timeout))

	if err != nil {
		log.Errorf("Error setting write deadline on conn: %v", err)
		return
	}

	n, err = conn.Write(resp)

	if err != nil || n != len(resp) {
		log.Errorf("Error writing to conn: %v", err)
		return
	}

	//log.Printf("Wrote %d bytes", n)
}

var bufPool = make(chan []byte, 64)

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
