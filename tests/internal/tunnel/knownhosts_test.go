package tunnel_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ioliveros/tunlr/internal/model"
	"github.com/ioliveros/tunlr/internal/tunnel"
	"golang.org/x/crypto/ssh"
)

func TestKnownHostsPath_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	got, err := tunnel.KnownHostsPath()
	if err != nil {
		t.Fatalf("KnownHostsPath: %v", err)
	}

	want := filepath.Join(tmpDir, ".ssh", "known_hosts")
	if got != want {
		t.Errorf("KnownHostsPath() = %q, want %q", got, want)
	}
	if _, err := os.Stat(got); os.IsNotExist(err) {
		t.Errorf("known_hosts file not created at %q", got)
	}
}

func TestAppendKnownHost(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}

	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:22")
	if err != nil {
		t.Fatalf("resolve addr: %v", err)
	}

	if err := tunnel.AppendKnownHost(knownHostsFile, "myhost", addr, signer.PublicKey()); err != nil {
		t.Fatalf("AppendKnownHost: %v", err)
	}

	contents, err := os.ReadFile(knownHostsFile)
	if err != nil {
		t.Fatalf("read known_hosts: %v", err)
	}
	if !strings.Contains(string(contents), "myhost") {
		t.Errorf("known_hosts does not contain %q:\n%s", "myhost", string(contents))
	}
}

func TestHostKeyCallback_AcceptNew(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate server key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}

	cb, err := tunnel.HostKeyCallback(model.HostKeyAcceptNew)
	if err != nil {
		t.Fatalf("HostKeyCallback: %v", err)
	}

	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:22")
	if err != nil {
		t.Fatalf("resolve addr: %v", err)
	}

	// knownhosts requires hostname in host:port format.
	const hostname = "testserver:22"

	// First call: unknown host — TOFU should accept it.
	if err := cb(hostname, addr, signer.PublicKey()); err != nil {
		t.Errorf("first TOFU call returned error: %v", err)
	}

	// Reload the callback so it picks up the newly written known_hosts entry.
	cb2, err := tunnel.HostKeyCallback(model.HostKeyAcceptNew)
	if err != nil {
		t.Fatalf("HostKeyCallback (2nd): %v", err)
	}

	// Second call: same key for same host — should be accepted.
	if err := cb2(hostname, addr, signer.PublicKey()); err != nil {
		t.Errorf("second call with same key returned error: %v", err)
	}
}
