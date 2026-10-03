# Comics

Web based, tablet-first comic reader.

![Groups](public/screenshots/001.png)
![Group](public/screenshots/002.png)
![Page](public/screenshots/003.png)

## Features

- Upload CBZ / CBR archives (drag and drop, multiple at once), organized into groups
- Reader with swipe and arrow-key navigation; remembers your place in every comic
- Reading history and per-group disk usage stats
- Multi-user, with an admin panel for managing users

## Requirements

- PostgreSQL
- To build: Go 1.26+ and Node.js 22+

## Building

```
$ make build
```

This builds the frontend (`web/`) and embeds it into a single static binary at `build/comics`. Copy that binary anywhere and run it.

## Configuration

The server reads `config.yaml` from the first of these locations that exists:

1. `./config.yaml` (the current working directory)
2. The OS config directory, e.g. `$HOME/.config/comics/config.yaml` on Linux,
   `~/Library/Application Support/comics/config.yaml` on macOS

You can also pass a path explicitly with `comics -config /path/to/config.yaml`. See [`config.example.yaml`](config.example.yaml) for all options:

```yaml
listen: ":3000"
database_url: "postgres://postgres:postgres@localhost:5432/comics_development?sslmode=disable"
storage_dir: "storage"      # page images; relative to the config file
max_upload_mb: 2048
secure_cookies: false       # set to true behind HTTPS
```

`COMICS_DATABASE_URL` overrides `database_url` if set. The database schema is created/migrated automatically on startup.

## Use

Run `./build/comics` and open [http://localhost:3000](http://localhost:3000). On first run you'll be redirected to create the initial admin user.

## Upgrading from the Rails version

The Go server uses the same database schema, so point `database_url` at the existing database. Existing accounts and passwords keep working. To keep previously uploaded comics, set `storage_dir` to the old app's `public/system` directory (the on-disk layout is the same).

## Development

Run the Go server and the Vite dev server (with hot reload) in two terminals:

```
$ make dev-backend
$ make dev-frontend
```

Then open [http://localhost:5173](http://localhost:5173). Vite proxies `/api` to the Go server on port 3000.

Docker Compose is also available (`docker-compose up`), which runs PostgreSQL and the app.

Run tests with `make test`. `make clean` removes the `build/` directory and the built frontend.
