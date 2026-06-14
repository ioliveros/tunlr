package tunnel

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/ioliveros/tunlr/internal/model"
	"golang.org/x/crypto/ssh"
)

const (
	dialTimeout      = 10 * time.Second
	keepaliveEvery   = 15 * time.Second
	reconnectMinWait = 2 * time.Second
	reconnectMaxWait = 30 * time.Second
	maxRetries       = 3
)

// Manager owns the live SSH tunnels. It is safe for concurrent use and emits a
// fresh Status snapshot whenever anything changes.
type Manager struct {
	mu    sync.Mutex
	hosts map[uint]*hostConn
	emit  func(Status)
}

// NewManager creates a Manager. emit is called (off the caller's goroutine path
// where possible) with a full snapshot on every state change; it may be nil.
func NewManager(emit func(Status)) *Manager {
	if emit == nil {
		emit = func(Status) {}
	}
	return &Manager{hosts: make(map[uint]*hostConn), emit: emit}
}

// Apply ensures host is being managed with its current config and forwards,
// (re)connecting as needed. Safe to call repeatedly.
func (m *Manager) Apply(host model.Host) {
	m.mu.Lock()
	hc, ok := m.hosts[host.ID]
	if ok {
		m.mu.Unlock()
		hc.sync(host)
		m.publish()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	hc = &hostConn{
		mgr:      m,
		host:     host,
		state:    StateConnecting,
		forwards: make(map[uint]*fwdState),
		ctx:      ctx,
		cancel:   cancel,
	}
	for _, f := range host.Forwards {
		if f.Enabled {
			hc.forwards[f.ID] = &fwdState{fwd: f, state: StateConnecting}
		}
	}
	m.hosts[host.ID] = hc
	m.mu.Unlock()

	go hc.run()
	m.publish()
}

// StopHost tears down a host's connection and all its forwards.
func (m *Manager) StopHost(id uint) {
	m.mu.Lock()
	hc := m.hosts[id]
	delete(m.hosts, id)
	m.mu.Unlock()
	if hc != nil {
		hc.close()
	}
	m.publish()
}

// ReconnectHost resets the retry counter and restarts the connection loop for a
// host that has given up. Safe to call on a host in any state.
func (m *Manager) ReconnectHost(id uint) {
	m.mu.Lock()
	hc := m.hosts[id]
	m.mu.Unlock()
	if hc == nil {
		return
	}
	// Cancel the existing goroutine (no-op if already stopped).
	hc.cancel()

	hc.mu.Lock()
	host := hc.host
	hc.mu.Unlock()

	// Replace with a fresh hostConn so the run loop starts clean.
	m.mu.Lock()
	delete(m.hosts, id)
	m.mu.Unlock()

	m.Apply(host)
}

// StopForward closes a single forward's listener, leaving the host connected.
func (m *Manager) StopForward(hostID, forwardID uint) {
	m.mu.Lock()
	hc := m.hosts[hostID]
	m.mu.Unlock()
	if hc != nil {
		hc.removeForward(forwardID)
		m.publish()
	}
}

// Snapshot returns the current status of every managed host and forward.
func (m *Manager) Snapshot() Status {
	m.mu.Lock()
	conns := make([]*hostConn, 0, len(m.hosts))
	for _, hc := range m.hosts {
		conns = append(conns, hc)
	}
	m.mu.Unlock()

	st := Status{Hosts: map[string]HostStatus{}, Forwards: map[string]ForwardStatus{}}
	for _, hc := range conns {
		hc.mu.Lock()
		st.Hosts[fmt.Sprint(hc.host.ID)] = HostStatus{State: hc.state, Error: hc.err}
		for id, f := range hc.forwards {
			state := f.state
			if hc.state != StateConnected && state != StateError {
				state = StateDisconnected
			}
			st.Forwards[fmt.Sprint(id)] = ForwardStatus{State: state, Error: f.err}
		}
		hc.mu.Unlock()
	}
	return st
}

func (m *Manager) publish() { m.emit(m.Snapshot()) }

// hostConn manages one bastion's SSH client, reconnect loop, and forwards.
type hostConn struct {
	mgr *Manager

	mu       sync.Mutex
	host     model.Host
	client   *ssh.Client
	state    State
	err      string
	retries  int
	forwards map[uint]*fwdState

	ctx    context.Context
	cancel context.CancelFunc
}

type fwdState struct {
	fwd   model.Forward
	ln    net.Listener
	state State
	err   string
}

func (hc *hostConn) run() {
	wait := reconnectMinWait
	for {
		if hc.ctx.Err() != nil {
			return
		}

		hc.set(StateConnecting, "")
		hc.mgr.publish()

		client, err := hc.dial()
		if err != nil {
			hc.mu.Lock()
			hc.retries++
			retries := hc.retries
			hc.mu.Unlock()

			if retries >= maxRetries {
				hc.set(StateGivenUp, err.Error())
				hc.mgr.publish()
				return
			}
			hc.set(StateError, err.Error())
			hc.mgr.publish()
			if !hc.sleep(wait) {
				return
			}
			wait = nextBackoff(wait)
			continue
		}
		hc.mu.Lock()
		hc.retries = 0
		hc.mu.Unlock()
		wait = reconnectMinWait

		hc.mu.Lock()
		hc.client = client
		hc.state = StateConnected
		hc.err = ""
		hc.mu.Unlock()

		hc.startForwards(client)
		hc.mgr.publish()

		hc.monitor(client)

		hc.stopForwards()
		hc.mu.Lock()
		hc.client = nil
		if hc.ctx.Err() == nil {
			hc.state = StateDisconnected
		}
		hc.mu.Unlock()
		hc.mgr.publish()

		if !hc.sleep(wait) {
			return
		}
		wait = nextBackoff(wait)
	}
}

// monitor blocks until the connection drops or the host is stopped, sending
// periodic keepalives so a dead link is detected promptly.
func (hc *hostConn) monitor(client *ssh.Client) {
	closed := make(chan struct{})
	go func() {
		client.Wait()
		close(closed)
	}()

	t := time.NewTicker(keepaliveEvery)
	defer t.Stop()
	for {
		select {
		case <-hc.ctx.Done():
			client.Close()
			return
		case <-closed:
			return
		case <-t.C:
			if _, _, err := client.SendRequest("keepalive@openssh.com", true, nil); err != nil {
				client.Close()
				return
			}
		}
	}
}

func (hc *hostConn) dial() (*ssh.Client, error) {
	hc.mu.Lock()
	host := hc.host
	hc.mu.Unlock()

	auths := authMethods(host)
	if len(auths) == 0 {
		return nil, fmt.Errorf("no SSH credentials available (load a key into ssh-agent)")
	}
	cb, err := hostKeyCallback(host.HostKeyPolicy)
	if err != nil {
		return nil, err
	}
	cfg := &ssh.ClientConfig{
		User:            host.User,
		Auth:            auths,
		HostKeyCallback: cb,
		Timeout:         dialTimeout,
	}
	return ssh.Dial("tcp", fmt.Sprintf("%s:%d", host.Hostname, host.Port), cfg)
}

func (hc *hostConn) startForwards(client *ssh.Client) {
	hc.mu.Lock()
	fwds := make([]*fwdState, 0, len(hc.forwards))
	for _, f := range hc.forwards {
		fwds = append(fwds, f)
	}
	hc.mu.Unlock()
	for _, f := range fwds {
		hc.startForward(f, client)
	}
}

func (hc *hostConn) startForward(f *fwdState, client *ssh.Client) {
	hc.mu.Lock()
	if f.ln != nil {
		hc.mu.Unlock()
		return
	}
	hc.mu.Unlock()

	// Take over the local port: kill anything already bound to it (e.g. a
	// manually-started `ssh -f -N -L` tunnel) before claiming it.
	FreeLocalPort(f.fwd.LocalPort)

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", f.fwd.LocalPort))
	if err != nil {
		hc.mu.Lock()
		f.state = StateError
		f.err = portError(f.fwd.LocalPort, err)
		hc.mu.Unlock()
		hc.mgr.publish()
		return
	}

	hc.mu.Lock()
	f.ln = ln
	f.state = StateConnected
	f.err = ""
	hc.mu.Unlock()
	hc.mgr.publish()

	go hc.acceptLoop(f, ln, client)
}

func (hc *hostConn) acceptLoop(f *fwdState, ln net.Listener, client *ssh.Client) {
	for {
		local, err := ln.Accept()
		if err != nil {
			return // listener closed
		}
		go func() {
			remote, err := client.Dial("tcp", fmt.Sprintf("%s:%d", f.fwd.RemoteHost, f.fwd.RemotePort))
			if err != nil {
				local.Close()
				return
			}
			pipe(local, remote)
		}()
	}
}

func (hc *hostConn) stopForwards() {
	hc.mu.Lock()
	for _, f := range hc.forwards {
		if f.ln != nil {
			f.ln.Close()
			f.ln = nil
		}
		f.state = StateDisconnected
	}
	hc.mu.Unlock()
}

// sync reconciles the host's config and forward set with the latest from the
// database, starting/stopping listeners as needed without dropping the link.
func (hc *hostConn) sync(host model.Host) {
	hc.mu.Lock()
	hc.host = host
	client := hc.client

	desired := make(map[uint]model.Forward)
	for _, f := range host.Forwards {
		if f.Enabled {
			desired[f.ID] = f
		}
	}
	for id, f := range hc.forwards {
		if _, ok := desired[id]; !ok {
			if f.ln != nil {
				f.ln.Close()
			}
			delete(hc.forwards, id)
		}
	}
	var toStart []*fwdState
	for id, f := range desired {
		if _, ok := hc.forwards[id]; !ok {
			fs := &fwdState{fwd: f, state: StateConnecting}
			hc.forwards[id] = fs
			toStart = append(toStart, fs)
		}
	}
	hc.mu.Unlock()

	if client != nil {
		for _, fs := range toStart {
			hc.startForward(fs, client)
		}
	}
}

func (hc *hostConn) removeForward(id uint) {
	hc.mu.Lock()
	if f, ok := hc.forwards[id]; ok {
		if f.ln != nil {
			f.ln.Close()
		}
		delete(hc.forwards, id)
	}
	hc.mu.Unlock()
}

func (hc *hostConn) close() {
	hc.cancel()
	hc.mu.Lock()
	for _, f := range hc.forwards {
		if f.ln != nil {
			f.ln.Close()
		}
	}
	if hc.client != nil {
		hc.client.Close()
	}
	hc.mu.Unlock()
}

func (hc *hostConn) set(state State, errMsg string) {
	hc.mu.Lock()
	hc.state = state
	hc.err = errMsg
	hc.mu.Unlock()
}

// sleep waits for d, returning false if the host was stopped meanwhile.
func (hc *hostConn) sleep(d time.Duration) bool {
	select {
	case <-hc.ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

func pipe(a, b net.Conn) {
	done := make(chan struct{}, 2)
	go func() { io.Copy(a, b); done <- struct{}{} }()
	go func() { io.Copy(b, a); done <- struct{}{} }()
	<-done
	a.Close()
	b.Close()
}

func nextBackoff(d time.Duration) time.Duration {
	d *= 2
	if d > reconnectMaxWait {
		return reconnectMaxWait
	}
	return d
}

func portError(port int, err error) string {
	return fmt.Sprintf("local port %d unavailable: %v", port, err)
}
