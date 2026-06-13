import { useCallback, useEffect, useState } from 'react';
import './App.css';
import { ListHosts, AddForward, DeleteForward } from '../wailsjs/go/main/App';
import { model } from '../wailsjs/go/models';

function isPort(v: string): boolean {
    const n = Number(v);
    return Number.isInteger(n) && n >= 1 && n <= 65535;
}

// StatusDot is a placeholder until the SSH engine reports live state.
function StatusDot() {
    return <span className="status-dot idle" title="idle — connect coming soon" />;
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

function AddForwardRow({ hostId, onAdded }: { hostId: number; onAdded: () => void }) {
    const [name, setName] = useState('');
    const [ip, setIp] = useState('');
    const [local, setLocal] = useState('');
    const [remote, setRemote] = useState('');
    const [busy, setBusy] = useState(false);
    const [err, setErr] = useState('');

    const valid = ip.trim() !== '' && isPort(local) && isPort(remote);

    async function submit() {
        if (!valid || busy) return;
        setErr('');
        setBusy(true);
        try {
            await AddForward(
                model.Forward.createFrom({
                    hostId,
                    label: name.trim(),
                    remoteHost: ip.trim(),
                    localPort: Number(local),
                    remotePort: Number(remote),
                    enabled: true,
                })
            );
            setName('');
            setIp('');
            setLocal('');
            setRemote('');
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
        <div className="fwd-row add-row">
            <input placeholder="name" value={name} onChange={(e) => setName(e.target.value)} onKeyDown={onKey} />
            <input placeholder="10.0.0.0" value={ip} onChange={(e) => setIp(e.target.value)} onKeyDown={onKey} />
            <input placeholder="local" value={local} onChange={(e) => setLocal(e.target.value)} onKeyDown={onKey} />
            <input placeholder="remote" value={remote} onChange={(e) => setRemote(e.target.value)} onKeyDown={onKey} />
            <div className="col-action">
                <button className="add-btn" disabled={!valid || busy} onClick={submit}>
                    {busy ? '…' : '+ Add'}
                </button>
            </div>
            {err && <div className="row-err">{err}</div>}
        </div>
    );
}

function HostPane({ host, onChanged }: { host: model.Host; onChanged: () => void }) {
    async function remove(id: number) {
        if (!window.confirm('Remove this forward?')) return;
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
                    <div className="col-ip">ORIGINAL IP</div>
                    <div className="col-port">LOCAL</div>
                    <div className="col-port">REMOTE</div>
                    <div className="col-action" />
                </div>
                {host.forwards?.length ? (
                    host.forwards.map((f) => <ForwardRow key={f.id} forward={f} onDelete={remove} />)
                ) : (
                    <div className="empty">no forwards yet</div>
                )}
                <AddForwardRow hostId={host.id} onAdded={onChanged} />
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
                {hosts.map((h) => (
                    <HostPane key={h.id} host={h} onChanged={refresh} />
                ))}
            </main>
        </div>
    );
}
