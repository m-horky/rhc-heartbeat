# Execution steps

- [x] Configuration
  - [x] Parse rhsm.conf
  - [x] Convert rhsm.conf into a public-facing Config object with inferred content
  - [x] Allow environment variables to override rhsm.conf
- [ ] Heartbeat fields
  - [ ] Read consumer certificate to obtain UUID and ORG ID
  - [ ] Read boot ID
  - [ ] Read monotonic and wall-clock time
  - [ ] Call chrony and parse its output
  - [ ] Construct the Heartbeat object
- [ ] Profile
  - [ ] Read product certificate to obtain ID
  - [ ] Read sub-man facts output to obtain the system profile
