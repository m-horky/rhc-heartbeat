"""Run openssl commands for test certificate generation."""

from __future__ import annotations

import shutil
import subprocess

from .errors import TestFailure


def openssl(*arguments: str) -> None:
    """Run an openssl command and fail with its diagnostics on error."""
    binary = shutil.which("openssl")
    if binary is None:
        raise TestFailure("openssl is required")

    result = subprocess.run(
        [binary, *arguments],
        check=False,
        capture_output=True,
        text=True,
    )

    if result.returncode != 0:
        raise TestFailure(f"openssl {' '.join(arguments)} failed: {result.stderr.strip()}")
