#!/usr/bin/env python3
# Copyright 2025 Flant JSC
#
# SPDX-License-Identifier: Apache-2.0

"""Build CHANGELOG/vX.Y.Z.yml from git subjects since the previous release tag."""

import argparse
import re
import subprocess
import sys
from pathlib import Path

import yaml

_VERSION_RE = re.compile(r"^v(\d+)\.(\d+)\.(\d+)(?:\.ru)?\.yml$")


def repo_root() -> Path:
    return Path(__file__).resolve().parent.parent


def iter_changelog_versions(changelog_dir: Path) -> list[tuple[tuple[int, int, int], str]]:
    """Unique (parsed_tuple, 'X.Y.Z') from v*.yml / v*.ru.yml."""
    seen: set[str] = set()
    out: list[tuple[tuple[int, int, int], str]] = []
    for path in changelog_dir.glob("v*.yml"):
        m = _VERSION_RE.match(path.name)
        if not m:
            continue
        major, minor, patch = int(m.group(1)), int(m.group(2)), int(m.group(3))
        ver_str = f"{major}.{minor}.{patch}"
        if ver_str in seen:
            continue
        seen.add(ver_str)
        out.append(((major, minor, patch), ver_str))
    return out


def normalize_version_arg(raw: str) -> str:
    """
    Accept 1.2.3, v1.2.3, V1.2.3, v.1.2.3 → '1.2.3'.
    """
    s = raw.strip()
    if re.match(r"^v\.", s, re.IGNORECASE):
        s = s[2:].lstrip()
    elif s.lower().startswith("v"):
        s = s[1:].lstrip()
    parts = s.split(".")
    if len(parts) != 3:
        raise ValueError(f"expected MAJOR.MINOR.PATCH, got {raw!r}")
    try:
        a, b, c = (int(parts[0]), int(parts[1]), int(parts[2]))
    except ValueError as e:
        raise ValueError(f"invalid semver {raw!r}") from e
    if a < 0 or b < 0 or c < 0:
        raise ValueError(f"negative segment in {raw!r}")
    return f"{a}.{b}.{c}"


def bump_patch(ver_str: str) -> str:
    a, b, c = map(int, ver_str.split("."))
    return f"{a}.{b}.{c + 1}"


def max_changelog_version(versions: list[tuple[tuple[int, int, int], str]]) -> str | None:
    """Highest X.Y.Z among CHANGELOG entries."""
    if not versions:
        return None
    versions = sorted(versions, key=lambda x: x[0], reverse=True)
    return versions[0][1]


def max_version_before(
    versions: list[tuple[tuple[int, int, int], str]],
    output: str,
) -> str | None:
    """Largest changelog version strictly less than output."""
    out_t = tuple(map(int, output.split(".")))
    candidates: list[tuple[tuple[int, int, int], str]] = []
    for t, s in versions:
        if t < out_t:
            candidates.append((t, s))
    if not candidates:
        return None
    candidates.sort(key=lambda x: x[0], reverse=True)
    return candidates[0][1]


def verify_git_tag(root: Path, tag: str) -> bool:
    try:
        subprocess.run(
            ["git", "rev-parse", "--verify", f"{tag}^{{commit}}"],
            cwd=root,
            check=True,
            capture_output=True,
            text=True,
        )
        return True
    except subprocess.CalledProcessError:
        return False


def main() -> int:
    parser = argparse.ArgumentParser(
        description=(
            "Write CHANGELOG/vX.Y.Z.yml with commit subjects since the previous "
            "release (largest changelog version strictly below the target)."
        )
    )
    parser.add_argument(
        "tag",
        nargs="?",
        default=None,
        help="Target version (1.2.3 or v1.2.3). Default: patch bump after latest in CHANGELOG.",
    )
    args = parser.parse_args()

    root = repo_root()
    changelog_dir = root / "CHANGELOG"
    if not changelog_dir.is_dir():
        print(f"CHANGELOG not found: {changelog_dir}", file=sys.stderr)
        return 1

    versions = iter_changelog_versions(changelog_dir)

    if args.tag is not None:
        try:
            output_ver = normalize_version_arg(args.tag)
        except ValueError as e:
            print(e, file=sys.stderr)
            return 1
    else:
        latest = max_changelog_version(versions)
        if not latest:
            print(
                "No vX.Y.Z entries in CHANGELOG; pass an explicit version.",
                file=sys.stderr,
            )
            return 1
        output_ver = bump_patch(latest)

    since_ver = max_version_before(versions, output_ver)
    if since_ver is None:
        print(
            f"No changelog version < {output_ver}; cannot choose a git range.",
            file=sys.stderr,
        )
        return 1

    since_tag = f"v{since_ver}"
    if not verify_git_tag(root, since_tag):
        print(
            f"Git ref {since_tag!r} not found. Create the tag or fix CHANGELOG.",
            file=sys.stderr,
        )
        return 1

    result = subprocess.run(
        ["git", "log", "--format=%s", f"{since_tag}..HEAD"],
        cwd=root,
        check=True,
        capture_output=True,
        text=True,
    )
    lines = [ln.strip() for ln in result.stdout.splitlines() if ln.strip()]

    out_path = changelog_dir / f"v{output_ver}.yml"
    payload = {"Changes": lines}
    with open(out_path, "w", encoding="utf-8") as f:
        yaml.safe_dump(
            payload,
            f,
            allow_unicode=True,
            default_flow_style=False,
            sort_keys=False,
        )

    print(f"Target version: v{output_ver}")
    print(f"Since tag: {since_tag}")
    print(f"git log --format=%s {since_tag}..HEAD → {len(lines)} commit(s)")
    print(f"Wrote {out_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
