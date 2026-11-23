"""Manage helper processes used by the network scenarios."""

from __future__ import annotations

import os
import socket
import subprocess
import sys
import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import TYPE_CHECKING, TextIO

from .errors import TestFailure

if TYPE_CHECKING:
    from .environment import TestContext

PROXY_VARIABLES = ("HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy")


def assert_port_available(port: int) -> None:
    """Fail if a local process already listens on the port."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as probe:
        probe.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            probe.bind(("127.0.0.1", port))
        except OSError as error:
            raise TestFailure(f"port {port} is already in use: {error}") from error


def wait_for_port(port: int, timeout: float = 10) -> None:
    """Wait until a helper process accepts TCP connections on a local port."""
    deadline = time.monotonic() + timeout

    while time.monotonic() < deadline:
        try:
            with socket.create_connection(("127.0.0.1", port), timeout=0.2):
                return
        except OSError:
            time.sleep(0.2)

    raise TestFailure(f"helper did not open port {port}")


@dataclass
class HelperProcess:
    """A helper script running in the background and its captured log."""

    ctx: TestContext
    name: str
    port: int
    script: str
    env: dict[str, str] = field(default_factory=dict)
    process: subprocess.Popen[str] | None = None
    log_file: TextIO | None = None
    log_path: Path | None = None

    def __enter__(self) -> HelperProcess:
        """Start the helper and wait for its port."""
        self.start()
        return self

    def __exit__(self, *_: object) -> None:
        """Stop the helper."""
        self.stop()

    def start(self) -> None:
        """Start the helper with proxy variables removed from its environment."""
        assert_port_available(self.port)

        self.log_path = self.ctx.test_data / f"{self.name}.log"
        self.log_file = self.log_path.open("w", encoding="utf-8")

        environment = dict(os.environ)
        for variable in PROXY_VARIABLES:
            environment.pop(variable, None)
        environment["PORT"] = str(self.port)
        environment.update(self.env)

        try:
            self.process = subprocess.Popen(
                [sys.executable, str(self.ctx.scripts_dir / self.script)],
                env=environment,
                stdout=self.log_file,
                stderr=subprocess.STDOUT,
                text=True,
            )
            wait_for_port(self.port)
            if self.process.poll() is not None:
                raise TestFailure(
                    f"helper {self.name} exited while opening port {self.port}\n"
                    f"{self.log_path.read_text(encoding='utf-8', errors='replace')}"
                )
        except Exception:
            self.stop()
            raise

    def stop(self) -> None:
        """Terminate the helper and close its log."""
        if self.process is not None:
            self.process.terminate()
            try:
                self.process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self.process.kill()
                self.process.wait()
            self.process = None

        if self.log_file is not None:
            self.log_file.close()
            self.log_file = None
