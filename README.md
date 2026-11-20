# URL Shortener

An HTTP service for short links. It stores a URL under a short alias and redirects requests for that alias to the original URL.

Stack: Go, chi, PostgreSQL. Layers: the handler parses requests and builds responses, the service validates URLs and aliases, and the storage layer runs SQL.

## Getting Started

You need Docker and Docker Compose.

```bash
cp .env.example .env
```

Fill in the values in `.env`. `APP_ENV` selects the config file `config/<APP_ENV>.yaml`. The repository includes `local` and `prod`.

```bash
docker compose up --build
```

The service listens on `http://localhost:8082`. Postgres is exposed on `localhost:5432`.

The Basic Auth username comes from `http_server.user` in the YAML config (`admin` in `config/local.yaml`). The password comes from `HTTP_SERVER_PASSWORD` in `.env`.

The Postgres image applies `POSTGRES_PASSWORD` only when the `pgdata` volume is first created. Changing `POSTGRES_PASSWORD` in an existing volume does not change the database password. To apply a new password to an empty database, run `docker compose down -v` and then `docker compose up --build` again.

## API

Create, update, and delete require Basic Auth. Redirect is public.

### `POST /url` — create a short link

Returns `201 Created`.

```bash
curl -u admin:password -X POST http://localhost:8082/url \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com","alias":"ex"}'
```

`alias` is optional. If it is omitted, the service generates a random 6-character alias.

```json
{"status":"OK","alias":"ex"}
```

### `PUT /url` — change the URL of an existing alias

Returns `200 OK`.

```bash
curl -u admin:password -X PUT http://localhost:8082/url \
  -H 'Content-Type: application/json' \
  -d '{"alias":"ex","url":"https://example.org"}'
```

### `DELETE /url/{alias}` — delete a short link

Returns `200 OK`.

```bash
curl -u admin:password -X DELETE http://localhost:8082/url/ex
```

### `GET /{alias}` — redirect

Returns `302 Found` with the original URL in the `Location` header.

```bash
curl -i http://localhost:8082/ex
```

### Error Codes

| Situation | Status |
|---|---|
| Missing or invalid credentials | 401 |
| Malformed JSON, empty or invalid URL, empty alias | 400 |
| Alias already taken | 409 |
| Alias not found | 404 |
| Database error | 500 |

Error body: `{"status":"Error","error":"..."}`.

Only `http` and `https` URLs are accepted.

## Tests

Unit tests for handlers and the service do not need a running server:

```bash
go test ./internal/...
```

The end-to-end tests in `tests/` call `localhost:8082`. Start the application before running them:

```bash
go test ./tests/
```

They read the password from `.env` and the username from the same YAML config as the service.
