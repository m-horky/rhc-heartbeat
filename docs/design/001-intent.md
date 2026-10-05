# rhc-heartbeat

Red Hat Enterprise Linux heartbeat software. The intent is to provide high-quality temporal data so the cloud services can reliably tell how long the system has been on.

## Architecture

- Go application
- The OpenTelemetry stream contains heartbeats (temporal data: system identification, monotonic time, wall-clock time, and type)
- System heartbeats consist of:
  - RHSM UUID and ORG ID (extracted from /etc/pki/consumer/cert.pem)
  - /proc/sys/kernel/random/boot_id (owned by the kernel)
  - CLOCK_MONOTONIC (to tell how long the system has been on)
  - CLOCK_REALTIME (to tell when the system has been on)
- `ping` heartbeats are sent periodically. `off` reports that updates are not expected to be possible for the foreseeable future (including an expected pause such as sleep); it describes update availability, not a shutdown event. `on` reports that the system is able to send updates.
- If the network is unavailable, retain heartbeats locally for up to 72 hours and upload them when connectivity returns.
- Scope: RHEL 8, 9, and 10. Minimize system impact. Log user-actionable warnings and unrecoverable errors to stderr for systemd collection; timers and services must retry after failures.
- Must be capable of reading `/etc/rhsm/rhsm.conf` to determine API path (based on the value of Candlepin URL) and proxy configuration to use for the HTTP transport.

## Network API contract

- We target Observatorium API.
- Use `/etc/pki/consumer/{cert,key}.pem` for mTLS; the server must validate the client certificate to authorize it.

## Internal API contract

Here are some drafts of how the package is structured internally. There should be a strong distinction between internal object representation and the wire-format DTO.

```go
type Identity struct {
  UUID  string
  OrgID string
}
type Heartbeat struct {
  Identity Identity
  BootID   string
  WallTime time.Time
  BootTime time.Duration
  MonoTime time.Duration
  Kind     string // one of 'on', 'off', or 'ping'; defaults to 'ping'
}
```
