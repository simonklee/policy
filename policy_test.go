package policy

import (
	"bytes"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/simonz05/util/assert"
	"github.com/simonz05/util/log"
)

func init() {
	log.Severity = log.LevelInfo
	Timeout = time.Second
}

var policyTests = []struct {
	send      []byte
	exp       []byte
	failWrite bool
	failRead  bool
}{
	{
		send: protocolPing,
		exp:  protocolPingResponse,
	},
	{
		send:     []byte(""),
		exp:      []byte(""),
		failRead: true,
	},
	{
		send: protocolPolicy,
		exp:  protocolPolicyResponse,
	},
	{
		send:     protocolPolicy[:len(protocolPolicy)-1],
		exp:      protocolPolicyResponse,
		failRead: true,
	},
}

const testBufSize = 8 << 4

func TestPolicy(t *testing.T) {
	ast := assert.NewAssert(t)

	var wg sync.WaitGroup
	l, err := net.Listen("tcp", ":9001")
	defer l.Close()

	ast.Nil(err)
	log.Printf("Listen on %v", l.Addr())

	wg.Add(1)
	go func() {
		serve(l)
		wg.Done()
	}()

	for _, p := range policyTests {
		conn, err := net.Dial("tcp", ":9001")
		ast.Nil(err)

		n, err := conn.Write(p.send)

		if !p.failWrite {
			ast.Nil(err)
			ast.Equal(len(p.send), n)
		} else {
			ast.NotNil(err)
			continue
		}

		buf := make([]byte, testBufSize)
		n, err = conn.Read(buf)

		if !p.failRead {
			ast.Nil(err)
			ast.Equal(len(p.exp), n)
			ast.True(bytes.Equal(p.exp, buf[:n]))
		} else {
			ast.NotNil(err)
			continue
		}
	}
	l.Close()
	wg.Wait()
}

func BenchmarkPolicy(b *testing.B) {
	b.StopTimer()
	var wg sync.WaitGroup
	l, err := net.Listen("tcp", ":9001")
	defer l.Close()

	if err != nil {
		b.Fatal(err)
	}

	log.Printf("Listen on %v", l.Addr())

	wg.Add(1)
	go func() {
		serve(l)
		wg.Done()
	}()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		conn, err := net.Dial("tcp", ":9001")

		if err != nil {
			b.Errorf("error dialing: %v", err)
		}

		n, err := conn.Write(protocolPolicy)

		if err != nil || n != len(protocolPolicy) {
			b.Errorf("error sending: %v, %d == %d", err, n, len(protocolPolicy))
		}

		buf := make([]byte, testBufSize)
		n, err = conn.Read(buf)

		if err != nil || n != len(protocolPolicyResponse) {
			b.Errorf("error reading: %v, %d == %d", err, n, len(protocolPolicyResponse))
		}

		if !bytes.Equal(protocolPolicyResponse, buf[:n]) {
			b.Errorf("unexpected response value, exp: %+q, got %+q", protocolPolicyResponse, buf[:n])
		}
	}
	b.StopTimer()
	l.Close()
	wg.Wait()
	b.StartTimer()
}
