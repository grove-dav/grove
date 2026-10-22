<!--
SPDX-FileCopyrightText: 2026 Grove contributors

SPDX-License-Identifier: Apache-2.0
-->

# grove

Calendar and contacts for ownCloud Infinite Scale — CalDAV/CardDAV, Graph-shaped API, Vue UI

## Development

Requires Go 1.26.10 and [go-task](https://taskfile.dev).

```sh
task build
task test
GROVE_OIDC_ISSUER=https://ocis.localhost task run
```

Run `task --list-all` for the full list, or `task check` to run fmt/vet/lint/test
together (what CI runs).

Config is env vars and/or a YAML file passed via `--config grove.yaml`; env wins
over file, file wins over defaults.

| Env var | YAML key | Notes |
|---|---|---|
| `GROVE_HTTP_ADDR` | `http_addr` | Listen address, default `:8080` |
| `GROVE_LOG_LEVEL` | `log_level` | `debug`, `info`, `warn` or `error`, default `info` |
| `GROVE_OIDC_ISSUER` | `oidc_issuer` | **Required.** The oCIS public URL, which is also the OIDC issuer |
| `GROVE_OIDC_AUDIENCE` | `oidc_audience` | Optional. If set, tokens must carry it in `aud`; oCIS sets that to the client ID of the requesting app, e.g. `web` |
| `GROVE_OIDC_CA_FILE` | `oidc_ca_file` | Optional PEM bundle of extra CAs to trust when contacting the issuer |
| `GROVE_DB_DSN` | `db_dsn` | Not used yet |

## Authentication

Every request needs an `Authorization: Bearer <access token>` header, validated
against the keys the oCIS identity provider publishes. The exceptions are
`/healthz`, `/metrics`, `/.well-known/*` and `/web/*`. The token's `sub` claim
identifies the user.

Grove contacts the issuer on the first authenticated request, and retries after
a failure, so it can start before oCIS is ready. While the issuer is
unreachable, authenticated requests get `503`.

## Docker

```sh
task docker:build
task docker:run
```

## Local stack

`docker-compose.yml` runs oCIS and Grove behind one Traefik on the same
domain, which Grove requires. Grove's DAV, Graph, `/web/` and ops paths route
to Grove; everything else goes to oCIS.

```sh
task stack:up
```

Then open <https://ocis.localhost> (`admin` / `admin`). A `certs` service generates a
self-signed certificate on each start. Your browser will warn about it; Grove
trusts it through `GROVE_OIDC_CA_FILE`.
Set `OCIS_DOMAIN`, `OCIS_VERSION` or `IDM_ADMIN_PASSWORD` in the environment
or a `.env` file to override the defaults. This stack is for development only.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Please also read the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Security

See [SECURITY.md](SECURITY.md) for how to report vulnerabilities.

## License

[Apache License 2.0](LICENSE).
