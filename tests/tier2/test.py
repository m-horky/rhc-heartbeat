"""Tier 2 integration tests: resilience, recovery and edge cases."""

from __future__ import annotations

import os
import sys
import uuid as uuid_module
from pathlib import Path

sys.path.insert(0, str(Path(os.environ["TMT_TREE"]) / "tests"))

from testlib.environment import guest_ipv4
from testlib.heartbeat import (
    assert_cache_valid,
    assert_pending_count,
    assert_pending_identity,
    run as run_heartbeat,
    spawn as spawn_heartbeat,
)
from testlib.prometheus import wait_for_series
from testlib.prometheus import write_url
from testlib.tls import generate as generate_tls
from testlib.case import TmtTestCase


class TestTier2(TmtTestCase):
    """Tier 2 integration tests."""

    def test_recovery(self) -> None:
        """Retry a failed upload after the endpoint recovers.

        Given RHSM identity certificate pair is present
        And the Remote Write endpoint is unavailable
        When `heartbeat run` is run
        Then /var/lib/rhc/heartbeat.jsonl contains the pending heartbeat
        When the Remote Write endpoint becomes available
        And `heartbeat run` is run
        Then a new heartbeat is uploaded
        And /var/lib/rhc/heartbeat.jsonl is absent or empty
        """
        ctx = self.ctx
        host_uuid = ctx.generate_identity()

        ctx.write_config("http://127.0.0.1:1/api/v1/write")
        run_heartbeat(ctx).assert_failure()
        assert_pending_count(ctx, 1)

        ctx.write_config(f"{ctx.prometheus_url}/api/v1/write")
        run_heartbeat(ctx).assert_success()
        assert_pending_count(ctx, 0)
        wait_for_series(ctx.prometheus_url, f'host_uuid="{host_uuid}"')

    def test_identity_rotation(self) -> None:
        """Keep the identity of a queued heartbeat when the consumer changes.

        Given RHSM identity certificate pair for UUID "identity-a" is present
        And a heartbeat for "identity-a" is pending because the endpoint is unavailable
        When the consumer identity changes to UUID "identity-b"
        And the Remote Write endpoint becomes available
        And `heartbeat run` is run
        Then the queued heartbeat is uploaded with UUID "identity-a"
        And a subsequent heartbeat is uploaded with UUID "identity-b"
        And /var/lib/rhc/heartbeat.jsonl is absent or empty
        """
        ctx = self.ctx
        uuid_a = str(uuid_module.uuid4())
        uuid_b = str(uuid_module.uuid4())

        ctx.generate_identity(uuid_a)
        ctx.write_config("http://127.0.0.1:1/api/v1/write")
        run_heartbeat(ctx).assert_failure()
        assert_pending_count(ctx, 1)
        assert_pending_identity(ctx, uuid_a)

        ctx.generate_identity(uuid_b)
        ctx.write_config(f"{ctx.prometheus_url}/api/v1/write")
        run_heartbeat(ctx).assert_success()
        assert_pending_count(ctx, 0)

        wait_for_series(ctx.prometheus_url, f'host_uuid="{uuid_a}"')
        wait_for_series(ctx.prometheus_url, f'host_uuid="{uuid_b}"')

    def test_upload_rejected(self) -> None:
        """Keep the pending heartbeat when the endpoint rejects the upload.

        Given RHSM identity certificate pair is present
        And the Remote Write endpoint responds with HTTP 5xx
        When `heartbeat run` is run
        Then `heartbeat run` reports an upload error
        And /var/lib/rhc/heartbeat.jsonl contains the pending heartbeat
        """
        ctx = self.ctx
        host_uuid = ctx.generate_identity()
        ctx.start_helper("http500", 7001, "http500.py")
        ctx.write_config("http://127.0.0.1:7001/api/v1/write")

        run_heartbeat(ctx).assert_failure()
        assert_pending_count(ctx, 1)
        assert_pending_identity(ctx, host_uuid)

    def test_concurrency(self) -> None:
        """Keep the pending cache valid when two runs overlap.

        Given RHSM identity certificate pair is present
        When two `heartbeat run` processes are run concurrently
        Then /var/lib/rhc/heartbeat.jsonl contains only valid heartbeat rows
        """
        ctx = self.ctx
        ctx.generate_identity()
        ctx.write_config(f"{ctx.prometheus_url}/api/v1/write")

        first = spawn_heartbeat(ctx)
        second = spawn_heartbeat(ctx)
        first.wait()
        second.wait()

        assert_cache_valid(ctx)

    def test_proxy_auth(self) -> None:
        """Let the proxy accept or reject uploads based on credentials.

        Given RHSM identity certificate pair is present
        And the HTTP proxy requires authentication
        When `heartbeat run` is run with valid proxy credentials
        Then the heartbeat is uploaded to Prometheus
        And /var/lib/rhc/heartbeat.jsonl is absent or empty
        When `heartbeat run` is run with invalid proxy credentials
        Then `heartbeat run` reports an upload error
        And /var/lib/rhc/heartbeat.jsonl contains the pending heartbeat
        """
        ctx = self.ctx
        host_uuid = ctx.generate_identity()
        endpoint = write_url(ctx.prometheus_url, guest_ipv4())
        ctx.start_helper("authproxy", 7000, "authproxy.py")

        ctx.write_config(endpoint, proxy="http://good:pass@127.0.0.1:7000")
        run_heartbeat(ctx).assert_success()
        assert_pending_count(ctx, 0)
        wait_for_series(ctx.prometheus_url, f'host_uuid="{host_uuid}"')

        ctx.write_config(endpoint, proxy="http://bad:pass@127.0.0.1:7000")
        run_heartbeat(ctx).assert_failure()
        assert_pending_count(ctx, 1)
        assert_pending_identity(ctx, host_uuid)

    def test_tls_ca(self) -> None:
        """Verify the upload against the configured certificate authority.

        Given RHSM identity certificate pair is present
        And the Remote Write endpoint uses TLS with a certificate signed by the configured authority
        When `heartbeat run` is run
        Then the heartbeat is uploaded to Prometheus
        When the endpoint uses a certificate not trusted by the configured authority
        And `heartbeat run` is run
        Then `heartbeat run` reports a TLS error
        And /var/lib/rhc/heartbeat.jsonl contains the pending heartbeat
        """
        ctx = self.ctx
        host_uuid = ctx.generate_identity()
        material = generate_tls(ctx.test_data / "tls")
        upstream = f"{ctx.prometheus_url}/api/v1/write"

        ctx.start_helper(
            "tls-trusted",
            7443,
            "tlsforward.py",
            UPSTREAM=upstream,
            TLS_CERT=str(material.server_cert),
            TLS_KEY=str(material.server_key),
        )
        ctx.write_config("https://127.0.0.1:7443/api/v1/write", ca=str(material.ca))
        run_heartbeat(ctx).assert_success()
        assert_pending_count(ctx, 0)
        wait_for_series(ctx.prometheus_url, f'host_uuid="{host_uuid}"')

        ctx.start_helper(
            "tls-rogue",
            7444,
            "tlsforward.py",
            UPSTREAM=upstream,
            TLS_CERT=str(material.rogue_cert),
            TLS_KEY=str(material.rogue_key),
        )
        ctx.write_config("https://127.0.0.1:7444/api/v1/write", ca=str(material.ca))
        run_heartbeat(ctx).assert_failure()
        assert_pending_count(ctx, 1)
        assert_pending_identity(ctx, host_uuid)

    def test_dropin_precedence(self) -> None:
        """Send the heartbeat to the endpoint selected by a drop-in.

        Given the base configuration points to a Remote Write endpoint
        And a drop-in configures a different Remote Write endpoint
        When `heartbeat run` is run
        Then the heartbeat is sent to the endpoint selected by the documented configuration precedence
        """
        ctx = self.ctx
        host_uuid = ctx.generate_identity()

        ctx.write_config("http://127.0.0.1:1/api/v1/write")
        ctx.dropins.mkdir(parents=True, exist_ok=True)
        (ctx.dropins / "10-test.conf").write_text(
            "\n".join(
                [
                    "[api.heartbeat]",
                    f'uri = "{ctx.prometheus_url}/api/v1/write"',
                    "",
                ]
            ),
            encoding="utf-8",
        )

        run_heartbeat(ctx).assert_success()
        assert_pending_count(ctx, 0)
        wait_for_series(ctx.prometheus_url, f'host_uuid="{host_uuid}"')

    def test_invalid_config(self) -> None:
        """Keep the pending heartbeat when the configuration is invalid.

        Given RHSM identity certificate pair is present
        And the configuration contains an invalid Remote Write URI
        When `heartbeat run` is run
        Then `heartbeat run` reports a configuration or upload error
        And /var/lib/rhc/heartbeat.jsonl contains the pending heartbeat
        """
        ctx = self.ctx
        host_uuid = ctx.generate_identity()

        ctx.write_config("http://127.0.0.1:1/api/v1/write")
        run_heartbeat(ctx).assert_failure()
        assert_pending_count(ctx, 1)

        ctx.write_config("not-a-uri")
        run_heartbeat(ctx).assert_failure()
        assert_pending_count(ctx, 1)
        assert_pending_identity(ctx, host_uuid)
