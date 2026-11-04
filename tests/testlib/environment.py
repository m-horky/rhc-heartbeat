"""Prepare the paths and state used by a heartbeat test scenario."""

from __future__ import annotations

import ipaddress
import json
import os
import shutil
import subprocess
from contextlib import ExitStack
from dataclasses import dataclass, field
from pathlib import Path

from .errors import TestFailure
from .identity import generate as generate_identity
from .prometheus import wait_healthy
from .services import HelperProcess


def _required_environment(name: str) -> str:
    """Return a required environment variable or fail with a clear message."""
    value = os.environ.get(name)
    if not value:
        raise TestFailure(f"{name} is not set")
    return value


def guest_ipv4() -> str:
    """Return the first non-loopback IPv4 address of the guest."""
    try:
        result = subprocess.run(
            ["hostname", "-I"],
            check=False,
            capture_output=True,
            text=True,
        )
    except OSError:
        result = None

    for token in result.stdout.split() if result is not None else []:
        try:
            address = ipaddress.ip_address(token)
        except ValueError:
            continue
        if address.version == 4 and not address.is_loopback:
            return str(address)

    try:
        result = subprocess.run(
            ["ip", "route", "get", "1.1.1.1"],
            check=False,
            capture_output=True,
            text=True,
        )
    except OSError as error:
        raise TestFailure("cannot determine a non-loopback IPv4 address") from error

    tokens = result.stdout.split()
    for index, token in enumerate(tokens):
        if token == "src" and index + 1 < len(tokens):
            return tokens[index + 1]

    raise TestFailure("cannot determine a non-loopback IPv4 address")


def _run_quietly(arguments: list[str]) -> None:
    """Run a best-effort system command and ignore failures."""
    subprocess.run(arguments, check=False, capture_output=True, text=True)


@dataclass
class TestContext:
    """Filesystem paths and helper lifecycle for one tmt test."""

    tree: Path
    test_data: Path
    heartbeat_bin: Path
    prometheus_url: str
    sandbox_root: Path | None
    config: Path
    dropins: Path
    rhsm_config: Path
    pending_cache: Path
    lock: Path
    consumer_cert: Path
    consumer_key: Path
    product_dir: Path
    default_product_dir: Path
    syspurpose_file: Path
    rhsm_bin: Path
    _stack: ExitStack = field(default_factory=ExitStack, init=False)

    @classmethod
    def from_tmt(cls) -> TestContext:
        """Build a context from the environment supplied by tmt."""
        tree = Path(_required_environment("TMT_TREE"))
        test_data = Path(_required_environment("TMT_TEST_DATA"))
        heartbeat_bin = Path(os.environ.get("HEARTBEAT_BIN", "/usr/libexec/rhc/heartbeat"))
        prometheus_url = os.environ.get("PROMETHEUS_URL", "")
        sandbox_root_raw = os.environ.get("HEARTBEAT_ROOT", "")
        sandbox_root = Path(sandbox_root_raw) if sandbox_root_raw else None

        if sandbox_root is not None:
            config = sandbox_root / "etc-rhc" / "rhc-heartbeat.conf"
            dropins = sandbox_root / "etc-rhc" / "rhc-heartbeat.conf.d"
            rhsm_config = sandbox_root / "etc-rhsm" / "rhsm.conf"
            pending_cache = sandbox_root / "var-lib-rhc" / "heartbeat.jsonl"
            lock = sandbox_root / "run" / "heartbeat.lock"
            consumer_cert = sandbox_root / "identity" / "cert.pem"
            consumer_key = sandbox_root / "identity" / "key.pem"
            product_dir = sandbox_root / "product"
            default_product_dir = sandbox_root / "product-default"
            syspurpose_file = sandbox_root / "etc-rhsm" / "syspurpose.json"
        else:
            config = Path("/etc/rhc/rhc-heartbeat.conf")
            dropins = Path("/etc/rhc/rhc-heartbeat.conf.d")
            rhsm_config = Path("/etc/rhsm/rhsm.conf")
            pending_cache = Path("/var/lib/rhc/heartbeat.jsonl")
            lock = Path("/run/rhc-heartbeat/heartbeat.lock")
            consumer_cert = Path("/etc/pki/consumer/cert.pem")
            consumer_key = Path("/etc/pki/consumer/key.pem")
            product_dir = Path("/etc/pki/product")
            default_product_dir = Path("/etc/pki/product-default")
            syspurpose_file = Path("/etc/rhsm/syspurpose/syspurpose.json")

        return cls(
            tree=tree,
            test_data=test_data,
            heartbeat_bin=heartbeat_bin,
            prometheus_url=prometheus_url,
            sandbox_root=sandbox_root,
            config=config,
            dropins=dropins,
            rhsm_config=rhsm_config,
            pending_cache=pending_cache,
            lock=lock,
            consumer_cert=consumer_cert,
            consumer_key=consumer_key,
            product_dir=product_dir,
            default_product_dir=default_product_dir,
            syspurpose_file=syspurpose_file,
            rhsm_bin=heartbeat_bin.parent / "heartbeat-rhsm",
        )

    @property
    def tests_root(self) -> Path:
        """Return the directory containing the test support code."""
        return self.tree / "tests"

    @property
    def scripts_dir(self) -> Path:
        """Return the directory containing standalone helper scripts."""
        return self.tests_root / "scripts"

    def __enter__(self) -> TestContext:
        """Enter the context and prepare helper process cleanup."""
        self._stack = ExitStack()
        return self

    def __exit__(self, *_: object) -> None:
        """Stop all helpers started through this context."""
        self._stack.close()

    def prepare(self, *, reset: bool = True, prometheus_timeout: float = 30) -> None:
        """Validate the environment, reset state and create the required paths."""
        if not self.heartbeat_bin.is_file() or not os.access(self.heartbeat_bin, os.X_OK):
            raise TestFailure(f"HEARTBEAT_BIN is not executable: {self.heartbeat_bin}")

        self.test_data.mkdir(parents=True, exist_ok=True)
        wait_healthy(self.prometheus_url, prometheus_timeout)

        if reset:
            self.reset()

        self.create_directories()

    def reset(self) -> None:
        """Remove mutable heartbeat state and recreate the base directories."""
        if self.sandbox_root is not None:
            shutil.rmtree(self.sandbox_root, ignore_errors=True)
            self.sandbox_root.mkdir(parents=True, exist_ok=True)
            return

        for directory in (Path("/var/lib/rhc"), Path("/etc/rhc"), Path("/run/rhc-heartbeat")):
            shutil.rmtree(directory, ignore_errors=True)
            directory.mkdir(parents=True, exist_ok=True)

        self.consumer_cert.unlink(missing_ok=True)
        self.consumer_key.unlink(missing_ok=True)

        if shutil.which("systemd-tmpfiles"):
            _run_quietly(["systemd-tmpfiles", "--create", "rhc-heartbeat.conf"])
        if shutil.which("restorecon"):
            _run_quietly(
                [
                    "restorecon",
                    "-R",
                    "/var/lib/rhc",
                    "/etc/rhc",
                    "/run/rhc-heartbeat",
                ]
            )

    def create_directories(self) -> None:
        """Create every parent directory required by the configured paths."""
        for directory in (
            self.config.parent,
            self.dropins,
            self.rhsm_config.parent,
            self.pending_cache.parent,
            self.lock.parent,
            self.consumer_cert.parent,
            self.product_dir,
            self.default_product_dir,
            self.syspurpose_file.parent,
        ):
            directory.mkdir(parents=True, exist_ok=True)

    def subprocess_env(self) -> dict[str, str]:
        """Return the environment that points the heartbeat binary at this context."""
        environment = dict(os.environ)
        environment.update(
            {
                "RHC_HEARTBEAT_CONFIG": str(self.config),
                "RHC_HEARTBEAT_DROPINS": str(self.dropins),
                "RHC_HEARTBEAT_RHSM_CONFIG": str(self.rhsm_config),
                "RHC_HEARTBEAT_PENDING_CACHE": str(self.pending_cache),
                "RHC_HEARTBEAT_LOCK": str(self.lock),
                "RHC_HEARTBEAT_CONSUMER_CERT": str(self.consumer_cert),
                "RHC_HEARTBEAT_CONSUMER_KEY": str(self.consumer_key),
                "RHC_HEARTBEAT_PRODUCT_DIR": str(self.product_dir),
                "RHC_HEARTBEAT_DEFAULT_PRODUCT_DIR": str(self.default_product_dir),
                "RHC_HEARTBEAT_SYSPURPOSE_FILE": str(self.syspurpose_file),
                "RHC_HEARTBEAT_RHSM_BIN": str(self.rhsm_bin),
            }
        )
        return environment

    def write_config(self, endpoint: str, *, proxy: str = "", ca: str = "system") -> None:
        """Write the heartbeat configuration file."""
        self.config.parent.mkdir(parents=True, exist_ok=True)
        self.config.write_text(
            "\n".join(
                [
                    "[http.proxy]",
                    f"uri = {json.dumps(proxy)}",
                    'no-proxy = ""',
                    "",
                    "[api.heartbeat]",
                    f"uri = {json.dumps(endpoint)}",
                    "tls-verify = true",
                    f"ca-path = {json.dumps(ca)}",
                    "",
                ]
            ),
            encoding="utf-8",
        )

    def generate_identity(self, record_uuid: str | None = None) -> str:
        """Generate the consumer certificate pair and return its UUID."""
        return generate_identity(self.consumer_cert.parent, record_uuid, force=True)

    def remove_identity(self) -> None:
        """Remove the consumer certificate pair."""
        self.consumer_cert.unlink(missing_ok=True)
        self.consumer_key.unlink(missing_ok=True)

    def start_helper(self, name: str, port: int, script: str, **environment: str) -> HelperProcess:
        """Start a helper script and stop it when the context exits."""
        helper = HelperProcess(
            ctx=self,
            name=name,
            port=port,
            script=script,
            env=environment,
        )
        return self._stack.enter_context(helper)
