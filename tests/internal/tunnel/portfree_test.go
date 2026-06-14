package tunnel_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/ioliveros/tunlr/internal/tunnel"
)

// FreeLocalPort must never kill the current process, even though tunlr's own
// listeners show up in lsof on the same port.
func TestFreeLocalPortSkipsSelf(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	tunnel.FreeLocalPort(port) // owned by this test process; must be left alone

	c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("self listener was killed: %v", err)
	}
	c.Close()
}
