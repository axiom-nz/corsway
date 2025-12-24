# corsway

A simple CORS proxy written in Go. Append any URL to your corsway instance and get back a CORS-enabled response.

## Usage

Prepend your corsway host to any URL:

```
https://[your-corsway-host].com/https://api.example.com/data
```

The proxy manages the CORS handshake and forwards the request transparently.

## Usage
With default configuration, exposed on port 8080:

### Docker
```bash
just build-docker
just run-docker

# or

docker build -t axiom-nz/corsway .
docker run -p 8080:8080 axiom-nz/corsway
```

### Local

```bash
just build
just run

# or

go build -o corsway cmd/corsway/main.go
chmod +x corsway
./corsway
```

## Configuration

Configuration is handled via command-line flags or environment variables. **Command-line flags take precedence over environment variables.**

| Flag                 | Env Var             | Default | Description                                 |
|----------------------|---------------------|---------|---------------------------------------------|
| `-port`              | `PORT`              | 8080    | Server listening port                       |
| `-whitelist`         | `WHITELIST`         | (none)  | Comma-separated list of allowed origins     |
| `-rate-limit`        | `RATE_LIMIT`        | 20      | Max requests per IP per window              |
| `-rate-limit-window` | `RATE_LIMIT_WINDOW` | 5m      | Duration of rate limit window (e.g. 1m, 1h) |
| `-max-request-bytes` | `MAX_REQUEST_BYTES` | 10MB    | Maximum allowed request body size           |

### Examples:

```bash
just run 8080 --rate-limit=50 --rate-limit-window=1m --whitelist=https://myapp.com,https://staging.myapp.com
# or
just run-docker 8080 --rate-limit=50 --rate-limit-window=1m --whitelist=https://myapp.com,https://staging.myapp.com
```


```bash
docker run -p 8080:8080 \
  -e RATE_LIMIT=50 \
  -e RATE_LIMIT_WINDOW=1m \
  -e WHITELIST=https://myapp.com,https://staging.myapp.com \
  axiom-nz/corsway
```


## Features

*   **CORS Proxy:** Adds required `Access-Control-Allow-*` headers to responses.
*   **Origin Control:** Optional `Origin` allowlist to restrict access to specific domains.
*   **Rate Limiting:** IP-based limiting to prevent abuse.
*   **Request Safety:** Enforces limits on request body size.
*   **Normalisation:** Handles common malformed URL patterns.

## Contributing

Contributions are welcome. Please open an issue or submit a pull request for any improvements.

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE).

It includes code adapted from [BradPerbs/cors.lol](https://github.com/BradPerbs/cors.lol), licensed under the MIT License.
