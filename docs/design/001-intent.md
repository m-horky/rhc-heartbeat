# rhc-heartbeat

Red Hat Enterprise Linux heartbeat software. The intent is to provide high-quality temporal data so the cloud services can reliably tell how long the system has been on.

## Architecture

- Go application
- Two OpenTelemetry streams: heartbeats (temporal data: system identification, monotonic time, network time, network time quality, type), and facts (CPU topology, /etc/os-release values, installed RHEL products)
- System heartbeats consist of:
  - RHSM UUID and ORG ID (extracted from /etc/pki/consumer/cert.pem)
  - /proc/sys/kernel/random/boot_id (owned by the kernel)
  - CLOCK_MONOTONIC_RAW (to tell how long the system has been on)
  - CLOCK_REALTIME (to tell when the system has been on)
  - Time quality (chronyc -c tracking output, collected on each run)
    - `clock_synchronized: true` based on the leap status value.
    - `clock_distance_seconds:` chrony’s root distance, calculated as root dispersion + root delay / 2.
- System facts consist of:
  - RHSM UUID and ORG ID
  - BIOS UUID (extracted by dmi or by hand)
  - CPU topology (architecture, core count (including hyperthreading), socket count)
  - Product IDs (extracted from /etc/pki/product-default and /etc/pki/product)
  - System name and version (/etc/os-release)
  - System purpose (owned by `subscription-manager syspurpose` after being configured by the administrator)
  - Virtualization state (reported by `virt-what`)
  - Marketplace information (instance ID, marketplace name, account ID; reported by the IMDS servers)
- System facts are collected on each run, emitted when they differ or when the upload was over 24 hours ago.
- Heartbeats are sent every ten minutes and on shutdown. Nothing is sent during sleep.
- If the network is unavailable, retain heartbeats locally for up to 72 hours and upload them when connectivity returns.
- Scope: RHEL 8, 9, and 10. Minimize system impact. Log user-actionable warnings and unrecoverable errors to stderr for systemd collection; timers and services must retry after failures.
- Must be capable of reading `/etc/rhsm/rhsm.conf` to determine API path (based on the value of Candlepin URL) and proxy configuration to use for the HTTP transport.

## Network API contract

- Use `POST /v1/logs` HTTP transport (OTLP's JSON mapping with `application/json`).
- Use `/etc/pki/consumer/{cert,key}.pem` for mTLS; the server must validate the client certificate to authorize it.
- Two event names: `heartbeat` and `profile`.
- Fiels are typed attributes. Heartbeat's wall time is OTLP log timestamp.

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
  Uptime   time.Duration
  WallTime time.Time
  Chrony   ChronyStatus
  Trigger  Trigger // 'ping' or 'off' enum
}
type Profile struct {
  Identity Identity
  CPU      CPUFacts
  OS       OSFacts
  Products []int
}
```
