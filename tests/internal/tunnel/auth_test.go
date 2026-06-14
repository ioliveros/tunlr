package tunnel_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/ioliveros/tunlr/internal/model"
	"github.com/ioliveros/tunlr/internal/tunnel"
)

func TestExpandPath_WithTilde(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	got := tunnel.ExpandPath("~/foo")
	want := filepath.Join(tmpDir, "foo")
	if got != want {
		t.Errorf("ExpandPath(~/foo) = %q, want %q", got, want)
	}
}

func TestExpandPath_Absolute(t *testing.T) {
	got := tunnel.ExpandPath("/tmp/key")
	if got != "/tmp/key" {
		t.Errorf("ExpandPath(/tmp/key) = %q, want /tmp/key", got)
	}
}

// writeED25519Key generates an ed25519 key, writes it as PKCS8 PEM to path,
// and returns the path.
func writeED25519Key(t *testing.T, path string) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	if err := os.WriteFile(path, keyPEM, 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
}

func TestLoadKey_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_ed25519")
	writeED25519Key(t, keyPath)

	signer, err := tunnel.LoadKey(keyPath)
	if err != nil {
		t.Fatalf("LoadKey: %v", err)
	}
	if signer == nil {
		t.Error("expected non-nil signer")
	}
}

func TestLoadKey_NotFound(t *testing.T) {
	_, err := tunnel.LoadKey("/nonexistent/path/key")
	if err == nil {
		t.Error("expected error for non-existent key path, got nil")
	}
}

func TestAgentAuth_NoSocket(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	got := tunnel.AgentAuth()
	if got != nil {
		t.Errorf("AgentAuth() = %v, want nil when SSH_AUTH_SOCK is empty", got)
	}
}

func TestAuthMethods_WithKeyFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("SSH_AUTH_SOCK", "")

	keyPath := filepath.Join(tmpDir, "id_ed25519")
	writeED25519Key(t, keyPath)

	methods := tunnel.AuthMethods(model.Host{
		AuthMethod: model.AuthKey,
		KeyPath:    keyPath,
	})
	if len(methods) == 0 {
		t.Error("expected at least one auth method, got none")
	}
}

func TestAuthMethods_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("SSH_AUTH_SOCK", "")

	// Empty home dir: no keys, no agent.
	methods := tunnel.AuthMethods(model.Host{AuthMethod: model.AuthAgent})
	if len(methods) != 0 {
		t.Errorf("expected nil/empty auth methods, got %d", len(methods))
	}
}

func TestDefaultSigners_WithKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	sshDir := filepath.Join(tmpDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatalf("mkdir .ssh: %v", err)
	}
	writeED25519Key(t, filepath.Join(sshDir, "id_ed25519"))

	signers := tunnel.DefaultSigners()
	if len(signers) == 0 {
		t.Error("expected at least one signer, got none")
	}
}

func TestDefaultSigners_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	signers := tunnel.DefaultSigners()
	if len(signers) != 0 {
		t.Errorf("expected empty signers, got %d", len(signers))
	}
}
