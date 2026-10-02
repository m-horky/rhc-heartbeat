# rhc-heartbeat-as-a-product

- Package under `cmd/rhc-heartbeat/`.
- One-shot binary triggered by systemd.
- Whether the payload type is `ping` or `off` is controlled by TBD.
- With online connectivity, payload is uploaded to OTel.
  - The correct way to react to `partialSuccess` is TBD.
- Without online telemetry, payload is cached at TBD.
  - On next invocation, both past and current heartbeat is attempted.
  - At least 72 hours worth of offline heartbeats must be kept.
  - Only send TBD heartbeats at a time.
