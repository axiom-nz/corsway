# corsway

## Overview

**corsway** is a free-to-use CORS proxy that adds CORS headers to your requests. This service allows you to bypass the Same-Origin Policy and make requests to external APIs without facing CORS issues.

## How to Use

Simply prepend your desired URL with `https://[your-corsway-host]/`. 


### Example

If you want to proxy a request to `https://example.com/api/data`, you would use the following URL:



````

https://[your-corsway-host]/https://example.com/api/data

````

## Features

- **Free to Use**: No subscription or payment required.
- **Reliable**: Built in Go to handle a large number of requests with high reliability.

## Use Cases

- Accessing third-party APIs that do not support CORS.
- Developing web applications that require data from multiple sources.
- Testing APIs during development without dealing with CORS restrictions.

## Contributing

We welcome contributions! Please open an issue or submit a pull request with your enhancements.

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE)

This project includes code from [cors.lol](https://github.com/BradPerbs/cors.lol), licensed under the [MIT License](LICENSE).
