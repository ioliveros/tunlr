package main

import (
	"context"

	"github.com/ioliveros/tunlr/internal/config"
	"github.com/ioliveros/tunlr/internal/db"
	"github.com/ioliveros/tunlr/internal/dto"
	"github.com/ioliveros/tunlr/internal/model"
	"github.com/ioliveros/tunlr/internal/repository"
	"github.com/ioliveros/tunlr/internal/service"
)

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

	a.tunnels = service.NewTunnelService(repo)
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
