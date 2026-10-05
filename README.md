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

For local end-to-end testing, Podman is required to run the OpenTelemetry Collector used by the test setup.

```shell
$ make server &
$ make build >/dev/null && RHC_HEARTBEAT_CONFIG=test/dev.toml ./build/rhc-heartbeat
```

## License

This project is licensed under the GNU General Public License version 3 only. See [`LICENSE`](LICENSE) for the full license text.
