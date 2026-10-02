## Testing

```shell
$ make server &
$ make build >/dev/null && RHC_HEARTBEAT_CONFIG=test/dev.toml ./build/rhc-heartbeat
```
