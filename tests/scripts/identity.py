#!/usr/bin/env python3
"""Generate the consumer certificate pair used by the heartbeat.

Run from the repository root::

    python3 tests/scripts/identity.py /etc/pki/consumer
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from testlib.identity import generate


def main(argv: list[str] | None = None) -> int:
    """Generate a certificate pair and print its UUID."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("-f", "--force", action="store_true", help="replace an existing pair")
    parser.add_argument("-u", "--uuid", help="use the supplied UUID as the common name")
    parser.add_argument("directory", nargs="?", default=".", help="output directory")
    args = parser.parse_args(argv)

    print(generate(Path(args.directory), args.uuid, args.force))
    return 0


if __name__ == "__main__":
    sys.exit(main())
