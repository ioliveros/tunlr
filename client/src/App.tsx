import { useCallback, useEffect, useState } from 'react';
import './App.css';
import { ListHosts, AddConnection, DeleteForward } from '../wailsjs/go/main/App';
import { dto, model } from '../wailsjs/go/models';

function isPort(v: string): boolean {
    const n = Number(v);
    return Number.isInteger(n) && n >= 1 && n <= 65535;
}

// StatusDot is a placeholder until the SSH engine reports live state.
function StatusDot() {
    return <span className="status-dot idle" title="idle — connect coming soon" />;
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
                <div className="lights">
                    <i className="r" />
                    <i className="y" />
                    <i className="g" />
                </div>
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

function ForwardRow({ forward, onDelete }: { forward: model.Forward; onDelete: (id: number) => void }) {
    return (
        <div className="fwd-row">
            <div className="col-name">{forward.label || <span className="muted">—</span>}</div>
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

function HostPane({ host, onChanged }: { host: model.Host; onChanged: () => void }) {
    async function remove(id: number) {
        await DeleteForward(id);
        onChanged();
    }

    return (
        <section className="pane">
            <header className="pane-header">
                <div className="lights">
                    <i className="r" />
                    <i className="y" />
                    <i className="g" />
                </div>
                <span className="pane-title">{host.name}</span>
                <span className="muted">
                    {host.user}@{host.hostname}:{host.port}
                </span>
                <StatusDot />
            </header>
            <div className="pane-body">
                <div className="fwd-row head">
                    <div className="col-name">NAME</div>
                    <div className="col-ip">HOST</div>
                    <div className="col-port">LOCAL</div>
                    <div className="col-port">REMOTE</div>
                    <div className="col-action" />
                </div>
                {host.forwards?.length ? (
                    host.forwards.map((f) => <ForwardRow key={f.id} forward={f} onDelete={remove} />)
                ) : (
                    <div className="empty">no connections yet</div>
                )}
            </div>
        </section>
    );
}

export default function App() {
    const [hosts, setHosts] = useState<model.Host[]>([]);
    const [error, setError] = useState('');

    const refresh = useCallback(() => {
        ListHosts()
            .then(setHosts)
            .catch((e) => setError(String(e)));
    }, []);

    useEffect(() => {
        refresh();
    }, [refresh]);

    return (
        <div id="App">
            <div className="topbar">
                <span className="brand">tunlr</span>
                <span className="muted">ssh tunnel manager</span>
            </div>
            {error && <div className="error">{error}</div>}
            <main className="panes">
                <NewConnectionForm onAdded={refresh} />
                {hosts.map((h) => (
                    <HostPane key={h.id} host={h} onChanged={refresh} />
                ))}
            </main>
        </div>
    );
}
