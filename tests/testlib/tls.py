"""Generate certificate material for the TLS verification scenarios."""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

from .openssl import openssl


@dataclass(frozen=True)
class TLSMaterial:
    """Paths to the CA and the trusted and untrusted server pairs."""

    ca: Path
    server_cert: Path
    server_key: Path
    rogue_cert: Path
    rogue_key: Path


def generate(directory: Path) -> TLSMaterial:
    """Generate a CA, a trusted server pair and an untrusted server pair."""
    directory.mkdir(parents=True, exist_ok=True)

    ca_cert = directory / "ca.pem"
    ca_key = directory / "ca.key"
    server_cert = directory / "server.pem"
    server_key = directory / "server.key"
    server_request = directory / "server.csr"
    extension = directory / "san.ext"
    rogue_cert = directory / "rogue.pem"
    rogue_key = directory / "rogue.key"

    openssl(
        "req",
        "-x509",
        "-newkey",
        "rsa:2048",
        "-nodes",
        "-days",
        "1",
        "-keyout",
        str(ca_key),
        "-out",
        str(ca_cert),
        "-subj",
        "/CN=Test CA",
    )
    openssl(
        "req",
        "-newkey",
        "rsa:2048",
        "-nodes",
        "-keyout",
        str(server_key),
        "-out",
        str(server_request),
        "-subj",
        "/CN=127.0.0.1",
    )

    extension.write_text("subjectAltName=IP:127.0.0.1\n", encoding="utf-8")
    openssl(
        "x509",
        "-req",
        "-in",
        str(server_request),
        "-CA",
        str(ca_cert),
        "-CAkey",
        str(ca_key),
        "-CAcreateserial",
        "-out",
        str(server_cert),
        "-days",
        "1",
        "-extfile",
        str(extension),
    )
    openssl(
        "req",
        "-x509",
        "-newkey",
        "rsa:2048",
        "-nodes",
        "-days",
        "1",
        "-keyout",
        str(rogue_key),
        "-out",
        str(rogue_cert),
        "-subj",
        "/CN=127.0.0.1",
        "-addext",
        "subjectAltName=IP:127.0.0.1",
    )

    return TLSMaterial(
        ca=ca_cert,
        server_cert=server_cert,
        server_key=server_key,
        rogue_cert=rogue_cert,
        rogue_key=rogue_key,
    )
