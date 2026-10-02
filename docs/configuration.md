# Configuration

The heartbeat reads `/etc/rhc/rhc-heartbeat.conf` as TOML and uses `/etc/rhsm/rhsm.conf` as a compatibility fallback. Override these paths with `RHC_HEARTBEAT_CONFIG` and `RHC_HEARTBEAT_RHSM_CONFIG`, respectively. Either file may be absent. A present but unreadable or invalid file causes loading to fail.

Example application configuration:

```toml
[api.heartbeat]
# Overrides the endpoint derived from rhsm.conf.
uri = "https://telemetry.example.com/otel/v1/logs"
tls-verify = true
ca-path = ""

[http.proxy]
# An empty URI disables the RHSM-derived proxy.
uri = ""
user = ""
password = ""
```

The resolved configuration starts with a secure TLS default, applies supported `rhsm.conf` values, and then applies explicit heartbeat configuration values. Explicit empty strings and `false` values override legacy values.

From `rhsm.conf`, the loader uses:

- `[server] hostname` and `port` to build the Candlepin origin, then appends `/otel/v1/logs` to produce the OTLP/HTTP URI. The `[server] prefix` API path is ignored. For example, with hostname `satellite.example.com`, port `8443`, and prefix `/rhsm`, the endpoint is `https://satellite.example.com:8443/otel/v1/logs`.
- `[server] insecure` to set `api.heartbeat.tls-verify` to the inverse value.
- `[rhsm] repo_ca_cert` as the OTEL endpoint CA path.
- `[proxy] proxy_hostname`, `proxy_port`, `proxy_user`, and `proxy_password` as HTTP proxy defaults.

The heartbeat configuration overrides the derived OTEL URI and each proxy field independently. RHSM proxy URIs use the HTTPS scheme, matching Elk's existing compatibility behavior. Proxy passwords and configuration contents are not included in parser errors.
