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

// noopEngine satisfies service.Engine without touching the network.
type noopEngine struct{}

func (noopEngine) Apply(model.Host)        {}
func (noopEngine) StopHost(uint)           {}
func (noopEngine) StopForward(uint, uint)  {}
func (noopEngine) ReconnectHost(uint)      {}
func (noopEngine) Snapshot() tunnel.Status { return tunnel.Status{} }

// trackingEngine records calls so tests can assert on engine interactions.
type trackingEngine struct {
	applied     []model.Host
	stopped     []uint
	reconnected []uint
}

func (e *trackingEngine) Apply(h model.Host)      { e.applied = append(e.applied, h) }
func (e *trackingEngine) StopHost(id uint)        { e.stopped = append(e.stopped, id) }
func (e *trackingEngine) StopForward(_, _ uint)   {}
func (e *trackingEngine) ReconnectHost(id uint)   { e.reconnected = append(e.reconnected, id) }
func (e *trackingEngine) Snapshot() tunnel.Status { return tunnel.Status{} }

func newRepo(t *testing.T) *repository.HostRepository {
	t.Helper()
	t.Setenv("DB_PATH", filepath.Join(t.TempDir(), "test.db"))
	return repository.NewHostRepository(db.Connect(config.Load()))
}

func newService(t *testing.T) *service.TunnelService {
	t.Helper()
	return service.NewTunnelService(newRepo(t), noopEngine{})
}

func newServiceTracked(t *testing.T) (*service.TunnelService, *trackingEngine) {
	t.Helper()
	eng := &trackingEngine{}
	return service.NewTunnelService(newRepo(t), eng), eng
}

// --- AddConnection / grouping ---

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

func TestAddConnectionRequiresDomain(t *testing.T) {
	svc := newService(t)
	if _, err := svc.AddConnection(dto.ConnectionInput{Host: "10.0.0.1", RemotePort: 5432, LocalPort: 5432}); err == nil {
		t.Fatal("expected error when domain is empty")
	}
}

func TestAddConnectionWithCustomPort(t *testing.T) {
	svc := newService(t)

	host, err := svc.AddConnection(dto.ConnectionInput{
		ConnectionName: "db",
		Host:           "10.0.0.1",
		RemotePort:     5432,
		LocalPort:      5432,
		Domain:         "user@bastion.example.com:2222",
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if host.Port != 2222 {
		t.Fatalf("expected port 2222, got %d", host.Port)
	}
	if host.User != "user" || host.Hostname != "bastion.example.com" {
		t.Fatalf("unexpected host: %+v", host)
	}
}

func TestAddConnectionDefaultsToCurrentUser(t *testing.T) {
	svc := newService(t)

	host, err := svc.AddConnection(dto.ConnectionInput{
		ConnectionName: "db",
		Host:           "10.0.0.1",
		RemotePort:     5432,
		LocalPort:      5432,
		Domain:         "bastion.example.com",
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if host.User == "" {
		t.Fatal("expected non-empty user from current username fallback")
	}
}

func TestAddConnectionWithKeyPath(t *testing.T) {
	svc := newService(t)

	host, err := svc.AddConnection(dto.ConnectionInput{
		ConnectionName: "db",
		Host:           "10.0.0.1",
		RemotePort:     5432,
		LocalPort:      5432,
		Domain:         "u@bastion",
		KeyPath:        "/tmp/id_rsa",
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if host.AuthMethod != model.AuthKey || host.KeyPath != "/tmp/id_rsa" {
		t.Fatalf("expected AuthKey + KeyPath, got: %+v", host)
	}
}

func TestAddConnectionUpdatesKeyOnExistingHost(t *testing.T) {
	svc, eng := newServiceTracked(t)

	_, err := svc.AddConnection(dto.ConnectionInput{ConnectionName: "a", Host: "10.0.0.1", RemotePort: 1, LocalPort: 1, Domain: "u@bastion"})
	if err != nil {
		t.Fatalf("first add: %v", err)
	}
	eng.stopped = nil
	eng.applied = nil

	host, err := svc.AddConnection(dto.ConnectionInput{ConnectionName: "b", Host: "10.0.0.2", RemotePort: 2, LocalPort: 2, Domain: "u@bastion", KeyPath: "/tmp/key"})
	if err != nil {
		t.Fatalf("second add: %v", err)
	}
	if host.AuthMethod != model.AuthKey || host.KeyPath != "/tmp/key" {
		t.Fatalf("key not applied to existing host: %+v", host)
	}
	if len(eng.stopped) != 1 {
		t.Fatalf("expected StopHost for key change, got %d stops", len(eng.stopped))
	}
}

// --- DeleteForward ---

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

// --- StartAll ---

func TestStartAll(t *testing.T) {
	svc, eng := newServiceTracked(t)

	_, err := svc.AddConnection(dto.ConnectionInput{ConnectionName: "a", Host: "10.0.0.1", RemotePort: 1, LocalPort: 1, Domain: "u@h"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	eng.applied = nil

	if err := svc.StartAll(); err != nil {
		t.Fatalf("StartAll: %v", err)
	}
	if len(eng.applied) != 1 {
		t.Fatalf("expected 1 Apply call from StartAll, got %d", len(eng.applied))
	}
}

// --- Status ---

func TestStatus(t *testing.T) {
	svc := newService(t)
	_ = svc.Status() // just verify no panic; noop engine returns zero value
}

// --- GetHost / CreateHost / UpdateHost / DeleteHost ---

func TestGetHost(t *testing.T) {
	svc := newService(t)

	host, err := svc.AddConnection(dto.ConnectionInput{ConnectionName: "x", Host: "1.2.3.4", RemotePort: 80, LocalPort: 8080, Domain: "u@host"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	got, err := svc.GetHost(host.ID)
	if err != nil {
		t.Fatalf("GetHost: %v", err)
	}
	if got.ID != host.ID {
		t.Fatalf("expected id %d, got %d", host.ID, got.ID)
	}
}

func TestCreateAndUpdateHost(t *testing.T) {
	svc := newService(t)

	h, err := svc.CreateHost(model.Host{Name: "myhost", Hostname: "myhost.example.com", User: "admin", Port: 22})
	if err != nil {
		t.Fatalf("CreateHost: %v", err)
	}
	if h.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	h.Name = "renamed"
	updated, err := svc.UpdateHost(*h)
	if err != nil {
		t.Fatalf("UpdateHost: %v", err)
	}
	if updated.Name != "renamed" {
		t.Fatalf("expected name 'renamed', got %q", updated.Name)
	}
}

func TestDeleteHost(t *testing.T) {
	svc, eng := newServiceTracked(t)

	h, err := svc.CreateHost(model.Host{Name: "tmp", Hostname: "tmp.example.com", User: "u", Port: 22})
	if err != nil {
		t.Fatalf("CreateHost: %v", err)
	}
	eng.stopped = nil

	if err := svc.DeleteHost(h.ID); err != nil {
		t.Fatalf("DeleteHost: %v", err)
	}
	if len(eng.stopped) != 1 || eng.stopped[0] != h.ID {
		t.Fatalf("expected StopHost(%d), got %v", h.ID, eng.stopped)
	}
	hosts, _ := svc.ListHosts()
	if len(hosts) != 0 {
		t.Fatalf("expected 0 hosts after delete, got %d", len(hosts))
	}
}

// --- AddForward / UpdateForward ---

func TestAddAndUpdateForward(t *testing.T) {
	svc := newService(t)

	h, err := svc.CreateHost(model.Host{Name: "h", Hostname: "h.example.com", User: "u", Port: 22})
	if err != nil {
		t.Fatalf("CreateHost: %v", err)
	}

	fwd, err := svc.AddForward(model.Forward{HostID: h.ID, Label: "db", RemoteHost: "10.0.0.1", RemotePort: 5432, LocalPort: 5432, Enabled: true})
	if err != nil {
		t.Fatalf("AddForward: %v", err)
	}
	if fwd.ID == 0 {
		t.Fatal("expected non-zero forward ID")
	}

	fwd.Label = "renamed-db"
	updated, err := svc.UpdateForward(*fwd)
	if err != nil {
		t.Fatalf("UpdateForward: %v", err)
	}
	if updated.Label != "renamed-db" {
		t.Fatalf("expected label 'renamed-db', got %q", updated.Label)
	}
}

// --- SetHostKey ---

func TestSetHostKey(t *testing.T) {
	svc, eng := newServiceTracked(t)

	host, err := svc.AddConnection(dto.ConnectionInput{ConnectionName: "db", Host: "10.0.0.1", RemotePort: 5432, LocalPort: 5432, Domain: "u@bastion"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	eng.stopped = nil
	eng.applied = nil

	updated, err := svc.SetHostKey(host.ID, "/home/user/.ssh/id_rsa")
	if err != nil {
		t.Fatalf("SetHostKey: %v", err)
	}
	if updated.KeyPath != "/home/user/.ssh/id_rsa" || updated.AuthMethod != model.AuthKey {
		t.Fatalf("key not set: %+v", updated)
	}
	if len(eng.stopped) != 1 {
		t.Fatalf("expected StopHost call, got %d", len(eng.stopped))
	}
	if len(eng.applied) != 1 {
		t.Fatalf("expected Apply call, got %d", len(eng.applied))
	}
}

func TestSetHostKeyClear(t *testing.T) {
	svc := newService(t)

	host, err := svc.AddConnection(dto.ConnectionInput{ConnectionName: "db", Host: "10.0.0.1", RemotePort: 5432, LocalPort: 5432, Domain: "u@bastion", KeyPath: "/tmp/key"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	updated, err := svc.SetHostKey(host.ID, "")
	if err != nil {
		t.Fatalf("SetHostKey clear: %v", err)
	}
	if updated.KeyPath != "" || updated.AuthMethod != model.AuthAgent {
		t.Fatalf("key not cleared: %+v", updated)
	}
}

// --- ReconnectHost ---

func TestReconnectHost(t *testing.T) {
	svc, eng := newServiceTracked(t)

	host, err := svc.AddConnection(dto.ConnectionInput{ConnectionName: "db", Host: "10.0.0.1", RemotePort: 5432, LocalPort: 5432, Domain: "u@bastion"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	eng.reconnected = nil

	svc.ReconnectHost(host.ID)
	if len(eng.reconnected) != 1 || eng.reconnected[0] != host.ID {
		t.Fatalf("expected ReconnectHost(%d), got %v", host.ID, eng.reconnected)
	}
}

// --- ListSSHKeys ---

func TestListSSHKeys(t *testing.T) {
	svc := newService(t)
	keys := svc.ListSSHKeys()
	if keys == nil {
		t.Fatal("expected non-nil slice")
	}
}
