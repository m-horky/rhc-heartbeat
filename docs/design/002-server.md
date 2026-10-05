# Server design

We are effectively dealing with fault-tolerant computing and eventual accuracy. There are several concepts in play:

- **Completeness:** a genuinely off system is eventually detected.
- **Accuracy:** a healthy system is not suspected.
- **Eventual accuracy:** a healthy system may be temporarily suspected.
- **Event time processing**: discrete events are converted into continual metric.

## Ingestion

Since heartbeat collection is a fault-sensitive responsibility, the actual ingestion is performed by Red Hat hosted [Observatorium](https://github.com/observatorium) acting as an API gateway in front of [Thanos](https://thanos.io/). RHOBS allows clients to use the [Prometheus remote write protocol](https://prometheus.io/docs/specs/prw/remote_write_spec/) to upload their data.

### Payload

The host includes the following information as its heartbeat:

- RHSM UUID
- RHSM ORG ID
- Boot ID
- Monotonic time (does not include suspend time)
- Boot time (includes suspend time)
- Kind ("ping", “on”, "off")

The Observatorium additionally injects the ingestion time.

## Interval reconstruction

### The Dying Gasp

If no subsequent `ping` heartbeats arrive, and `off` or `sleep` are not received either, the business logic adds additional time on top of the last reported monotonic time. We assume the system has been shut off sometime in that interval period, and so we charge for it.

To keep the implementation (and explanation to the customers) simple, supportable and maintainable, the 10 minute constant is only included if the host has been online for less than ten minutes.

(The problem comes with tracking systems that have been restarted. A `ping` at 10:08 with uptime of 15 hours followed by a `on` at 10:10 means that we either double-bill for the overlap of the dying gasp of the previous boot, or that we have to retroactively change the default dying gasp value based on the requests we receive in the future.

UUID-based deduplication may also incorrectly erase incorrectly cloned VMs and make them be perceived as a single host.)

## Stateless host lifetime tracking

Since the systems already report the monotonic time, which excludes the time the system has been suspended (frozen VMs etc), we already have everything we need to bill for that system.

In this context, the backend needs to determine how much the systems have consumed within a given time period.

| UUID | BOOT\_ID | TIME SINCE BOOT | MONOTONIC TIME | TYPE | INGESTION TIME | TOTAL TIME |
| ----: | ----: | ----: | ----: | ----: | ----: | ----: |
| abc | 12 | 1:50:23 | 1:50:23 | ping | 09:19:42 | 6623 |
| cde | 45 | 784:20:02 | 557:47:12 | off | 09:19:44 | 2008032 |
| abc | 13 | 0:08:12 | 0:08:12 | ping | 09:29:46 | 492 |
| cde | 46 | 0:00:54 | 0:00:54 | ping | 09:29:48 | 54 |

Without state reconstruction, deferred heartbeats are not required at all. Instead of keeping 6\*24 heartbeat signals per day per machine, we only need one \-- the latest one. The only state tracking is the "the dying gasp" of freshly booted systems.
