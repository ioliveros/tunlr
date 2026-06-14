package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/ioliveros/tunlr/internal/config"
	"github.com/ioliveros/tunlr/internal/db"
	"github.com/ioliveros/tunlr/internal/dto"
	"github.com/ioliveros/tunlr/internal/model"
	"github.com/ioliveros/tunlr/internal/repository"
	"github.com/ioliveros/tunlr/internal/service"
	"github.com/ioliveros/tunlr/internal/tunnel"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// statusEvent is the Wails event the frontend subscribes to for live status.
const statusEvent = "tunnel:status"

// App is the root struct bound to the frontend. Its exported methods are
// callable from TypeScript via the generated Wails bindings.
type App struct {
	ctx     context.Context
	tunnels *service.TunnelService
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. It opens the database, seeds the
// default tunnels on first run, and wires up the service layer.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	cfg := config.Load()
	database := db.Connect(cfg)
	repo := repository.NewHostRepository(database)

	engine := tunnel.NewManager(func(s tunnel.Status) {
		wailsruntime.EventsEmit(a.ctx, statusEvent, s)
	})
	a.tunnels = service.NewTunnelService(repo, engine)

	if err := a.tunnels.StartAll(); err != nil {
		log.Printf("starting tunnels failed: %v", err)
	}
}

// --- Host configuration (bound to the frontend) ---

func (a *App) ListHosts() ([]model.Host, error) { return a.tunnels.ListHosts() }

func (a *App) GetHost(id uint) (*model.Host, error) { return a.tunnels.GetHost(id) }

func (a *App) CreateHost(host model.Host) (*model.Host, error) { return a.tunnels.CreateHost(host) }

func (a *App) UpdateHost(host model.Host) (*model.Host, error) { return a.tunnels.UpdateHost(host) }

func (a *App) DeleteHost(id uint) error { return a.tunnels.DeleteHost(id) }

func (a *App) AddConnection(in dto.ConnectionInput) (*model.Host, error) {
	return a.tunnels.AddConnection(in)
}

func (a *App) AddForward(f model.Forward) (*model.Forward, error) { return a.tunnels.AddForward(f) }

func (a *App) UpdateForward(f model.Forward) (*model.Forward, error) {
	return a.tunnels.UpdateForward(f)
}

func (a *App) DeleteForward(id uint) error { return a.tunnels.DeleteForward(id) }

// GetStatus returns the current live status of all hosts and forwards. The
// frontend also receives pushed updates via the "tunnel:status" event.
func (a *App) GetStatus() tunnel.Status { return a.tunnels.Status() }

// ReconnectHost resets the retry counter and restarts the connection loop for
// a host that has given up after exhausting its retries.
func (a *App) ReconnectHost(id uint) { a.tunnels.ReconnectHost(id) }

// ListSSHKeys returns the SSH private keys available in ~/.ssh.
func (a *App) ListSSHKeys() []dto.SSHKey { return a.tunnels.ListSSHKeys() }

// SetHostKey pins (or clears, when keyPath is empty) the SSH key for a host and
// reconnects it.
func (a *App) SetHostKey(hostID uint, keyPath string) (*model.Host, error) {
	return a.tunnels.SetHostKey(hostID, keyPath)
}

// PickSSHKey opens a native file dialog (defaulting to ~/.ssh) for the user to
// choose a private key. Returns an empty string if the dialog is cancelled.
func (a *App) PickSSHKey() (string, error) {
	dir := ""
	if home, err := os.UserHomeDir(); err == nil {
		dir = filepath.Join(home, ".ssh")
	}
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:                "Select SSH private key",
		DefaultDirectory:     dir,
		ShowHiddenFiles:      true,
		CanCreateDirectories: false,
	})
}
