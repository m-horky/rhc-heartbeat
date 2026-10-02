# Execution steps

- [x] Configuration
  - [x] Parse rhsm.conf
  - [x] Convert rhsm.conf into a public-facing Config object with inferred content
  - [x] Allow environment variables to override rhsm.conf
- [x] Heartbeat fields
  - [x] Read consumer certificate to obtain UUID and ORG ID
  - [x] Read kernel boot ID
  - [x] Read monotonic uptime and wall-clock time
  - [x] Call chrony and parse its output
  - [x] Construct the Heartbeat object
- [ ] Profile
  - [ ] Read product certificate to obtain ID
  - [ ] Read sub-man facts output to obtain the system profile
- [ ] OTLP client
  - [ ] Accept context.Context, resolve config.Config, and heartbeat.Heartbeat.
