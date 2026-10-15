# rhc-heartbeat SELinux policy

This package ships a SELinux policy module (`rhc_heartbeat`) for the rhc-heartbeat service, as `rhc-heartbeat-selinux`.

## `selinux/` layout

- `rhc_heartbeat.te`:
  - the `rhc_heartbeat_t` daemon domain,
  - the `rhc_heartbeat_rhsm_exec_t` wrapper type for the transition into `rhsmcertd_t`,
  - the `rhc_heartbeat_cache_t` state type, and their rules.
- `rhc_heartbeat.fc`: labels for the daemon binary, the RHSM wrapper, the systemd units, and the pending cache.
- `rhc_heartbeat.if`: interfaces for other modules (domtrans, exec, systemctl, admin).
- `Makefile`: wrapper around `/usr/share/selinux/devel/Makefile`.

## Developing

```shell
$ dnf install -y selinux-policy-devel
$ make -C selinux
```

On a test machine, the module can be loaded directly without packaging:

```shell
$ make -C selinux install
```

When denials are observed, run the command below. Add the resulting rules to `rhc_heartbeat.te`, rebuild, and repeat.

```shell
$ ausearch -m avc -ts recent | audit2allow -R
```

## Design notes

subscription-manager (natively `bin_t`) is run within the `rhsmcertd_t` domain (which is otherwise reserved for its daemon) so we don't have to duplicate its policy. We ship a wrapper, `heartbeat-rhsm`, that is labeled `rhc_heartbeat_rhsm_exec_t` and is allowed to transition into `rhsmcertd_t`.
