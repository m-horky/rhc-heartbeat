Feature: System integration
    Background:
      Given rhc-heartbeat is installed with its systemd units and SELinux policy
        And /var/lib/rhc/heartbeat.jsonl is absent or empty
        And a Prometheus Remote Write server is available

    Scenario: The service uploads a heartbeat and finishes successfully
      Given RHSM identity certificate pair is present
       When `systemctl start rhc-heartbeat.service` is run
       Then the rhc-heartbeat.service unit finishes successfully
        And the heartbeat is uploaded to Prometheus
        And /var/lib/rhc/heartbeat.jsonl is absent or empty

    Scenario: The service fails while the Remote Write endpoint is unavailable
      Given RHSM identity certificate pair is present
        And the Remote Write endpoint is unavailable
       When `systemctl start rhc-heartbeat.service` is run
       Then the rhc-heartbeat.service unit enters the failed state
        And the unit schedules an automatic restart
        And /var/lib/rhc/heartbeat.jsonl contains the pending heartbeat

    Scenario: The service fails while the system is not registered
      Given RHSM identity certificate pair is absent
       When `systemctl start rhc-heartbeat.service` is run
       Then the rhc-heartbeat.service unit enters the failed state
        And /var/lib/rhc/heartbeat.jsonl contains a single row

    Scenario: The timer runs the service after boot and on every interval
      Given RHSM identity certificate pair is present
        And the rhc-heartbeat.timer unit is enabled
       When the system boots
       Then the rhc-heartbeat.service unit runs shortly after boot
        And the heartbeat is uploaded to Prometheus
       When the timer interval elapses
       Then the rhc-heartbeat.service unit runs again
        And every run uploads a heartbeat to Prometheus

    Scenario: The online report unit uploads a heartbeat of kind "on"
      Given RHSM identity certificate pair is present
       When `systemctl start rhc-heartbeat-on.service` is run
       Then the rhc-heartbeat-on.service unit finishes successfully
        And Prometheus instance has a record with kind "on"

    Scenario: The service runs without SELinux violations
      Given RHSM identity certificate pair is present
       When `systemctl start rhc-heartbeat.service` is run
       Then the audit log contains no SELinux denials for the rhc-heartbeat domain
        And the heartbeat system profile is collected
        And the uploaded heartbeat contains the system profile facts

    Scenario: A reboot starts the service and preserves queued heartbeats
      Given RHSM identity certificate pair is present
        And /var/lib/rhc/heartbeat.jsonl contains a pending heartbeat for the current boot
        And the rhc-heartbeat.timer unit is enabled
       When the system reboots
       Then the rhc-heartbeat.service unit runs shortly after boot
        And the new heartbeat has a different boot ID
        And its monotonic timestamp is interpreted within the new boot
        And the pending heartbeat from the previous boot is uploaded
        And /var/lib/rhc/heartbeat.jsonl is absent or empty
