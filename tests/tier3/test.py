"""Tier 3 integration tests: systemd units, SELinux and reboot behavior."""

from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(os.environ["TMT_TREE"]) / "tests"))

from testlib.errors import TestFailure
from testlib.heartbeat import (
    assert_pending_count,
    assert_pending_identity,
    assert_service_failed,
    enable_timer,
    restart_policy,
    start_service,
    stop_and_reset,
    stop_timer,
    timer_is_active,
    timer_schedules,
)
from testlib.prometheus import query, wait_for_series
from testlib.case import TmtTestCase


class TestTier3(TmtTestCase):
    """Tier 3 integration tests."""

    def prepare_context(self) -> None:
        """Reset the services and prepare the context for the current phase."""
        reboot_second_phase = (
            self._testMethodName == "test_reboot"
            and int(os.environ.get("TMT_REBOOT_COUNT", "0")) > 0
        )

        if not reboot_second_phase:
            stop_timer("rhc-heartbeat.timer")
            stop_and_reset("rhc-heartbeat.service")
            stop_and_reset("rhc-heartbeat-on.service")

        self.ctx.prepare(
            reset=not reboot_second_phase,
            prometheus_timeout=120 if reboot_second_phase else 30,
        )

    def test_service_upload(self) -> None:
        """Upload a heartbeat through the oneshot service.

        Given rhc-heartbeat is installed with its systemd units and SELinux policy
        And /var/lib/rhc/heartbeat.jsonl is absent or empty
        And a Prometheus Remote Write server is available
        And RHSM identity certificate pair is present
        When `systemctl start rhc-heartbeat.service` is run
        Then the rhc-heartbeat.service unit finishes successfully
        And the heartbeat is uploaded to Prometheus
        And /var/lib/rhc/heartbeat.jsonl is absent or empty
        """
        ctx = self.ctx
        host_uuid = ctx.generate_identity()
        ctx.write_config(f"{ctx.prometheus_url}/api/v1/write")

        start_service(ctx, "rhc-heartbeat.service")

        assert_pending_count(ctx, 0)
        wait_for_series(ctx.prometheus_url, f'host_uuid="{host_uuid}"')

    def test_service_endpoint_down(self) -> None:
        """Fail the service and keep the heartbeat while the endpoint is down.

        Given RHSM identity certificate pair is present
        And the Remote Write endpoint is unavailable
        When `systemctl start rhc-heartbeat.service` is run
        Then the last rhc-heartbeat.service run failed
        And the unit schedules an automatic restart
        And /var/lib/rhc/heartbeat.jsonl contains the pending heartbeat
        """
        ctx = self.ctx
        host_uuid = ctx.generate_identity()
        ctx.write_config("http://127.0.0.1:1/api/v1/write")

        start_service(ctx, "rhc-heartbeat.service", expect_failure=True)
        assert_service_failed("rhc-heartbeat.service")
        assert_pending_count(ctx, 1)
        assert_pending_identity(ctx, host_uuid)

        if restart_policy("rhc-heartbeat.service") != "on-failure":
            raise TestFailure("rhc-heartbeat.service does not schedule on-failure restarts")

        stop_and_reset("rhc-heartbeat.service")

    def test_service_unregistered(self) -> None:
        """Fail the service while keeping the collected heartbeat.

        Given RHSM identity certificate pair is absent
        When `systemctl start rhc-heartbeat.service` is run
        Then the last rhc-heartbeat.service run failed
        And /var/lib/rhc/heartbeat.jsonl contains a single row
        """
        ctx = self.ctx
        ctx.remove_identity()
        ctx.write_config(f"{ctx.prometheus_url}/api/v1/write")

        start_service(ctx, "rhc-heartbeat.service", expect_failure=True)
        assert_service_failed("rhc-heartbeat.service")
        assert_pending_count(ctx, 1)

        stop_and_reset("rhc-heartbeat.service")

    def test_timer(self) -> None:
        """Upload a heartbeat and let the timer schedule the next service run.

        Given RHSM identity certificate pair is present
        And the rhc-heartbeat.timer unit is enabled
        When the timer is started and the service is run
        Then the rhc-heartbeat.service unit finishes successfully
        And the heartbeat is uploaded to Prometheus
        And the timer schedules the next service run
        """
        ctx = self.ctx
        host_uuid = ctx.generate_identity()
        ctx.write_config(f"{ctx.prometheus_url}/api/v1/write")

        enable_timer("rhc-heartbeat.timer")
        start_service(ctx, "rhc-heartbeat.service")

        assert_pending_count(ctx, 0)
        wait_for_series(ctx.prometheus_url, f'host_uuid="{host_uuid}"')

        if not timer_is_active("rhc-heartbeat.timer"):
            raise TestFailure("rhc-heartbeat.timer is not active")
        if not timer_schedules("rhc-heartbeat.timer", "rhc-heartbeat.service"):
            raise TestFailure("rhc-heartbeat.timer does not schedule rhc-heartbeat.service")

        stop_timer("rhc-heartbeat.timer")

    def test_online_report(self) -> None:
        """Upload an online heartbeat through the on service.

        Given RHSM identity certificate pair is present
        When `systemctl start rhc-heartbeat-on.service` is run
        Then the rhc-heartbeat-on.service unit finishes successfully
        And Prometheus instance has a record with kind "on"
        """
        ctx = self.ctx
        host_uuid = ctx.generate_identity()
        ctx.write_config(f"{ctx.prometheus_url}/api/v1/write")

        start_service(ctx, "rhc-heartbeat-on.service")

        assert_pending_count(ctx, 0)
        wait_for_series(ctx.prometheus_url, f'host_uuid="{host_uuid}",kind="on"')

    def test_selinux(self) -> None:
        """Upload profile facts from the confined service.

        Given RHSM identity certificate pair is present
        When `systemctl start rhc-heartbeat.service` is run
        Then the audit log contains no SELinux denials for the rhc-heartbeat domain
        And the heartbeat system profile is collected
        And the uploaded heartbeat contains the system profile facts
        """
        ctx = self.ctx
        host_uuid = ctx.generate_identity()
        ctx.write_config(f"{ctx.prometheus_url}/api/v1/write")

        start_service(ctx, "rhc-heartbeat.service")

        assert_pending_count(ctx, 0)
        wait_for_series(ctx.prometheus_url, f'host_uuid="{host_uuid}"')

        results = query(ctx.prometheus_url, [f'host_uuid="{host_uuid}"'])
        if not any(result.get("metric", {}).get("vcpu_count") not in (None, "") for result in results):
            raise TestFailure("uploaded heartbeat has no vcpu_count profile fact")

    def test_reboot(self) -> None:
        """Upload the queued heartbeat after a reboot.

        Given RHSM identity certificate pair is present
        And /var/lib/rhc/heartbeat.jsonl contains a pending heartbeat for the current boot
        And the rhc-heartbeat.timer unit is enabled
        When the system reboots
        Then the rhc-heartbeat.service unit runs shortly after boot
        And the new heartbeat has a different boot ID
        And the pending heartbeat from the previous boot is uploaded
        And /var/lib/rhc/heartbeat.jsonl is absent or empty
        """
        ctx = self.ctx
        reboot_count = int(os.environ.get("TMT_REBOOT_COUNT", "0"))

        if reboot_count == 0:
            host_uuid = ctx.generate_identity()
            ctx.write_config("http://127.0.0.1:1/api/v1/write")

            start_service(ctx, "rhc-heartbeat.service", expect_failure=True)
            stop_and_reset("rhc-heartbeat.service")

            assert_pending_count(ctx, 1)
            assert_pending_identity(ctx, host_uuid)

            (ctx.test_data / "old-boot-id").write_text(
                Path("/proc/sys/kernel/random/boot_id").read_text(encoding="utf-8"),
                encoding="utf-8",
            )
            (ctx.test_data / "uuid").write_text(f"{host_uuid}\n", encoding="utf-8")

            ctx.write_config(f"{ctx.prometheus_url}/api/v1/write")
            enable_timer("rhc-heartbeat.timer", start=False)

            result = subprocess.run(["tmt-reboot"], check=False, capture_output=True, text=True)
            if result.returncode != 0:
                raise TestFailure(f"tmt-reboot failed: {result.stderr}")
            return

        old_boot = (ctx.test_data / "old-boot-id").read_text(encoding="utf-8").strip()
        new_boot = Path("/proc/sys/kernel/random/boot_id").read_text(encoding="utf-8").strip()
        host_uuid = (ctx.test_data / "uuid").read_text(encoding="utf-8").strip()

        if old_boot == new_boot:
            raise TestFailure("boot ID did not change after reboot")

        wait_for_series(ctx.prometheus_url, f'host_uuid="{host_uuid}",boot_id="{old_boot}"', 180)
        wait_for_series(ctx.prometheus_url, f'host_uuid="{host_uuid}",boot_id="{new_boot}"', 180)
        assert_pending_count(ctx, 0)
