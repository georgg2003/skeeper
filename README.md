# Skeeper

Skeeper is a **password manager** with a Go CLI, **client-side encryption**, and **gRPC** backends for **authentication** and **vault synchronization**. The network layer never sees your **vault master password** or **plaintext secrets**—only ciphertext, KDF salt, and a master-key verifier.

For a specification-style description (architecture, security model, sync rules, and API overview), see **[docs/PROJECT.md](docs/PROJECT.md)**.

## Repository layout

| Path | Role |
|------|------|
| `cmd/client` | `skeepercli` — Cobra CLI |
| `cmd/auther` | **Auther** — accounts, JWT issue/refresh (Postgres) |
| `cmd/skeeper` | **Skeeper** — encrypted entry sync + vault crypto metadata (Postgres) |
| `api/` | Protobuf definitions and generated Go gRPC code |
| `internal/client/` | CLI use cases, local SQLite vault, gRPC clients |
| `internal/auther/`, `internal/skeeper/` | Server use cases, delivery, repositories |
| `config/` | Example YAML for client and servers |

## Prerequisites

- Go toolchain matching `go.mod`
- PostgreSQL for **auther** and **skeeper** (see server configs under `config/`)
- `protoc` with Go plugins if you regenerate `api/` (`make proto`)

## Build

```bash
make build    # bin/auther, bin/skeeper, bin/skeepercli
```

Other useful targets: `make test`, `make lint`, `make check`.

## TLS and keys (optional)

For TLS-terminated gRPC, generate material (example paths in `Makefile`):

```bash
make gen-keys
```

Point server and client YAML at the certificate and enable `grpc_tls` where appropriate.

## Configuration

- **CLI**: `config/client.yaml` or `--config`; overrides via `SKEEPERCLI_*` env vars (see file comments).
- **Servers**: `config/auther.yaml`, `config/skeeper.yaml` (ports, Postgres, JWT paths).

## Quick start (local dev)

1. Start Postgres and apply migrations if your setup requires them.
2. Run **auther** and **skeeper** (each with its config).
3. Use the CLI:

```bash
./bin/skeepercli --config config/client.yaml register
./bin/skeepercli login
./bin/skeepercli add password   # follow prompts (master password encrypts locally)
./bin/skeepercli sync
```

Use `skeepercli --help` and subcommands such as `list`, `get`, `update`, `delete` for full CRUD.

## Documentation

- **[docs/PROJECT.md](docs/PROJECT.md)** — project specification: components, cryptography, entry types, sync semantics, and gRPC surface.

## License

See repository license if present.
