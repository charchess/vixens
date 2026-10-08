#!/usr/bin/env python3
"""Verify an official Hermes annotated Git release receipt, independently of Docker tag."""
import json
import re
import sys
from pathlib import Path

TAG = re.compile(r"^v[0-9]+(?:\.[0-9]+){1,3}$")
SHA = re.compile(r"^[0-9a-f]{40}$")
DIGEST = re.compile(r"^sha256:[0-9a-f]{64}$")


def verify(tag, git_ref, git_tag):
    if not TAG.fullmatch(tag) or git_ref.get("ref") != "refs/tags/" + tag:
        raise ValueError("invalid or mismatched versioned release tag")
    ref = git_ref.get("object", {})
    if ref.get("type") != "tag" or not SHA.fullmatch(str(ref.get("sha", ""))):
        raise ValueError("official release requires an annotated Git receipt tag")
    if git_tag.get("sha") and git_tag["sha"] != ref["sha"]:
        raise ValueError("annotated tag SHA changed")
    if git_tag.get("tag") != tag:
        raise ValueError("annotated tag name changed")
    commit = git_tag.get("object", {})
    if commit.get("type") != "commit" or not SHA.fullmatch(str(commit.get("sha", ""))):
        raise ValueError("annotated release tag must point to a commit")
    try:
        data = json.loads(git_tag["message"])
    except (KeyError, TypeError, ValueError) as exc:
        raise ValueError("missing or invalid upstream release receipt JSON") from exc
    if not isinstance(data, dict) or data.get("schema") != 1:
        raise ValueError("unsupported release receipt schema")
    if data.get("version") != tag[1:] or data.get("commit") != commit["sha"]:
        raise ValueError("upstream receipt version/commit mismatch")
    digest = data.get("dockerManifestDigest")
    if not isinstance(digest, str) or not DIGEST.fullmatch(digest):
        raise ValueError("upstream receipt lacks an OCI manifest-index digest")
    return digest


if __name__ == "__main__":
    if len(sys.argv) != 4:
        raise SystemExit("usage: verify_release_receipt.py TAG GIT_REF_FILE GIT_TAG_FILE")
    try:
        ref = json.loads(Path(sys.argv[2]).read_text(encoding="utf-8"))
        annotated = json.loads(Path(sys.argv[3]).read_text(encoding="utf-8"))
        print(verify(sys.argv[1], ref, annotated))
    except (ValueError, TypeError, OSError) as exc:
        raise SystemExit(f"Hermes upstream release verification FAILED: {exc}") from exc
