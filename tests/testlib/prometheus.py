"""Query and wait for the Prometheus Remote Write receiver."""

from __future__ import annotations

import json
import time
import urllib.error
import urllib.parse
import urllib.request
from typing import Any, Iterable

from .errors import TestFailure

HEARTBEAT_METRIC = "rhc_heartbeat_time_monotonic_seconds"


def base_url(url: str) -> str:
    """Return the receiver base URL without the query API suffix."""
    return url.rstrip("/").removesuffix("/api/v1/query")


def opener() -> urllib.request.OpenerDirector:
    """Return an opener that ignores proxy environment variables."""
    return urllib.request.build_opener(urllib.request.ProxyHandler({}))


def wait_healthy(url: str, timeout: float = 30) -> None:
    """Wait until the receiver answers its health endpoint."""
    if not url:
        raise TestFailure("PROMETHEUS_URL is not set")

    root = base_url(url)
    deadline = time.monotonic() + timeout
    last_error: Exception | None = None

    while time.monotonic() < deadline:
        try:
            with opener().open(f"{root}/-/healthy", timeout=5):
                return
        except (urllib.error.URLError, OSError) as error:
            last_error = error

        time.sleep(1)

    raise TestFailure(f"Prometheus is not reachable at {root} after {timeout}s: {last_error}")


def port(url: str) -> int:
    """Return the port used by the receiver URL."""
    return urllib.parse.urlsplit(base_url(url)).port or 9090


def write_url(url: str, fallback_host: str) -> str:
    """Return a Remote Write URL, replacing a loopback host with fallback_host."""
    parts = urllib.parse.urlsplit(base_url(url))
    host = parts.hostname or ""

    if host in ("127.0.0.1", "localhost", "::1"):
        host = fallback_host

    if ":" in host:
        host = f"[{host}]"

    netloc = f"{host}:{parts.port}" if parts.port else host
    return urllib.parse.urlunsplit((parts.scheme, netloc, "/api/v1/write", "", ""))


def query(url: str, matchers: Iterable[str] = (), limit: int = 1) -> list[dict[str, Any]]:
    """Query heartbeat samples matching the supplied label matchers."""
    selector = HEARTBEAT_METRIC
    matcher_list = list(matchers)

    if matcher_list:
        selector += "{" + ",".join(matcher_list) + "}"

    statement = f"{selector} and topk({limit}, timestamp({selector}))"
    parameters = urllib.parse.urlencode({"query": statement, "limit": str(limit)})
    request_url = f"{base_url(url)}/api/v1/query?{parameters}"

    try:
        with opener().open(request_url, timeout=30) as response:
            payload = json.load(response)
    except (urllib.error.URLError, OSError, json.JSONDecodeError) as error:
        raise TestFailure(f"Prometheus query failed for {matcher_list}: {error}") from error

    if payload.get("status") != "success":
        raise TestFailure(payload.get("error", "Prometheus returned an unsuccessful query response"))

    return payload.get("data", {}).get("result", [])


def wait_for_series(url: str, matcher: str, timeout: float = 30) -> None:
    """Wait until Prometheus has a series matching a label matcher."""
    deadline = time.monotonic() + timeout
    last_error: Exception | None = None

    while time.monotonic() < deadline:
        try:
            if query(url, [matcher], limit=1):
                return
        except TestFailure as error:
            last_error = error

        time.sleep(1)

    detail = f": {last_error}" if last_error else ""
    raise TestFailure(f"Prometheus has no series matching {matcher} after {timeout}s{detail}")
