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

The integration suite runs with `tmt`.

```shell
$ make test-local      # locally built binary
$ make test-container  # RPMs in a CentOS Stream container
$ make test-vm         # RPMs in a CentOS Stream VM
```

See [`docs/TESTING.md`](docs/TESTING.md) for plan selection, Prometheus setup and debugging.

## License

This project is licensed under the GNU General Public License version 3 only. See [`LICENSE`](LICENSE) for the full license text.
