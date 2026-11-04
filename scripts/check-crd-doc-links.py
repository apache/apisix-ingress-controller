#!/usr/bin/env python3
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.

"""Verify that generated CRD reference self-links have matching headings."""

from __future__ import annotations

import re
import sys
from collections import defaultdict
from pathlib import Path
from urllib.parse import unquote


HEADING_PATTERN = re.compile(r"^#{1,6}\s+(.+?)\s*#*\s*$")
SELF_LINK_PATTERN = re.compile(r"\]\(#([^)]+)\)")


def heading_slug(heading: str) -> str:
    """Match Docusaurus' generated IDs for the plain headings in this file."""
    return re.sub(r"[^\w -]", "", heading.lower()).replace(" ", "-")


def main() -> int:
    path = Path(sys.argv[1] if len(sys.argv) > 1 else "docs/en/latest/reference/api-reference.md")
    lines = path.read_text(encoding="utf-8").splitlines()

    anchors: set[str] = set()
    occurrences: defaultdict[str, int] = defaultdict(int)
    for line in lines:
        match = HEADING_PATTERN.match(line)
        if not match:
            continue
        base = heading_slug(match.group(1))
        count = occurrences[base]
        occurrences[base] += 1
        anchors.add(base if count == 0 else f"{base}-{count}")

    missing: list[tuple[int, str]] = []
    checked = 0
    for line_number, line in enumerate(lines, start=1):
        for match in SELF_LINK_PATTERN.finditer(line):
            checked += 1
            fragment = unquote(match.group(1))
            if fragment not in anchors:
                missing.append((line_number, fragment))

    if missing:
        print(f"Broken generated CRD self-links in {path}:", file=sys.stderr)
        for line_number, fragment in missing:
            print(f"  line {line_number}: #{fragment}", file=sys.stderr)
        return 1

    print(f"Generated CRD self-links: {checked} checked, all targets exist.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
