import { useCallback, useEffect, useState } from 'react';
import './App.css';
import appIcon from './assets/appicon.png';
import { ListHosts, AddConnection, DeleteForward, GetStatus, SetHostKey, PickSSHKey, ListSSHKeys, ReconnectHost, GetVersion } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime';
import { dto, model } from '../wailsjs/go/models';

type ConnState = { state: string; error?: string };
type Status = { hosts: Record<string, ConnState>; forwards: Record<string, ConnState> };

const emptyStatus: Status = { hosts: {}, forwards: {} };

function isPort(v: string): boolean {
    const n = Number(v);
    return Number.isInteger(n) && n >= 1 && n <= 65535;
}

// Friendly label + one-line hint shown in the status tooltip.
const STATE_INFO: Record<string, { label: string; hint: string }> = {
    connected: { label: 'Connected', hint: 'Tunnel is live and forwarding traffic.' },
    connecting: { label: 'Connecting…', hint: 'Establishing the SSH connection.' },
    disconnected: { label: 'Disconnected', hint: 'Tunnel is not active.' },
    error: { label: 'Error', hint: 'The connection failed.' },
    'given-up': { label: 'Gave up', hint: 'Automatic reconnect exhausted after repeated failures.' },
};

// Dot shows a connection's live state: green connected, amber connecting,
// red disconnected/error. Hovering reveals a styled tooltip with a friendly
// label, a short hint, and the underlying error when present. `align` anchors
// the tooltip's edge so it doesn't overflow the window (right for header dots
// near the pane edge, left for forward-row dots).
function Dot({ conn, align = 'left' }: { conn?: ConnState; align?: 'left' | 'right' }) {
    const state = conn?.state ?? 'disconnected';
    const cls = state === 'connected' ? 'green' : state === 'connecting' ? 'amber' : 'red';
    const info = STATE_INFO[state] ?? { label: state, hint: '' };
    return (
        <span className={`status-wrap tip-${align}`}>
            <span className={`status-dot ${cls}`} />
            <span className="status-tip" role="tooltip">
                <span className="tip-state">
                    <span className={`status-dot ${cls}`} />
                    {info.label}
                </span>
                {info.hint && <span className="tip-hint">{info.hint}</span>}
                {conn?.error && <span className="tip-err">{conn.error}</span>}
            </span>
        </span>
    );
}

function NewConnectionForm({ onAdded }: { onAdded: () => void }) {
    const [name, setName] = useState('');
    const [host, setHost] = useState('');
    const [remotePort, setRemotePort] = useState('');
    const [localPort, setLocalPort] = useState('');
    const [domain, setDomain] = useState('');
    const [busy, setBusy] = useState(false);
    const [err, setErr] = useState('');
    const [collapsed, setCollapsed] = useState(true);

    const valid =
        name.trim() !== '' &&
        host.trim() !== '' &&
        domain.trim() !== '' &&
        isPort(remotePort) &&
        isPort(localPort);

    async function submit() {
        if (!valid || busy) return;
        setErr('');
        setBusy(true);
        try {
            await AddConnection(
                dto.ConnectionInput.createFrom({
                    connectionName: name.trim(),
                    host: host.trim(),
                    remotePort: Number(remotePort),
                    localPort: Number(localPort),
                    domain: domain.trim(),
                })
            );
            setName('');
            setHost('');
            setRemotePort('');
            setLocalPort('');
            setDomain('');
            onAdded();
        } catch (e) {
            setErr(String(e));
        } finally {
            setBusy(false);
        }
    }

    function onKey(e: { key: string }) {
        if (e.key === 'Enter') submit();
    }

    return (
        <section className="pane new-conn">
            <header className="pane-header">
                <span className="pane-title">create new connection</span>
                <button
                    className="collapse-btn"
                    onClick={() => setCollapsed((c) => !c)}
                    title={collapsed ? 'Expand' : 'Minimize'}
                    aria-label={collapsed ? 'Expand' : 'Minimize'}
                    aria-expanded={!collapsed}
                >
                    {collapsed ? '+' : '−'}
                </button>
            </header>
            {!collapsed && (
            <>
            <div className="conn-grid">
                <label className="field">
                    <span>connection name</span>
                    <input placeholder="my-db" value={name} onChange={(e) => setName(e.target.value)} onKeyDown={onKey} />
                </label>
                <label className="field">
                    <span>host</span>
                    <input placeholder="10.0.0.0" value={host} onChange={(e) => setHost(e.target.value)} onKeyDown={onKey} />
                </label>
                <label className="field">
                    <span>remote port</span>
                    <input placeholder="5432" value={remotePort} onChange={(e) => setRemotePort(e.target.value)} onKeyDown={onKey} />
                </label>
                <label className="field">
                    <span>local port</span>
                    <input placeholder="5432" value={localPort} onChange={(e) => setLocalPort(e.target.value)} onKeyDown={onKey} />
                </label>
                <label className="field">
                    <span>domain used to connect</span>
                    <input placeholder="gcp@me.ioliveros.dev" value={domain} onChange={(e) => setDomain(e.target.value)} onKeyDown={onKey} />
                </label>
            </div>
            <div className="conn-actions">
                <button className="add-btn" disabled={!valid || busy} onClick={submit}>
                    {busy ? '…' : '+ Add'}
                </button>
            </div>
            {err && <div className="conn-err">{err}</div>}
            </>
            )}
        </section>
    );
}

function ForwardRow({
    forward,
    conn,
    onDelete,
}: {
    forward: model.Forward;
    conn?: ConnState;
    onDelete: (id: number) => void;
}) {
    return (
        <div className="fwd-row">
            <div className="col-name">{forward.label || <span className="muted">—</span>}</div>
            <div className="col-ip">{forward.remoteHost}</div>
            <div className="col-port">{forward.remotePort}</div>
            <div className="col-port accent">{forward.localPort}</div>
            <div className="col-status">
                <Dot conn={conn} align="right" />
            </div>
            <div className="col-action">
                <button className="icon-btn danger" title="Remove" onClick={() => onDelete(forward.id)}>
                    ×
                </button>
            </div>
        </div>
    );
}

// Rows shown per page before the forward table paginates. Keeps each pane
// short enough to fit within the fixed window height.
const FORWARDS_PER_PAGE = 5;

function HostPane({ host, status, onChanged }: { host: model.Host; status: Status; onChanged: () => void }) {
    const [keyBusy, setKeyBusy] = useState(false);
    const [availableKeys, setAvailableKeys] = useState<dto.SSHKey[] | null>(null);
    const [page, setPage] = useState(0);
    const hostConn = status.hosts[String(host.id)];
    const keyName = host.keyPath ? host.keyPath.split('/').pop() : null;

    const forwards = host.forwards ?? [];
    const pageCount = Math.max(1, Math.ceil(forwards.length / FORWARDS_PER_PAGE));
    // Clamp in case forwards shrank (e.g. a deletion) below the current page.
    const current = Math.min(page, pageCount - 1);
    const visible = forwards.slice(current * FORWARDS_PER_PAGE, current * FORWARDS_PER_PAGE + FORWARDS_PER_PAGE);

    async function openKeyPicker() {
        if (keyBusy) return;
        setKeyBusy(true);
        try {
            const keys = await ListSSHKeys();
            setAvailableKeys(keys);
        } finally {
            setKeyBusy(false);
        }
    }

    async function selectKey(path: string) {
        setAvailableKeys(null);
        if (!path) return;
        if (path === '__browse__') {
            const picked = await PickSSHKey();
            if (!picked) return;
            setKeyBusy(true);
            try {
                await SetHostKey(host.id, picked);
                onChanged();
            } finally {
                setKeyBusy(false);
            }
            return;
        }
        setKeyBusy(true);
        try {
            await SetHostKey(host.id, path);
            onChanged();
        } finally {
            setKeyBusy(false);
        }
    }

    async function clearKey() {
        if (keyBusy) return;
        setKeyBusy(true);
        try {
            await SetHostKey(host.id, '');
            onChanged();
        } finally {
            setKeyBusy(false);
        }
    }

    async function remove(id: number) {
        await DeleteForward(id);
        onChanged();
    }

    return (
        <section className="pane">
            <header className="pane-header">
                <span className="pane-title">
                    <span className="bastion-label">bastion:</span> {host.name}
                </span>
                <span className="muted">
                    {host.user}@{host.hostname}:{host.port}
                </span>
                <div className="pane-status">
                    <div className="key-ctrl">
                        {availableKeys !== null ? (
                            <select
                                className="key-select"
                                autoFocus
                                defaultValue=""
                                onChange={(e) => selectKey(e.target.value)}
                                onBlur={() => setAvailableKeys(null)}
                            >
                                <option value="" disabled>select key…</option>
                                {availableKeys.map((k) => (
                                    <option key={k.path} value={k.path}>{k.name}</option>
                                ))}
                                <option value="__browse__">browse…</option>
                            </select>
                        ) : keyName ? (
                            <span className="key-badge">
                                <button className="key-badge-name" disabled={keyBusy} onClick={openKeyPicker} title="Change key">
                                    {keyName}
                                </button>
                                <button
                                    className="icon-btn"
                                    disabled={keyBusy}
                                    onClick={clearKey}
                                    title="Clear key — revert to ssh-agent"
                                >×</button>
                            </span>
                        ) : (
                            <button className="key-btn" disabled={keyBusy} onClick={openKeyPicker}>
                                {keyBusy ? '…' : 'pick key'}
                            </button>
                        )}
                    </div>
                    <Dot conn={hostConn} align="right" />
                </div>
            </header>
            {(hostConn?.state === 'error' || hostConn?.state === 'given-up') && hostConn.error && (
                <div className="host-err">
                    <span>{hostConn.error}</span>
                    {hostConn.state === 'given-up' && (
                        <button className="reconnect-btn" onClick={() => ReconnectHost(host.id)}>
                            reconnect
                        </button>
                    )}
                </div>
            )}
            <div className="pane-body">
                <div className="fwd-row head">
                    <div className="col-name">NAME</div>
                    <div className="col-ip">HOST</div>
                    <div className="col-port">REMOTE PORT</div>
                    <div className="col-port">LOCAL PORT</div>
                    <div className="col-status">STATUS</div>
                    <div className="col-action" />
                </div>
                {forwards.length ? (
                    visible.map((f) => (
                        <ForwardRow key={f.id} forward={f} conn={status.forwards[String(f.id)]} onDelete={remove} />
                    ))
                ) : (
                    <div className="empty">no connections yet</div>
                )}
                {/* Pad short pages so the table height stays constant across pages. */}
                {pageCount > 1 &&
                    Array.from({ length: FORWARDS_PER_PAGE - visible.length }).map((_, i) => (
                        <div className="fwd-row filler" key={`filler-${i}`} aria-hidden="true">
                            <div className="col-name">&nbsp;</div>
                            <div className="col-ip">&nbsp;</div>
                            <div className="col-port">&nbsp;</div>
                            <div className="col-port">&nbsp;</div>
                            <div className="col-status"><span className="status-dot" /></div>
                            <div className="col-action">
                                <button className="icon-btn danger" tabIndex={-1}>×</button>
                            </div>
                        </div>
                    ))}
                {pageCount > 1 && (
                    <div className="pagination">
                        <button
                            className="page-btn"
                            disabled={current === 0}
                            onClick={() => setPage(current - 1)}
                            title="Previous page"
                            aria-label="Previous page"
                        >
                            ‹
                        </button>
                        <span className="page-info">
                            {current + 1} / {pageCount}
                        </span>
                        <button
                            className="page-btn"
                            disabled={current >= pageCount - 1}
                            onClick={() => setPage(current + 1)}
                            title="Next page"
                            aria-label="Next page"
                        >
                            ›
                        </button>
                    </div>
                )}
            </div>
        </section>
    );
}

function AboutPanel({ onClose }: { onClose: () => void }) {
    const [ver, setVer] = useState<Record<string, string>>({});

    useEffect(() => {
        GetVersion().then(setVer).catch(() => {});
    }, []);

    return (
        <div className="about-backdrop" onClick={onClose}>
            <div className="about-panel" onClick={(e) => e.stopPropagation()}>
                <img src={appIcon} width="64" className="about-icon" />
                <h2 className="about-name">tunlr</h2>
                <p className="about-desc">a lightweight ssh tunnel client manager</p>
                <div className="about-meta">
                    <span>{ver.version ?? 'dev'}</span>
                    {ver.commit && ver.commit !== 'unknown' && (
                        <span className="muted">({ver.commit})</span>
                    )}
                </div>
                <a className="about-author" href="https://ioliveros.dev" target="_blank" rel="noreferrer">
                    ioliveros.dev
                </a>
            </div>
        </div>
    );
}

// SkeletonPane mimics a grouped-connection pane while the initial host list
// loads, showing FORWARDS_PER_PAGE shimmer rows so the layout doesn't jump.
function SkeletonPane() {
    return (
        <section className="pane skeleton">
            <header className="pane-header">
                <span className="sk sk-title" />
                <span className="sk sk-sub" />
            </header>
            <div className="pane-body">
                <div className="fwd-row head">
                    <div className="col-name">NAME</div>
                    <div className="col-ip">HOST</div>
                    <div className="col-port">REMOTE PORT</div>
                    <div className="col-port">LOCAL PORT</div>
                    <div className="col-status">STATUS</div>
                    <div className="col-action" />
                </div>
                {Array.from({ length: FORWARDS_PER_PAGE }).map((_, i) => (
                    <div className="fwd-row" key={i}>
                        <div className="col-name"><span className="sk sk-cell" /></div>
                        <div className="col-ip"><span className="sk sk-cell" /></div>
                        <div className="col-port"><span className="sk sk-cell short" /></div>
                        <div className="col-port"><span className="sk sk-cell short" /></div>
                        <div className="col-status"><span className="sk sk-dot" /></div>
                        <div className="col-action" />
                    </div>
                ))}
            </div>
        </section>
    );
}

export default function App() {
    const [hosts, setHosts] = useState<model.Host[]>([]);
    const [status, setStatus] = useState<Status>(emptyStatus);
    const [error, setError] = useState('');
    const [showAbout, setShowAbout] = useState(false);
    const [loading, setLoading] = useState(true);

    const refresh = useCallback(() => {
        ListHosts()
            .then(setHosts)
            .catch((e) => setError(String(e)))
            .finally(() => setLoading(false));
    }, []);

    useEffect(() => {
        refresh();
        GetStatus()
            .then((s) => setStatus(s as Status))
            .catch(() => {});
        const off = EventsOn('tunnel:status', (s: Status) => setStatus(s));
        return () => off();
    }, [refresh]);

    return (
        <div id="App">
            <div className="topbar">
                <span className="brand">tunlr</span>
                <span className="muted">ssh tunnel manager</span>
                <button className="about-btn" onClick={() => setShowAbout(true)}>?</button>
            </div>
            {error && <div className="error">{error}</div>}
            <main className="panes">
                <NewConnectionForm onAdded={refresh} />
                {loading ? (
                    <SkeletonPane />
                ) : (
                    hosts.map((h) => (
                        <HostPane key={h.id} host={h} status={status} onChanged={refresh} />
                    ))
                )}
            </main>
            {showAbout && <AboutPanel onClose={() => setShowAbout(false)} />}
        </div>
    );
}
