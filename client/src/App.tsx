import { useEffect, useState } from 'react';
import './App.css';
import { ListHosts } from '../wailsjs/go/main/App';
import { model } from '../wailsjs/go/models';

function App() {
    const [hosts, setHosts] = useState<model.Host[]>([]);
    const [error, setError] = useState<string>('');

    useEffect(() => {
        ListHosts()
            .then(setHosts)
            .catch((e) => setError(String(e)));
    }, []);

    return (
        <div id="App">
            <h1>tunlr</h1>
            {error && <div className="error">{error}</div>}
            {hosts.map((host) => (
                <div key={host.id} className="host">
                    <h2>
                        {host.name}{' '}
                        <span className="muted">
                            {host.user}@{host.hostname}:{host.port}
                        </span>
                    </h2>
                    <ul>
                        {host.forwards?.map((f) => (
                            <li key={f.id}>
                                <code>
                                    localhost:{f.localPort} → {f.remoteHost}:{f.remotePort}
                                </code>{' '}
                                {f.label}
                            </li>
                        ))}
                    </ul>
                </div>
            ))}
        </div>
    );
}

export default App;
