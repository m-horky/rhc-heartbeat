"""Tier 1 integration tests: basic heartbeat lifecycle."""

from __future__ import annotations

import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(os.environ["TMT_TREE"]) / "tests"))

from testlib.environment import guest_ipv4
from testlib.errors import TestFailure
from testlib.heartbeat import (
    assert_pending_count,
    assert_pending_identity,
    assert_pending_org,
    parse_wall_clock,
    pending_rows,
    run as run_heartbeat,
)
from testlib.prometheus import wait_for_series
from testlib.prometheus import write_url
from testlib.case import TmtTestCase


class TestTier1(TmtTestCase):
    """Tier 1 integration tests."""

    def test_before_registration(self) -> None:
        """Keep one pending heartbeat before registration.

        Given RHSM identity certificate pair is absent
        When `heartbeat run` is run
        Then /var/lib/rhc/heartbeat.jsonl contains a single row
        When `heartbeat run` is run
        Then /var/lib/rhc/heartbeat.jsonl contains a single row
        And the wall-clock time in that row is higher
        """
        ctx = self.ctx
        ctx.write_config(f"{ctx.prometheus_url}/api/v1/write")
        ctx.remove_identity()

        run_heartbeat(ctx).assert_failure()
        assert_pending_count(ctx, 1)
        first = parse_wall_clock(pending_rows(ctx)[0]["TimeUnix"])

        run_heartbeat(ctx).assert_failure()
        assert_pending_count(ctx, 1)
        second = parse_wall_clock(pending_rows(ctx)[0]["TimeUnix"])

        if second <= first:
            raise TestFailure(f"wall-clock time did not increase ({first} -> {second})")

    def test_registration(self) -> None:
        """Upload the pending and new heartbeat after registration.

        Given RHSM identity certificate pair is absent
        When `heartbeat run` is run
        Then /var/lib/rhc/heartbeat.jsonl contains a single row
        When the system gets registered
        Then the RHSM identity certificate pair is present
        When `heartbeat run` is run
        Then /var/lib/rhc/heartbeat.jsonl is absent or empty
        And Prometheus instance has a record with RHSM UUID
        """
        ctx = self.ctx
        ctx.write_config(f"{ctx.prometheus_url}/api/v1/write")
        ctx.remove_identity()

        run_heartbeat(ctx).assert_failure()
        assert_pending_count(ctx, 1)

        host_uuid = ctx.generate_identity()

        run_heartbeat(ctx).assert_success()
        assert_pending_count(ctx, 0)
        wait_for_series(ctx.prometheus_url, f'host_uuid="{host_uuid}"')

    def test_local_caching(self) -> None:
        """Keep the pending heartbeat when it cannot be uploaded.

        Given RHSM identity certificate pair is present
        And HTTP proxy is configured to a URI where proxy does not run
        When `heartbeat run` is run
        Then /var/lib/rhc/heartbeat.jsonl contains a single row
        And the RHSM identity and organization ID are present in the row
        """
        ctx = self.ctx
        host_uuid = ctx.generate_identity()
        endpoint = write_url(ctx.prometheus_url, guest_ipv4())
        ctx.write_config(endpoint, proxy="http://127.0.0.1:1")

        run_heartbeat(ctx).assert_failure()
        assert_pending_count(ctx, 1)
        assert_pending_identity(ctx, host_uuid)
        assert_pending_org(ctx, "tests")
