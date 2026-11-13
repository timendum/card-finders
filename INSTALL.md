# Install and run

Card Finders is one Go process plus a built React bundle. The Go process can
serve the bundle itself, or you can put a reverse proxy in front and let it
serve the files.

## Requirements

- Go, to build the server.
- Bun, to build the frontend.
- An Argon2id hash of the host password.]

Use any Argon2id tool,
like [Argon2 Hash Generator](https://emn178.github.io/online-tools/argon2/)
and produce a PHC string such as
`$argon2id$v=19$m=8192,t=2,p=1$<salt>$<hash>`. 
Keep `m` (KiB) low, since verification runs in memory.

## Build

Frontend, from `frontend/`:

```
bun install
bun run build
```

This writes the bundle to `frontend/dist`. The server reads it from disk on
every request, so a rebuild is picked up without restarting.

Backend, from `backend/`:

```
go build -o card-finders ./cmd/server
```

On Windows, name the output `card-finders.exe`.

## Server options

| Flag           | Env var                  | Default      | Purpose                                              |
| -------------- | ------------------------ | ------------ | ---------------------------------------------------- |
| `--auth-hash`  | `CARD_FINDERS_AUTH_HASH` | required     | Argon2id PHC hash of the host password.              |
| `--config`     | `CARD_FINDERS_LEVELS`    | `levels.toml`| Level config file.                                   |
| `--static-dir` |                          | empty        | Directory with the built frontend. Empty disables it.|
| `--listen`     |                          | `:8080`      | Listen address.                                      |
| `--seed`       |                          | `0`          | RNG seed. `0` picks a random one, logged with `-v 1`.|

The flag wins over the env var. A `--config` path or `CARD_FINDERS_LEVELS` that
points to a missing file stops the server. If neither is set and `levels.toml`
is missing, the server logs a notice and uses a built-in single level. The
default config path is relative to the working directory, so start the server
from the repository root or pass absolute paths.

The hash contains `$`, so wrap it in single quotes in a shell.

## Level configuration

Levels are defined in a TOML file. Each level is a named table; the table name
is the level name and the level number is its position in the file. Keys:

- `silver`, `gold`, `lives`: the main settings.
- `extraSilver`, `extraGold`: optional decoy specials, default 0.
- `jokers`: optional, default 4.

```toml
["Tutorial One"]
silver = 0
gold = 0
lives = 4

["Level two"]
silver = 4
gold = 2
lives = 3
extraSilver = 1
jokers = 6
```

The allowed ranges are listed under Level in [GAME.md](GAME.md). Anything out
of range or malformed stops the server at startup. `levels.toml` in the
repository root is a sample.

## Option 1: Go serves everything

From the repository root:

```
./backend/card-finders \
  --config levels.toml \
  --static-dir frontend/dist \
  --auth-hash '<argon2id PHC hash>'
```

Open `http://localhost:8080`. The server serves the frontend from `/`, the REST
API from `/api` and the WebSocket from `/ws`, all on one port.

The bundle references its assets with absolute `/assets/...` URLs, so it must
be served from the root of a host. A sub-path such as `/cards/` does not work.

The server does not speak TLS. The host password is sent in clear text on
game creation, so on anything beyond a trusted network use option 2 with HTTPS.

## Option 2: Reverse proxy

Start the server without `--static-dir` and bind it to the loopback interface.
It then serves only `/api` and `/ws`.

```
./backend/card-finders \
  --config /srv/card-finders/levels.toml \
  --listen 127.0.0.1:8080 \
  --auth-hash '<argon2id PHC hash>'
```

Copy `frontend/dist` to a directory nginx can read, for example
`/var/www/card-finders`. nginx serves the files, forwards `/api` to the server
and forwards `/ws` with the WebSocket upgrade headers.

```nginx
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

server {
    listen 443 ssl;
    http2 on;
    server_name cards.example.com;

    ssl_certificate     /etc/letsencrypt/live/cards.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/cards.example.com/privkey.pem;

    root /var/www/card-finders;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 1h;
        proxy_send_timeout 1h;
    }
}

server {
    listen 80;
    server_name cards.example.com;
    return 301 https://$host$request_uri;
}
```

`proxy_read_timeout` matters for `/ws`. nginx closes an idle proxied
connection after 60 seconds by default, and a game can sit idle longer than
that while players think.

## Updating

- New frontend: run `bun run build` and replace the served directory (or let
  option 1 pick it up from `frontend/dist`). No restart needed.
- New backend: rebuild the binary and restart it. A restart ends the current
  game, because all state is in memory.
- New levels: edit the config file and restart.

To work on the code, see [README.md](README.md).
