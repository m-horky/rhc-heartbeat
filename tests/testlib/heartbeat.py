"""Run the heartbeat binary and inspect its pending cache."""

from __future__ import annotations

import json
import subprocess
import time
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import TYPE_CHECKING, Any, TextIO

from .errors import TestFailure

if TYPE_CHECKING:
    from .environment import TestContext


@dataclass(frozen=True)
class RunResult:
    """The exit status and captured log of one heartbeat run."""

    returncode: int
    log_path: Path

    def output(self) -> str:
        """Return the captured log, if it exists."""
        if not self.log_path.exists():
            return ""
        return self.log_path.read_text(encoding="utf-8", errors="replace")

    def assert_success(self) -> None:
        """Fail unless the run exited successfully."""
        if self.returncode != 0:
            raise TestFailure(
                f"heartbeat run failed with exit code {self.returncode}\n{self.output()}"
            )

    def assert_failure(self) -> None:
        """Fail unless the run exited with a nonzero status."""
        if self.returncode == 0:
            raise TestFailure(f"heartbeat run unexpectedly succeeded\n{self.output()}")


@dataclass
class HeartbeatProcess:
    """A heartbeat process that was started without waiting for it."""

    process: subprocess.Popen[str]
    log_path: Path
    log_file: TextIO

    def wait(self, timeout: float = 120) -> RunResult:
        """Wait for the process and return its result."""
        try:
            returncode = self.process.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            self.process.kill()
            self.process.wait()
            self.log_file.close()
            raise TestFailure(f"heartbeat run timed out after {timeout}s")

        self.log_file.close()
        return RunResult(returncode, self.log_path)


def command(ctx: TestContext, *args: str) -> list[str]:
    """Return the command line for the heartbeat binary."""
    return [str(ctx.heartbeat_bin), "run", *args]


def spawn(ctx: TestContext, *args: str) -> HeartbeatProcess:
    """Start the heartbeat binary and capture its output in the test data directory."""
    log_path = ctx.test_data / f"heartbeat-run-{time.time_ns()}.log"
    log_file = log_path.open("w", encoding="utf-8")
    process = subprocess.Popen(
        command(ctx, *args),
        env=ctx.subprocess_env(),
        stdout=log_file,
        stderr=subprocess.STDOUT,
        text=True,
    )
    return HeartbeatProcess(process, log_path, log_file)


def run(ctx: TestContext, *args: str, timeout: float = 120) -> RunResult:
    """Run the heartbeat binary and return its result."""
    return spawn(ctx, *args).wait(timeout)


def pending_rows(ctx: TestContext) -> list[dict[str, Any]]:
    """Read and decode all rows from the pending heartbeat cache."""
    if not ctx.pending_cache.exists() or ctx.pending_cache.stat().st_size == 0:
        return []

    rows: list[dict[str, Any]] = []
    content = ctx.pending_cache.read_text(encoding="utf-8")
    for number, line in enumerate(content.splitlines(), start=1):
        try:
            rows.append(json.loads(line))
        except json.JSONDecodeError as error:
            raise TestFailure(
                f"invalid pending heartbeat row {number}: {error}: {line!r}"
            ) from error

    return rows


def pending_count(ctx: TestContext) -> int:
    """Return the number of pending heartbeat rows."""
    return len(pending_rows(ctx))


def cache_text(ctx: TestContext) -> str:
    """Return the raw pending cache contents for diagnostics."""
    if not ctx.pending_cache.exists():
        return ""
    return ctx.pending_cache.read_text(encoding="utf-8", errors="replace")


def assert_pending_count(ctx: TestContext, expected: int) -> None:
    """Fail unless the cache contains exactly the expected number of rows."""
    actual = pending_count(ctx)
    if actual != expected:
        raise TestFailure(
            f"expected {expected} pending heartbeats, got {actual}\n{cache_text(ctx)}"
        )


def assert_pending_identity(ctx: TestContext, record_uuid: str) -> None:
    """Fail unless any pending row carries the supplied host UUID."""
    if not any(row.get("HostID") == record_uuid for row in pending_rows(ctx)):
        raise TestFailure(f"no pending heartbeat with HostID {record_uuid}\n{cache_text(ctx)}")


def assert_pending_org(ctx: TestContext, organization: str) -> None:
    """Fail unless any pending row carries the supplied organization ID."""
    if not any(row.get("HostOrg") == organization for row in pending_rows(ctx)):
        raise TestFailure(
            f"no pending heartbeat with HostOrg {organization}\n{cache_text(ctx)}"
        )


def assert_cache_valid(ctx: TestContext) -> None:
    """Fail unless every pending cache row is valid JSON."""
    pending_rows(ctx)


def parse_wall_clock(value: str) -> datetime:
    """Parse an RFC 3339 timestamp from a pending heartbeat row."""
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def journal(*arguments: str) -> str:
    """Return the journal of the heartbeat service for diagnostics."""
    result = subprocess.run(
        ["journalctl", "--no-pager", *arguments],
        check=False,
        capture_output=True,
        text=True,
    )
    return result.stdout


def start_service(ctx: TestContext, unit: str, *, expect_failure: bool = False) -> None:
    """Start a systemd unit and optionally assert that it fails.

    Matching the scenario intent is enough here: a successful service start
    must return zero, and an expected failure must return nonzero.
    """
    result = subprocess.run(
        ["systemctl", "start", unit],
        check=False,
        capture_output=True,
        text=True,
    )

    if expect_failure:
        if result.returncode == 0:
            raise TestFailure(
                f"systemctl start {unit} unexpectedly succeeded\n"
                f"{journal('-u', unit, '-n', '50')}"
            )
        return

    if result.returncode != 0:
        raise TestFailure(
            f"systemctl start {unit} failed with exit code {result.returncode}\n"
            f"{result.stderr}\n{journal('-u', unit, '-n', '50')}"
        )


def assert_service_failed(unit: str) -> None:
    """Fail unless the unit's last run failed and recovery is pending or complete."""
    result = subprocess.run(
        [
            "systemctl",
            "show",
            unit,
            "--property=ActiveState",
            "--property=SubState",
            "--property=Result",
        ],
        check=False,
        capture_output=True,
        text=True,
    )
    status = dict(
        line.split("=", 1) for line in result.stdout.splitlines() if "=" in line
    )
    active_state = status.get("ActiveState", "")
    sub_state = status.get("SubState", "")
    run_result = status.get("Result", "")

    if run_result not in ("", "success") and (
        active_state == "failed"
        or (active_state == "activating" and sub_state == "auto-restart")
    ):
        return

    raise TestFailure(
        f"{unit} did not fail: ActiveState={active_state}, "
        f"SubState={sub_state}, Result={run_result}"
    )


def stop_and_reset(unit: str) -> None:
    """Stop a unit and clear its failed state."""
    subprocess.run(["systemctl", "stop", unit], check=False, capture_output=True, text=True)
    subprocess.run(
        ["systemctl", "reset-failed", unit],
        check=False,
        capture_output=True,
        text=True,
    )


def restart_policy(unit: str) -> str:
    """Return the configured Restart policy of a unit."""
    result = subprocess.run(
        ["systemctl", "show", "-p", "Restart", "--value", unit],
        check=False,
        capture_output=True,
        text=True,
    )
    return result.stdout.strip()


def enable_timer(unit: str, *, start: bool = True) -> None:
    """Enable a timer unit and optionally start it immediately."""
    arguments = ["systemctl", "enable"]
    if start:
        arguments.append("--now")
    arguments.append(unit)

    result = subprocess.run(
        arguments,
        check=False,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise TestFailure(f"{' '.join(arguments)} failed: {result.stderr}")


def stop_timer(unit: str) -> None:
    """Stop a timer unit."""
    subprocess.run(["systemctl", "stop", unit], check=False, capture_output=True, text=True)


def timer_is_active(unit: str) -> bool:
    """Return whether a timer unit is active."""
    result = subprocess.run(
        ["systemctl", "is-active", "--quiet", unit],
        check=False,
        capture_output=True,
        text=True,
    )
    return result.returncode == 0


def timer_schedules(unit: str, service: str) -> bool:
    """Return whether the timer listing mentions the service unit."""
    result = subprocess.run(
        ["systemctl", "list-timers", unit, "--no-pager"],
        check=False,
        capture_output=True,
        text=True,
    )
    return service in result.stdout
