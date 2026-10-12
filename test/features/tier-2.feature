Feature: Resilience and edge cases
    Background:
      Given /var/lib/rhc/heartbeat.jsonl is absent or empty
        And a Prometheus Remote Write server is available

    Scenario: Connectivity recovers after a failed upload
      Given RHSM identity certificate pair is present
        And the Remote Write endpoint is unavailable
       When rhc-heartbeat is run
       Then /var/lib/rhc/heartbeat.jsonl contains the pending heartbeat
       When the Remote Write endpoint becomes available
        And rhc-heartbeat is run
       Then the pending heartbeat is uploaded
        And /var/lib/rhc/heartbeat.jsonl is absent or empty

    Scenario: A queued heartbeat keeps its original identity after identity rotation
      Given RHSM identity certificate pair for UUID "identity-a" is present
        And a heartbeat for "identity-a" is pending because the endpoint is unavailable
       When the consumer identity changes to UUID "identity-b"
        And the Remote Write endpoint becomes available
        And rhc-heartbeat is run
       Then the queued heartbeat is uploaded with UUID "identity-a"
        And a subsequent heartbeat is uploaded with UUID "identity-b"
        And /var/lib/rhc/heartbeat.jsonl is absent or empty

    Scenario: The Remote Write endpoint rejects an upload
      Given RHSM identity certificate pair is present
        And the Remote Write endpoint responds with HTTP 5xx
       When rhc-heartbeat is run
       Then rhc-heartbeat reports an upload error
        And /var/lib/rhc/heartbeat.jsonl contains the pending heartbeat

    Scenario: Concurrent runs do not corrupt or lose heartbeats
      Given RHSM identity certificate pair is present
       When two rhc-heartbeat processes are run concurrently
       Then /var/lib/rhc/heartbeat.jsonl contains only valid heartbeat rows
        And every heartbeat accepted for upload is present in Prometheus
        And no heartbeat is uploaded more than once

    Scenario: Proxy authentication controls upload access
      Given RHSM identity certificate pair is present
        And the HTTP proxy requires authentication
       When rhc-heartbeat is run with valid proxy credentials
       Then the heartbeat is uploaded to Prometheus
        And /var/lib/rhc/heartbeat.jsonl is absent or empty
       When rhc-heartbeat is run with invalid proxy credentials
       Then rhc-heartbeat reports an upload error
        And /var/lib/rhc/heartbeat.jsonl contains the pending heartbeat

    Scenario: TLS verification uses the configured certificate authority
      Given RHSM identity certificate pair is present
        And the Remote Write endpoint uses TLS with a certificate signed by the configured authority
       When rhc-heartbeat is run
       Then the heartbeat is uploaded to Prometheus
       When the endpoint uses a certificate not trusted by the configured authority
        And rhc-heartbeat is run
       Then rhc-heartbeat reports a TLS error
        And /var/lib/rhc/heartbeat.jsonl contains the pending heartbeat

    Scenario: A drop-in configuration overrides the base configuration
      Given the base configuration points to a Remote Write endpoint
        And a drop-in configures a different Remote Write endpoint
       When rhc-heartbeat is run
       Then the heartbeat is sent to the endpoint selected by the documented configuration precedence

    Scenario: Invalid configuration does not discard a pending heartbeat
      Given RHSM identity certificate pair is present
        And the configuration contains an invalid Remote Write URI
       When rhc-heartbeat is run
       Then rhc-heartbeat reports a configuration or upload error
        And /var/lib/rhc/heartbeat.jsonl contains the pending heartbeat

    Scenario: A reboot starts a new monotonic-time sequence
      Given RHSM identity certificate pair is present
        And a heartbeat has been recorded for the current boot
       When the system reboots and rhc-heartbeat is run
       Then the new heartbeat has a different boot ID
        And its monotonic timestamp is interpreted within the new boot
        And both boot records are handled without losing pending heartbeats

    Scenario: A wall-clock adjustment does not reorder monotonic heartbeats
      Given RHSM identity certificate pair is present
        And a heartbeat has been recorded
       When the wall clock moves backward and rhc-heartbeat is run
       Then the new heartbeat has a later monotonic timestamp for the same boot
        And both heartbeat records are handled without loss
