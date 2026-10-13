Feature: Basic use-cases
    Background:
      Given /var/lib/rhc/heartbeat.jsonl is absent or empty
        And a Prometheus Remote Write server is available

    Scenario: Before registration
      Given RHSM identity certificate pair is absent
       When `heartbeat run` is run
       Then /var/lib/rhc/heartbeat.jsonl contains a single row
       When `heartbeat run` is run
       Then /var/lib/rhc/heartbeat.jsonl contains a single row
        And the wall-clock time in that row is higher

    Scenario: System gets registered
      Given RHSM identity certificate pair is absent
       When `heartbeat run` is run
       Then /var/lib/rhc/heartbeat.jsonl contains a single row
       When `subscription-manager register` is run
       Then RHSM identity certificate pair is present
       When `heartbeat run` is run
       Then /var/lib/rhc/heartbeat.jsonl is absent or empty
        And Prometheus instance has a record with RHSM UUID

    Scenario: Network is misconfigured
      Given RHSM identity certificate pair is present
        And HTTP proxy is configured to a URI where proxy does not run
       When `heartbeat run` is run
       Then /var/lib/rhc/heartbeat.jsonl contains a single row
        And the RHSM identity and organization ID are present in the row
