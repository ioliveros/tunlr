package service_test

import (
	"path/filepath"
	"testing"

	"github.com/ioliveros/tunlr/internal/config"
	"github.com/ioliveros/tunlr/internal/db"
	"github.com/ioliveros/tunlr/internal/dto"
	"github.com/ioliveros/tunlr/internal/model"
	"github.com/ioliveros/tunlr/internal/repository"
	"github.com/ioliveros/tunlr/internal/service"
	"github.com/ioliveros/tunlr/internal/tunnel"
)

// noopEngine satisfies service.Engine without touching the network, so the
// persistence behaviour can be tested without dialing SSH.
type noopEngine struct{}

func (noopEngine) Apply(model.Host)        {}
func (noopEngine) StopHost(uint)           {}
func (noopEngine) StopForward(uint, uint)  {}
func (noopEngine) ReconnectHost(uint)      {}
func (noopEngine) Snapshot() tunnel.Status { return tunnel.Status{} }

func newService(t *testing.T) *service.TunnelService {
	t.Helper()
	t.Setenv("DB_PATH", filepath.Join(t.TempDir(), "test.db"))
	database := db.Connect(config.Load())
	return service.NewTunnelService(repository.NewHostRepository(database), noopEngine{})
}

func TestAddConnectionParsesDomainAndGroupsByHost(t *testing.T) {
	svc := newService(t)

	first, err := svc.AddConnection(dto.ConnectionInput{
		ConnectionName: "TimescaleDB",
		Host:           "10.10.10.10",
		RemotePort:     5432,
		LocalPort:      5437,
		Domain:         "dev@me.ioliveros.dev",
	})
	if err != nil {
		t.Fatalf("add first connection: %v", err)
	}
	if first.User != "dev" || first.Hostname != "me.ioliveros.dev" || first.Port != 22 {
		t.Fatalf("domain parsed wrong: %+v", first)
	}
	if len(first.Forwards) != 1 {
		t.Fatalf("expected 1 forward, got %d", len(first.Forwards))
	}

	// A second connection through the same domain reuses the same host.
	second, err := svc.AddConnection(dto.ConnectionInput{
		ConnectionName: "Neo4j",
		Host:           "10.10.10.11",
		RemotePort:     7687,
		LocalPort:      7687,
		Domain:         "dev@me.ioliveros.dev",
	})
	if err != nil {
		t.Fatalf("add second connection: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected same host id %d, got %d", first.ID, second.ID)
	}

	hosts, err := svc.ListHosts()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host after grouping, got %d", len(hosts))
	}
	if len(hosts[0].Forwards) != 2 {
		t.Fatalf("expected 2 forwards under host, got %d", len(hosts[0].Forwards))
	}
}

func TestDeleteForwardRemovesEmptyHost(t *testing.T) {
	svc := newService(t)

	host, err := svc.AddConnection(dto.ConnectionInput{
		ConnectionName: "only",
		Host:           "10.0.0.1",
		RemotePort:     5432,
		LocalPort:      5432,
		Domain:         "dev@me.ioliveros.dev",
	})
	if err != nil {
		t.Fatalf("add connection: %v", err)
	}

	if err := svc.DeleteForward(host.Forwards[0].ID); err != nil {
		t.Fatalf("delete forward: %v", err)
	}

	hosts, err := svc.ListHosts()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(hosts) != 0 {
		t.Fatalf("expected host removed after its last forward, got %d hosts", len(hosts))
	}
}

func TestDeleteForwardKeepsHostWithRemainingForwards(t *testing.T) {
	svc := newService(t)

	_, _ = svc.AddConnection(dto.ConnectionInput{ConnectionName: "a", Host: "10.0.0.1", RemotePort: 1, LocalPort: 1, Domain: "gcp@h"})
	host, err := svc.AddConnection(dto.ConnectionInput{ConnectionName: "b", Host: "10.0.0.2", RemotePort: 2, LocalPort: 2, Domain: "gcp@h"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	if err := svc.DeleteForward(host.Forwards[0].ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	hosts, err := svc.ListHosts()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(hosts) != 1 || len(hosts[0].Forwards) != 1 {
		t.Fatalf("expected host kept with 1 forward, got %d hosts", len(hosts))
	}
}

func TestAddConnectionRequiresDomain(t *testing.T) {
	svc := newService(t)
	if _, err := svc.AddConnection(dto.ConnectionInput{Host: "10.0.0.1", RemotePort: 5432, LocalPort: 5432}); err == nil {
		t.Fatal("expected error when domain is empty")
	}
}
