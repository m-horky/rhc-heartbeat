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

The RPM installs the command at `/usr/libexec/rhc/heartbeat`:

```shell
/usr/libexec/rhc/heartbeat run
/usr/libexec/rhc/heartbeat config
/usr/libexec/rhc/heartbeat profile
```

Format Go code with `make fmt`.

## Testing

For local end-to-end testing, Podman is required to run Prometheus with its Remote Write receiver enabled.

```shell
$ mkdir -p /var/lib/rhc
$ make server &
$ make build >/dev/null && RHC_HEARTBEAT_CONFIG=test/dev.toml ./build/heartbeat
```

A TLS certificate keypair must be present. Either register with subscription-manager, or generate one with openssl:

```shell
mkdir -p /etc/pki/consumer
./test/scripts/identity.sh /etc/pki/consumer
```

## License

This project is licensed under the GNU General Public License version 3 only. See [`LICENSE`](LICENSE) for the full license text.
