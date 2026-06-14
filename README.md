<div align="center">
  <img src="build/appicon.png" width="80" />
  <h1>tunlr</h1>
  <p>A lightweight desktop client for managing SSH tunnels. Define port forwards once and tunlr keeps them alive, reconnecting automatically if a connection drops.</p>

  [![CI](https://github.com/ioliveros/tunlr/actions/workflows/ci.yml/badge.svg)](https://github.com/ioliveros/tunlr/actions/workflows/ci.yml)
  [![codecov](https://codecov.io/gh/ioliveros/tunlr/branch/main/graph/badge.svg)](https://codecov.io/gh/ioliveros/tunlr)
  [![Version](https://img.shields.io/github/v/release/ioliveros/tunlr?label=version&color=2dd4a7)](https://github.com/ioliveros/tunlr/releases)
</div>

## Features

- Add and remove SSH tunnels through a clean UI
- Forwards grouped by bastion host (domain)
- Auto-reconnect with exponential backoff (up to 3 retries, then manual reconnect)
- SSH key picker — choose a key per host from `~/.ssh/`
- Live connection status (connecting / connected / error / given-up)
- Port conflict resolution — takes over a local port automatically

## Requirements

| Tool | Version |
|------|---------|
| Go | 1.24+ |
| Node.js | 18+ |
| npm | 9+ |
| Wails CLI | v2.12+ |

## Development

### 1. Install Wails

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 2. Clone and install dependencies

```bash
git clone https://github.com/ioliveros/tunlr.git
cd tunlr
go mod download
cd client && npm install && cd ..
```

### 3. Run in dev mode

```bash
wails dev
```

This starts the app with hot-reload for the frontend and live-recompile for Go. The Vite dev server runs automatically.

### 4. Run tests

```bash
go test ./...
```

## Building

### macOS

```bash
./build.sh
```

This reads the version from `VERSION`, stamps the git commit and build date into the binary via `-ldflags`, then runs `wails build`. The built app is output to `build/bin/tunlr.app`.

To bump the version, edit `VERSION` before running the script:

```bash
echo "0.2.0" > VERSION
./build.sh
```

Any extra flags are forwarded to `wails build` — e.g. `./build.sh -clean` for a clean build.

To produce a distributable `.dmg` or notarized build, see the [Wails packaging docs](https://wails.io/docs/guides/packaging).

### Regenerate the app icon

If you update `build/appicon.png`, rebuild the `.icns` with:

```bash
mkdir -p /tmp/tunlr.iconset
for size in 16 32 64 128 256 512; do
  sips -z $size $size build/appicon.png --out /tmp/tunlr.iconset/icon_${size}x${size}.png
  sips -z $((size*2)) $((size*2)) build/appicon.png --out /tmp/tunlr.iconset/icon_${size}x${size}@2x.png
done
iconutil -c icns /tmp/tunlr.iconset -o build/bin/tunlr.app/Contents/Resources/iconfile.icns
```

## Project structure

```
tunlr/
├── app.go                  # Wails bindings — all methods exposed to the frontend
├── main.go                 # Entry point
├── wails.json              # Wails project config
├── build/
│   └── appicon.png         # Source icon (1024×1024)
├── client/                 # React + TypeScript frontend (Vite)
│   └── src/
│       ├── App.tsx
│       └── App.css
└── internal/
    ├── config/             # App config (DB path, env)
    ├── db/                 # SQLite connection via GORM
    ├── dto/                # Input/output types for the frontend
    ├── model/              # GORM models (Host, Forward)
    ├── repository/         # Database access layer
    ├── service/            # Business logic (TunnelService)
    └── tunnel/             # SSH engine (Manager, auth, known_hosts)
```

## Contributing

1. Fork the repo and create a branch: `git checkout -b feat/your-feature`
2. Make your changes — run `go test ./...` and `wails dev` to verify
3. Keep commits focused; one logical change per commit
4. Open a pull request against `main` with a clear description of what and why

### Code conventions

- Go: standard `gofmt` formatting; exported symbols get a one-line doc comment
- TypeScript: no `any` except where the Wails runtime requires it; prefer named functions over anonymous arrows for components
- No comments that restate what the code already says — only comment the *why*

## License

MIT
