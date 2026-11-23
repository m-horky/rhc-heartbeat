#!/usr/bin/env python3
"""Query Prometheus from the command line.

Run from the repository root::

    python3 tests/scripts/query.py -q 'host_uuid="..."'
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from testlib.prometheus import query


def positive_integer(value: str) -> int:
    """Parse a positive integer argument."""
    number = int(value)
    if number < 1:
        raise argparse.ArgumentTypeError("value must be a positive integer")
    return number


def main(argv: list[str] | None = None) -> int:
    """Print Prometheus query results as compact JSON."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "-u",
        "--url",
        default="http://127.0.0.1:9090/api/v1/query",
        help="Prometheus API or base URL",
    )
    parser.add_argument(
        "-q",
        "--matcher",
        action="append",
        default=[],
        help="label matcher; may be specified more than once",
    )
    parser.add_argument(
        "-l",
        "--limit",
        type=positive_integer,
        default=1,
        help="maximum number of results",
    )
    args = parser.parse_args(argv)

    for result in query(args.url, args.matcher, args.limit):
        print(json.dumps(result, separators=(",", ":")))

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
