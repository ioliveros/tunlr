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

func seedHost(t *testing.T, r *repository.HostRepository) *model.Host {
	t.Helper()
	h := &model.Host{Name: "bastion", Hostname: "bastion.example.com", User: "user", Port: 22, AuthMethod: model.AuthAgent, HostKeyPolicy: model.HostKeyAcceptNew}
	if err := r.CreateHost(h); err != nil {
		t.Fatalf("CreateHost: %v", err)
	}
	return h
}

func seedForward(t *testing.T, r *repository.HostRepository, hostID uint) *model.Forward {
	t.Helper()
	f := &model.Forward{HostID: hostID, Label: "db", RemoteHost: "10.0.0.1", RemotePort: 5432, LocalPort: 5432, Enabled: true}
	if err := r.CreateForward(f); err != nil {
		t.Fatalf("CreateForward: %v", err)
	}
	return f
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

func TestListHosts(t *testing.T) {
	r := newTestRepo(t)
	seedHost(t, r)
	seedHost(t, r)

	hosts, err := r.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}
}

func TestListHostsPreloadsForwards(t *testing.T) {
	r := newTestRepo(t)
	h := seedHost(t, r)
	seedForward(t, r, h.ID)

	hosts, err := r.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts: %v", err)
	}
	if len(hosts[0].Forwards) != 1 {
		t.Fatalf("expected 1 forward preloaded, got %d", len(hosts[0].Forwards))
	}
}

func TestUpdateHost(t *testing.T) {
	r := newTestRepo(t)
	h := seedHost(t, r)

	h.Name = "updated-name"
	if err := r.UpdateHost(h); err != nil {
		t.Fatalf("UpdateHost: %v", err)
	}
	got, err := r.GetHost(h.ID)
	if err != nil {
		t.Fatalf("GetHost: %v", err)
	}
	if got.Name != "updated-name" {
		t.Fatalf("expected name 'updated-name', got %q", got.Name)
	}
}

func TestCountHosts(t *testing.T) {
	r := newTestRepo(t)

	n, err := r.CountHosts()
	if err != nil {
		t.Fatalf("CountHosts: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}

	seedHost(t, r)
	n, err = r.CountHosts()
	if err != nil {
		t.Fatalf("CountHosts after insert: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1, got %d", n)
	}
}

func TestFindHostFound(t *testing.T) {
	r := newTestRepo(t)
	h := seedHost(t, r)

	found, err := r.FindHost(h.User, h.Hostname, h.Port)
	if err != nil {
		t.Fatalf("FindHost: %v", err)
	}
	if found == nil || found.ID != h.ID {
		t.Fatalf("expected host %d, got %v", h.ID, found)
	}
}

func TestFindHostNotFound(t *testing.T) {
	r := newTestRepo(t)

	found, err := r.FindHost("nobody", "nonexistent.example.com", 22)
	if err != nil {
		t.Fatalf("FindHost: %v", err)
	}
	if found != nil {
		t.Fatalf("expected nil, got %+v", found)
	}
}

func TestGetForward(t *testing.T) {
	r := newTestRepo(t)
	h := seedHost(t, r)
	f := seedForward(t, r, h.ID)

	got, err := r.GetForward(f.ID)
	if err != nil {
		t.Fatalf("GetForward: %v", err)
	}
	if got.ID != f.ID {
		t.Fatalf("expected forward %d, got %d", f.ID, got.ID)
	}
}

func TestCountForwardsForHost(t *testing.T) {
	r := newTestRepo(t)
	h := seedHost(t, r)

	n, err := r.CountForwardsForHost(h.ID)
	if err != nil {
		t.Fatalf("CountForwardsForHost: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}

	seedForward(t, r, h.ID)
	seedForward(t, r, h.ID)
	n, err = r.CountForwardsForHost(h.ID)
	if err != nil {
		t.Fatalf("CountForwardsForHost after inserts: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2, got %d", n)
	}
}

func TestUpdateForward(t *testing.T) {
	r := newTestRepo(t)
	h := seedHost(t, r)
	f := seedForward(t, r, h.ID)

	f.Label = "cache"
	f.LocalPort = 6379
	if err := r.UpdateForward(f); err != nil {
		t.Fatalf("UpdateForward: %v", err)
	}
	got, err := r.GetForward(f.ID)
	if err != nil {
		t.Fatalf("GetForward: %v", err)
	}
	if got.Label != "cache" || got.LocalPort != 6379 {
		t.Fatalf("unexpected forward after update: %+v", got)
	}
}

func TestDeleteForward(t *testing.T) {
	r := newTestRepo(t)
	h := seedHost(t, r)
	f := seedForward(t, r, h.ID)

	if err := r.DeleteForward(f.ID); err != nil {
		t.Fatalf("DeleteForward: %v", err)
	}
	n, _ := r.CountForwardsForHost(h.ID)
	if n != 0 {
		t.Fatalf("expected 0 forwards after delete, got %d", n)
	}
}
