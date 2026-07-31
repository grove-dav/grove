# grove

Calendar and contacts for ownCloud Infinite Scale — CalDAV/CardDAV, Graph-shaped API, Vue UI

## Development

Requires Go 1.26.5.

```sh
go build ./...
go test ./...
go run ./cmd/grove
```

Config is env vars (`GROVE_HTTP_ADDR`, `GROVE_LOG_LEVEL`, `GROVE_OIDC_ISSUER`,
`GROVE_DB_DSN`) and/or a YAML file passed via `--config grove.yaml`; env wins
over file, file wins over defaults.

`/healthz` and `/metrics` are served unauthenticated on `GROVE_HTTP_ADDR`
(default `:8080`).

## Docker

```sh
docker build -t grove:dev .
docker run --rm -p 8080:8080 grove:dev
```
