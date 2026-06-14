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

// Dot shows a connection's live state: green connected, amber connecting,
// red disconnected/error.
function Dot({ conn }: { conn?: ConnState }) {
    const state = conn?.state ?? 'disconnected';
    const cls = state === 'connected' ? 'green' : state === 'connecting' ? 'amber' : 'red';
    const title = conn?.error ? `${state} — ${conn.error}` : state;
    return <span className={`status-dot ${cls}`} title={title} />;
}

function NewConnectionForm({ onAdded }: { onAdded: () => void }) {
    const [name, setName] = useState('');
    const [host, setHost] = useState('');
    const [remotePort, setRemotePort] = useState('');
    const [localPort, setLocalPort] = useState('');
    const [domain, setDomain] = useState('');
    const [busy, setBusy] = useState(false);
    const [err, setErr] = useState('');

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
                <span className="pane-title">new connection</span>
            </header>
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
                <div className="field add-field">
                    <span>&nbsp;</span>
                    <button className="add-btn" disabled={!valid || busy} onClick={submit}>
                        {busy ? '…' : '+ Add'}
                    </button>
                </div>
            </div>
            {err && <div className="conn-err">{err}</div>}
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
            <div className="col-name">
                <Dot conn={conn} />
                {forward.label || <span className="muted">—</span>}
            </div>
            <div className="col-ip">{forward.remoteHost}</div>
            <div className="col-port accent">{forward.localPort}</div>
            <div className="col-port">{forward.remotePort}</div>
            <div className="col-action">
                <button className="icon-btn danger" title="Remove" onClick={() => onDelete(forward.id)}>
                    ×
                </button>
            </div>
        </div>
    );
}

function HostPane({ host, status, onChanged }: { host: model.Host; status: Status; onChanged: () => void }) {
    const [keyBusy, setKeyBusy] = useState(false);
    const [availableKeys, setAvailableKeys] = useState<dto.SSHKey[] | null>(null);
    const hostConn = status.hosts[String(host.id)];
    const keyName = host.keyPath ? host.keyPath.split('/').pop() : null;

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
                <span className="pane-title">{host.name}</span>
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
                    <Dot conn={hostConn} />
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
                    <div className="col-port">LOCAL PORT</div>
                    <div className="col-port">REMOTE PORT</div>
                    <div className="col-action" />
                </div>
                {host.forwards?.length ? (
                    host.forwards.map((f) => (
                        <ForwardRow key={f.id} forward={f} conn={status.forwards[String(f.id)]} onDelete={remove} />
                    ))
                ) : (
                    <div className="empty">no connections yet</div>
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

export default function App() {
    const [hosts, setHosts] = useState<model.Host[]>([]);
    const [status, setStatus] = useState<Status>(emptyStatus);
    const [error, setError] = useState('');
    const [showAbout, setShowAbout] = useState(false);

    const refresh = useCallback(() => {
        ListHosts()
            .then(setHosts)
            .catch((e) => setError(String(e)));
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
                {hosts.map((h) => (
                    <HostPane key={h.id} host={h} status={status} onChanged={refresh} />
                ))}
            </main>
            {showAbout && <AboutPanel onClose={() => setShowAbout(false)} />}
        </div>
    );
}
