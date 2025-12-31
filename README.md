# corsway

A simple CORS proxy written in Go. Append any URL to your corsway instance and get back a CORS-enabled response.

A free hosted version is available at [2677929.xyz](https://2677929.xyz/).

## Usage

Prepend the corsway host to any URL:

```bash
https://[your-corsway-host].com/https://api.example.com/data

# or 

https://proxy.2677929.xyz/https://api.example.com/data
```

The proxy manages the CORS handshake and forwards the request transparently.

## Usage
`just test` will run all unit tests.

`just build` will run all unit tests and build the binary.

`just build-docker` run all unit tests and create a docker image.

`just run` will run the last built binary.

`just run-docker` will run the docker image.

### Running
With default configuration, exposed on port 8080:

#### Docker
```bash
just build-docker
just run-docker

# or

docker build -t axiom-nz/corsway .
docker run -p 8080:8080 axiom-nz/corsway
```

#### Local

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

| Flag                 | Env Var             | Default | Description                                                     |
|----------------------|---------------------|---------|-----------------------------------------------------------------|
| `-port`              | `PORT`              | 8080    | Server listening port                                           |
| `-whitelist`         | `WHITELIST`         | (none)  | Comma-separated list of allowed origins                         |
| `-rate-limit`        | `RATE_LIMIT`        | 20      | Max requests per IP per window                                  |
| `-rate-limit-window` | `RATE_LIMIT_WINDOW` | 5m      | Duration of rate limit window (e.g. 1m, 1h)                     |
| `-max-request-bytes` | `MAX_REQUEST_BYTES` | 10MB    | Maximum allowed request body size                               |
| `-trust-proxy`       | `TRUST_PROXY`       | false   | Trust X-Forwarded-For headers (use only behind a reverse proxy) |

### Examples:

```bash
just run 8080 --rate-limit=50 --rate-limit-window=1m --whitelist=https://myapp.com,https://staging.myapp.com --trust-proxy
```

## Features

*   **CORS Proxy:** Adds required `Access-Control-Allow-*` headers to responses.
*   **Origin Control:** Optional `Origin` allowlist to restrict access to specific domains.
*   **Rate Limiting:** IP-based limiting to prevent abuse.
*   **Proxy Support:** Configurable `X-Forwarded-For` support for deployments behind reverse proxies.
*   **Request Safety:** Enforces limits on request body size.
*   **Normalisation:** Handles common malformed URL patterns.

## Contributing

Contributions are welcome. Please open an issue or submit a pull request for any improvements.

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE).

`corsway` includes code adapted from [BradPerbs/cors.lol](https://github.com/BradPerbs/cors.lol), licensed under the MIT License.
