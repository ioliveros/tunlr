package repository_test

import (
	"path/filepath"
	"testing"

	"github.com/ioliveros/tunlr/internal/config"
	"github.com/ioliveros/tunlr/internal/db"
	"github.com/ioliveros/tunlr/internal/model"
	"github.com/ioliveros/tunlr/internal/repository"
)

func newTestRepo(t *testing.T) *repository.HostRepository {
	t.Helper()
	t.Setenv("DB_PATH", filepath.Join(t.TempDir(), "test.db"))
	database := db.Connect(config.Load())
	return repository.NewHostRepository(database)
}

func TestHostAndForwardCRUD(t *testing.T) {
	repo := newTestRepo(t)

	host := &model.Host{Name: "Local", Hostname: "bastion.example.com", User: "dev", Port: 22, AuthMethod: model.AuthAgent}
	if err := repo.CreateHost(host); err != nil {
		t.Fatalf("create host: %v", err)
	}
	if host.ID == 0 {
		t.Fatal("expected host ID to be set")
	}

	fwd := &model.Forward{HostID: host.ID, Label: "pg", LocalPort: 5432, RemoteHost: "10.0.0.1", RemotePort: 5432, Enabled: true}
	if err := repo.CreateForward(fwd); err != nil {
		t.Fatalf("create forward: %v", err)
	}

	got, err := repo.GetHost(host.ID)
	if err != nil {
		t.Fatalf("get host: %v", err)
	}
	if len(got.Forwards) != 1 {
		t.Fatalf("expected 1 forward, got %d", len(got.Forwards))
	}

	// Deleting the host cascades to its forwards.
	if err := repo.DeleteHost(host.ID); err != nil {
		t.Fatalf("delete host: %v", err)
	}
	if _, err := repo.GetHost(host.ID); err == nil {
		t.Fatal("expected error fetching deleted host")
	}
}
