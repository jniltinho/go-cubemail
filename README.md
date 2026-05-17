# Go CubeMail

Go CubeMail is a lightweight and fast webmail client built with Go. It connects directly to your existing mail servers via standard IMAP and SMTP protocols, providing a clean web interface to manage your emails.

## Features

*   **Fast and Lightweight:** Built with Go for high performance and low resource usage.
*   **Standard Protocols:** Uses standard IMAP for reading emails and SMTP for sending.
*   **Web-based UI:** Modern, responsive web interface.
*   **Easy Configuration:** Configurable via `config.toml` and environment variables.

## Getting Started

### Prerequisites

*   Go 1.21 or higher

### Building from source

1.  Clone the repository:
    ```bash
    git clone <repository-url>
    cd go-cubemail
    ```
2.  Build the binary:
    ```bash
    make build
    # or
    go build -o go-cubemail
    ```

## Configuration

Configuration is managed via a TOML file. By default, the application looks for `config.toml` in the current directory or `/etc/go-cubemail/`. 

Example `config.toml`:

```toml
[server]
port = 8080

[imap]
host = "imap.example.com"
port = 993
tls = true

[smtp]
host = "smtp.example.com"
port = 587
tls = true
```

You can also use environment variables prefixed with `GORC_` (e.g., `GORC_SERVER_PORT=8080`).

## Usage

Start the server:

```bash
./go-cubemail
```

Or specify a custom configuration file:

```bash
./go-cubemail --config /path/to/your/config.toml
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.
