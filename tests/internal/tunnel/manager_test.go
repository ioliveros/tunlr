package tunnel_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ioliveros/tunlr/internal/model"
	"github.com/ioliveros/tunlr/internal/tunnel"
	"golang.org/x/crypto/ssh"
)

// startTestSSHServer creates a minimal in-process SSH server suitable for
// integration tests. It returns the listening address, a model.Host pre-wired
// to authenticate against that server, and a cleanup function.
func startTestSSHServer(t *testing.T) (addr string, host model.Host, cleanup func()) {
	t.Helper()

	// ---- host (server) key ----
	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPriv)
	if err != nil {
		t.Fatalf("host signer: %v", err)
	}

	// ---- client key ----
	clientPub, clientPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}

	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatalf("mkdir .ssh: %v", err)
	}

	keyPath := filepath.Join(sshDir, "id_ed25519")
	keyBytes, err := x509.MarshalPKCS8PrivateKey(clientPriv)
	if err != nil {
		t.Fatalf("marshal client key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatalf("write client key: %v", err)
	}

	// Point HOME at the temp dir so known_hosts, key discovery all go there.
	t.Setenv("HOME", tmpDir)

	// ---- server config ----
	authorizedKey, err := ssh.NewPublicKey(clientPub)
	if err != nil {
		t.Fatalf("ssh public key: %v", err)
	}

	cfg := &ssh.ServerConfig{
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if string(key.Marshal()) == string(authorizedKey.Marshal()) {
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("unauthorized key")
		},
	}
	cfg.AddHostKey(hostSigner)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed
			}
			go handleSSHConn(conn, cfg)
		}
	}()

	port := ln.Addr().(*net.TCPAddr).Port
	h := model.Host{
		ID:            1,
		Name:          "test-bastion",
		Hostname:      "127.0.0.1",
		User:          "testuser",
		Port:          port,
		AuthMethod:    model.AuthKey,
		KeyPath:       keyPath,
		HostKeyPolicy: model.HostKeyAcceptNew,
		Forwards: []model.Forward{
			{
				ID:         1,
				HostID:     1,
				Label:      "fwd",
				RemoteHost: "127.0.0.1",
				RemotePort: 9999,
				LocalPort:  19999,
				Enabled:    true,
			},
		},
	}

	return ln.Addr().String(), h, func() { ln.Close() }
}

// handleSSHConn performs the server-side SSH handshake and accepts
// direct-tcpip channel requests by immediately closing them.
func handleSSHConn(conn net.Conn, cfg *ssh.ServerConfig) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(reqs)
	for newChan := range chans {
		if newChan.ChannelType() == "direct-tcpip" {
			ch, reqs2, err := newChan.Accept()
			if err != nil {
				continue
			}
			go ssh.DiscardRequests(reqs2)
			ch.Close()
		} else {
			_ = newChan.Reject(ssh.UnknownChannelType, "unsupported")
		}
	}
}

// waitForState polls ch until the named host reaches the desired state or the
// timeout elapses.
func waitForState(t *testing.T, ch <-chan tunnel.Status, hostID string, want tunnel.State, timeout time.Duration) {
	t.Helper()
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case <-deadline.C:
			t.Fatalf("timeout waiting for state %q on host %s", want, hostID)
		case s := <-ch:
			if h, ok := s.Hosts[hostID]; ok && h.State == want {
				return
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Unit tests for exported helpers
// ---------------------------------------------------------------------------

func TestNextBackoff(t *testing.T) {
	if got := tunnel.NextBackoff(2 * time.Second); got != 4*time.Second {
		t.Errorf("NextBackoff(2s) = %v, want 4s", got)
	}
	if got := tunnel.NextBackoff(16 * time.Second); got != 30*time.Second {
		t.Errorf("NextBackoff(16s) = %v, want 30s (max cap)", got)
	}
	if got := tunnel.NextBackoff(30 * time.Second); got != 30*time.Second {
		t.Errorf("NextBackoff(30s) = %v, want 30s", got)
	}
}

func TestPortError(t *testing.T) {
	got := tunnel.PortError(5432, fmt.Errorf("bind failed"))
	want := "local port 5432 unavailable: bind failed"
	if got != want {
		t.Errorf("PortError = %q, want %q", got, want)
	}
}

func TestPipe(t *testing.T) {
	a, b := net.Pipe()

	msg := []byte("hello tunnel")
	done := make(chan struct{})

	go func() {
		defer close(done)
		buf := make([]byte, len(msg))
		if _, err := io.ReadFull(b, buf); err != nil {
			t.Errorf("read from b: %v", err)
			return
		}
		if string(buf) != string(msg) {
			t.Errorf("got %q, want %q", buf, msg)
		}
	}()

	// Write to a; Pipe should copy to b.
	go func() {
		if _, err := a.Write(msg); err != nil {
			// a may be closed already by Pipe; that's OK if data was sent.
		}
	}()

	// Run Pipe — it blocks until one side closes.
	go tunnel.Pipe(a, b)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("TestPipe: timeout")
	}
}

// ---------------------------------------------------------------------------
// Manager lifecycle tests
// ---------------------------------------------------------------------------

func TestNewManager(t *testing.T) {
	// nil emit must not panic
	m := tunnel.NewManager(nil)
	if m == nil {
		t.Fatal("expected non-nil manager")
	}

	// channel-based emit
	ch := make(chan tunnel.Status, 8)
	m2 := tunnel.NewManager(func(s tunnel.Status) { ch <- s })
	if m2 == nil {
		t.Fatal("expected non-nil manager with channel emit")
	}
}

func TestManagerSnapshot_Empty(t *testing.T) {
	m := tunnel.NewManager(nil)
	s := m.Snapshot()
	if s.Hosts == nil {
		t.Error("Snapshot().Hosts should be non-nil")
	}
	if s.Forwards == nil {
		t.Error("Snapshot().Forwards should be non-nil")
	}
	if len(s.Hosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(s.Hosts))
	}
	if len(s.Forwards) != 0 {
		t.Errorf("expected 0 forwards, got %d", len(s.Forwards))
	}
}

func TestManagerStopHost_Noop(t *testing.T) {
	m := tunnel.NewManager(nil)
	m.StopHost(99) // must not panic
}

func TestManagerStopForward_Noop(t *testing.T) {
	m := tunnel.NewManager(nil)
	m.StopForward(99, 1) // must not panic
}

func TestManagerApply_ConnectsAndDisconnects(t *testing.T) {
	// Not parallel — modifies HOME env var via startTestSSHServer.
	_, host, _ := startTestSSHServer(t)

	ch := make(chan tunnel.Status, 32)
	m := tunnel.NewManager(func(s tunnel.Status) { ch <- s })

	m.Apply(host)
	waitForState(t, ch, "1", tunnel.StateConnected, 10*time.Second)

	s := m.Snapshot()
	hs, ok := s.Hosts["1"]
	if !ok {
		t.Fatal("host 1 not in snapshot after connect")
	}
	if hs.State != tunnel.StateConnected {
		t.Errorf("host state = %q, want %q", hs.State, tunnel.StateConnected)
	}

	m.StopHost(1)

	// After StopHost the host should be removed from the snapshot.
	final := m.Snapshot()
	if _, present := final.Hosts["1"]; present {
		t.Error("host 1 still in snapshot after StopHost")
	}
}

func TestManagerReconnectHost(t *testing.T) {
	// Not parallel — modifies HOME env var via startTestSSHServer.
	_, host, _ := startTestSSHServer(t)

	ch := make(chan tunnel.Status, 64)
	m := tunnel.NewManager(func(s tunnel.Status) { ch <- s })

	// First connect.
	m.Apply(host)
	waitForState(t, ch, "1", tunnel.StateConnected, 10*time.Second)

	// Drop the host (simulates StopHost), then re-apply.
	m.StopHost(1)

	// Drain stale events.
	draining := true
	for draining {
		select {
		case <-ch:
		default:
			draining = false
		}
	}

	// Re-apply — simulates reconnect flow.
	m.Apply(host)
	waitForState(t, ch, "1", tunnel.StateConnected, 10*time.Second)
}
