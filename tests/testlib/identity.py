"""Generate the consumer certificate pair used by the heartbeat."""

from __future__ import annotations

import grp
import os
import uuid as uuid_module
from pathlib import Path

from .errors import TestFailure
from .openssl import openssl


def generate(directory: Path, record_uuid: str | None = None, force: bool = True) -> str:
    """Generate a client certificate pair and return its UUID.

    The certificate subject uses ``O=tests`` so uploaded heartbeats carry
    the expected organization ID, and the common name is the host UUID.
    """
    directory.mkdir(parents=True, exist_ok=True)
    certificate = directory / "cert.pem"
    key = directory / "key.pem"

    if certificate.exists() or key.exists():
        if not force:
            raise TestFailure(f"certificate pair already exists under {directory}")
        certificate.unlink(missing_ok=True)
        key.unlink(missing_ok=True)

    host_uuid = record_uuid or str(uuid_module.uuid4())

    previous_umask = os.umask(0o077)
    try:
        openssl(
            "req",
            "-x509",
            "-newkey",
            "rsa:4096",
            "-sha256",
            "-nodes",
            "-days",
            "1",
            "-keyout",
            str(key),
            "-out",
            str(certificate),
            "-subj",
            f"/O=tests/CN={host_uuid}",
            "-addext",
            "extendedKeyUsage=clientAuth",
        )
    finally:
        os.umask(previous_umask)

    try:
        group_id = grp.getgrnam("rhsm").gr_gid
    except KeyError:
        group_id = -1

    for path in (certificate, key):
        if group_id != -1:
            os.chown(path, -1, group_id)
        path.chmod(0o640)

    return host_uuid
