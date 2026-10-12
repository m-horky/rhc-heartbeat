# Integration testing

- Run `make server` to start a (blocking) shell with Prometheus Remote Write available at `http://10.88.0.1:9090/api/v1/write`.
- To simulate registration with subscription-manager, run `bash test/scripts/identity.sh [-f] [dir=.]` to generate a `key.pem` and `cert.pem` in the supplied directory.
- Gherkin tests are stored under `test/features/`.
- To query Prometheus, call `bash test/scripts/query.sh [-q matcher]... [-l limit=1]`. Use `-h` to see the options.

Full setup and execution might look like so:

```shell
# Test system setup
$ nohup make server >/dev/null 2>&1 &

# Test setup
$ ROOT=$(mktemp -d)
$ mkdir -p ${ROOT}/identity ${ROOT}/etc-rhsm ${ROOT}/etc-rhc ${ROOT}/var-lib-rhc
$ cat > ${ROOT}/etc-rhc/rhc-heartbeat.conf <<EOF
[http.proxy]
uri = ""
[api.heartbeat]
uri = "http://10.88.0.1:9090/api/v1/write"
EOF
$ IDENTITY=$(bash test/scripts/identity.sh -f ${ROOT}/identity)

# Execution setup
$ export RHC_HEARTBEAT_CONFIG=${ROOT}/etc-rhc/rhc-heartbeat.conf
$ export RHC_HEARTBEAT_DROPINS=${ROOT}/etc-rhc/absent
$ export RHC_HEARTBEAT_RHSM_CONFIG=${ROOT}/etc-rhsm/rhsm.conf
$ export RHC_HEARTBEAT_PENDING_CACHE=${ROOT}/var-lib-rhc/heartbeat.jsonl
$ export RHC_HEARTBEAT_LOCK=${ROOT}/rhc-heartbeat.lock
$ export RHC_HEARTBEAT_CONSUMER_CERT=${ROOT}/identity/cert.pem
$ export RHC_HEARTBEAT_CONSUMER_KEY=${ROOT}/identity/key.pem

# Execute
$ ./build/rhc-heartbeat
$ echo $?
$ bash test/scripts/query.sh -q "host_uuid=\"${IDENTITY}\""

# Shut down the test server
$ podman stop prometheus
$ rm -rf ${ROOT}
```
