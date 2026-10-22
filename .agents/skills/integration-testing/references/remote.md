# Remote execution environment (SSH)

Suite runs on disposable remote VM supplied by operator: torn down after tests, so agent may freely manage it: start/stop Prometheus, adjust system clock (e.g. `timedatectl set-time`), reboot. No cleanup needed afterward.

## SSH access

Operator must supply SSH host (IP or hostname). If none supplied, ask first.

Verify access before anything else, e.g. `ssh <host> rpm -q podman rhc-heartbeat`. If access fails (name resolution, connection refused, auth failure), report to operator and stop; do not guess hosts, usernames, credentials. If Podman or rhc-heartbeat missing, report and stop; do not invent solutions.

## Prometheus

Port 9090 reserved for Cockpit; host Prometheus on 9000. Copy `test/prometheus.yml` to VM, mirror the `server` target from the `Makefile`, publish port 9000:

    podman run -d --rm --name prometheus \
        -p "0.0.0.0:9000:9090/tcp" \
        -v "prometheus:/prometheus" \
        -v ./prometheus.yml:/etc/prometheus/prometheus.yml:ro,Z \
        docker.io/prom/prometheus:latest \
        --config.file=/etc/prometheus/prometheus.yml \
        --web.enable-remote-write-receiver

Prometheus reachable at `http://<host>:9000` from agent machine (subject to VM firewall) and `http://localhost:9000` on VM. Verify before scenarios, e.g. `curl -fsS http://<host>:9000/-/healthy`; report to operator if inaccessible.

## Identity

If remote system unregistered with subscription-manager, generate identity cert pair: copy `test/scripts/identity.sh` to VM, run there (e.g. `bash identity.sh -f /etc/pki/consumer`), or run locally and copy `key.pem` and `cert.pem` to VM.

## Configuration and execution

Edit/create `/etc/rhc/rhc-heartbeat.conf` pointing at custom Prometheus location:

    [http.proxy]
    uri = ""
    [api.heartbeat]
    uri = "http://localhost:9000/api/v1/write"

Query Prometheus: `bash test/scripts/query.sh -u http://<host>:9000/api/v1/query` from agent machine. If VM firewall blocks port, open it (`firewall-cmd --add-port=9000/tcp`) or copy `test/scripts/query.sh` to VM and query `http://localhost:9000/api/v1/query` there.

## Environment pitfalls

**Proxy environment variables leak into helpers.** SSH sessions may carry `HTTP_PROXY`/`HTTPS_PROXY` pointing at remote proxy. Heartbeat ignores them (own config), but Python's `urllib` and curl (uppercase `HTTPS_PROXY` for https URLs) honor them. Run every helper with scrubbed environment:

    env -u HTTP_PROXY -u HTTPS_PROXY -u http_proxy -u https_proxy python3 <helper>.py

and `curl --noproxy "*"` for manual checks. Symptom if forgotten: helper "forwards" time out with `TimeoutError` while direct curl succeeds.

**Proxy scenarios need non-loopback endpoint.** Go's `httpproxy` package hardcodes bypass for `localhost` and loopback IPs regardless of configured proxy. For proxy testing, point `api.heartbeat.uri` at VM's external IP or Prometheus container IP (`podman inspect prometheus --format '{{.NetworkSettings.IPAddress}}'`, port 9090 inside container). Symptom if forgotten: "using proxy" logged but upload goes direct and succeeds, silently invalidating scenario.

Pre-create systemd oneshot unit for Prometheus, enable before testing:

    [Unit]
    Description=Prometheus remote write receiver (test)
    Wants=network-online.target
    After=network-online.target

    [Service]
    Type=oneshot
    RemainAfterExit=yes
    ExecStart=/usr/bin/podman run -d --replace --name prometheus -p 127.0.0.1:9000:9090 -v /root/prometheus.yml:/etc/prometheus/prometheus.yml:ro,Z docker.io/prom/prometheus:latest --config.file=/etc/prometheus/prometheus.yml --web.enable-remote-write-receiver
    ExecStop=/usr/bin/podman stop prometheus

    [Install]
    WantedBy=multi-user.target

**Missing or broken tools.** `bc` may be missing (use shell arithmetic), `setools-console` uninstalable on unregistered VM (analyze AVCs from raw audit log instead), `date -d` misparses some formats (pass numeric epochs to Prometheus range queries instead).

**`pkill -f` kills the SSH session.** Instead, capture PID at launch (`nohup ... & echo $! > pidfile; kill $(cat pidfile)`, or match string unique to process (`pkill -f "python3 /root/http500.py"`).

** `cmd | tee` masks exit codes, `echo $?` prints 0. Capture `${PIPESTATUS[0]}`, or avoid pipes when asserting exit codes.

**Quick Prometheus liveness check.** Empty POST to `/api/v1/write` must return HTTP 400. Any HTTP response means receiver live.

## Verifying uploads in Prometheus

- In instant query responses, `value[0]` is *evaluation* time, not sample timestamp. Use `timestamp(<selector>)` for real sample timestamp.
- Heartbeats with identical labels collapse into one series; instant query returns only newest sample. Use `count_over_time(<selector>[15m])` for raw sample counts, range queries for history.
- Sample timestamp equals heartbeat row's `TimeUnix`; sample value is `TimeMonotonic` in seconds.
- Prometheus rejects backdated samples with HTTP 400 (out of order). Expected in wall-clock-backward scenario; client should report error and keep pending row.
- Prefer `jq -r` when feeding query results into shell arithmetic, else quotes break `$((...))`.

## Configuration discovery

- `/usr/libexec/rhc/heartbeat config` prints resolved configuration; use it to discover valid keys instead of guessing. Unknown keys in `rhc-heartbeat.conf` silently ignored.
- TLS CA configurable via `[api.heartbeat] ca-path`. Default comes from `rhsm.conf` `[rhsm] repo_ca_cert`.
- `identity.sh` prints UUID on stdout; certificate's `O=` subject becomes `host_org` label.
- systemd's `ProtectHome=yes` makes `/root` invisible to service. systemd-driven scenarios must use real system paths (`/etc/rhc/rhc-heartbeat.conf`, `/etc/pki/consumer`, `/var/lib/rhc`), not sandbox root under `/root`.

## systemd and SELinux

- Sandboxed runs with `RHC_HEARTBEAT_*` environment overrides execute as `unconfined_t`, never exercise SELinux policy. Only systemd-started runs enter `rhc_heartbeat_t`. To attribute degraded system profile to policy, compare confined service run with unconfined manual run over same system paths.
- AVC denials may be missing from `ausearch -m avc` output and never in `journalctl -k` (auditd consumes them). Grep raw log instead:

      grep "type=AVC" /var/log/audit/audit.log | grep "rhc_heartbeat_t" \
        | sed 's/.*avc:  denied  \(.*\) for  pid=[0-9]* comm="[^"]*" name="\([^"]*\)".*tclass=\([^ ]*\).*/denied {\1} name=\2 tclass=\3/' \
        | sort | uniq -c

- After intentional service failure, promptly run `systemctl stop rhc-heartbeat.service && systemctl reset-failed rhc-heartbeat.service`. Else systemd re-runs service and mutates pending cache.

**Manual runs mislabel the pending cache.** `/var/lib/rhc/heartbeat.jsonl` created unconfined is `var_lib_t` (expected: `rhc_heartbeat_cache_t`). Run `restorecon /var/lib/rhc/heartbeat.jsonl` to fix.

**Check journal for unit warnings on first service start.**

## Reboot testing

1. Enable timer: `systemctl enable --now rhc-heartbeat.timer` (fires 30 s after boot, then every 10 minutes).
2. Queue pending heartbeat for current boot with one failed service run (dead endpoint in `/etc/rhc/rhc-heartbeat.conf`), then stop and reset-failed the unit so restart loop cannot upload early.
3. Restore working endpoint config, ensure Prometheus unit above enabled, `systemctl reboot`.
4. Poll SSH in loop until VM answers; compare `/proc/sys/kernel/random/boot_id` before/after, query Prometheus per `boot_id` label to verify both boots' records.

## Process

- Tee every scenario into per-scenario logs under test root on VM (e.g. `/root/hb-test/logs/tN-sM-*.log`); measurement-tool quirks often force re-checking earlier evidence.
- Wrap verifications with before/after sample counts, assert expected delta.
- If result looks wrong, distrust measurement tooling first (evaluation time vs sample time, series collapsing, AVC visibility have all caused false alarms), then re-run with precisely timed fresh run (`date +%s.%N` before/after) and narrowed query window.
