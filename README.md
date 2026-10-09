# rhc-heartbeat

`rhc-heartbeat` is a proof of concept of a service for collecting and reporting system heartbeat data.

## Documentation

See [`docs/`](docs/) for [architecture](docs/ARCHITECTURE.md) or [packaging instructions](docs/PACKAGING.md).

## Development

The project uses Go. The Makefile's version lookup also requires `rpmspec`. To build the application and run the test and lint checks:

```shell
make build
make check
```

Format Go code with `make fmt`.

## Testing

For local end-to-end testing, Podman is required to run Prometheus with its Remote Write receiver enabled.

```shell
$ mkdir -p /var/lib/rhc
$ make server &
$ make build >/dev/null && RHC_HEARTBEAT_CONFIG=test/dev.toml ./build/rhc-heartbeat
```

A TLS certificate keypair must be present. Either register with subscription-manager, or generate one with openssl:

```shell
mkdir -p /etc/pki/consumer/
openssl req -x509 -newkey rsa:4096 -sha256 -nodes -days 7 -keyout /etc/pki/consumer/key.pem -out /etc/pki/consumer/cert.pem -subj "/O=20008437/CN=test-client" -addext "extendedKeyUsage=clientAuth"
```

## License

This project is licensed under the GNU General Public License version 3 only. See [`LICENSE`](LICENSE) for the full license text.
