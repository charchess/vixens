from __future__ import annotations

import importlib
import os
import socket
import subprocess
from importlib import metadata
from pathlib import Path

from packaging.requirements import Requirement
from packaging.version import Version

HERMES_VERSION = "0.21.5"
HINDSIGHT_COMMIT = "176f8c2de1369f569c489b831d143b78128b5535"
HINDSIGHT_CLIENT_VERSION = "0.10.1"
AIOHTTP_RETRY_VERSION = "2.9.1"
PROVIDER_DIR = Path("/opt/hermes/plugins/memory/hindsight")


def require_version(distribution: str, expected: str) -> None:
    actual = metadata.version(distribution)
    if actual != expected:
        raise RuntimeError(f"{distribution}: expected {expected}, got {actual}")


def verify_distribution_requirements(distribution: str) -> None:
    dist = metadata.distribution(distribution)
    for raw_requirement in dist.requires or ():
        requirement = Requirement(raw_requirement)
        if requirement.marker and not requirement.marker.evaluate():
            continue
        installed = metadata.version(requirement.name)
        if requirement.specifier and Version(installed) not in requirement.specifier:
            raise RuntimeError(
                f"{distribution} requires {requirement}, installed "
                f"{requirement.name}=={installed}"
            )


def main() -> None:
    # Prove the exact Hermes runtime line and executable are present.
    require_version("hermes-agent", HERMES_VERSION)
    subprocess.run(
        ["/opt/hermes/.venv/bin/hermes", "--version"],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        timeout=30,
    )

    # Prove build-time additions are exact and compatible with the base image.
    require_version("hindsight-client", HINDSIGHT_CLIENT_VERSION)
    require_version("aiohttp-retry", AIOHTTP_RETRY_VERSION)
    verify_distribution_requirements("hindsight-client")
    importlib.import_module("hindsight_client")
    importlib.import_module("aiohttp_retry")

    marker = (PROVIDER_DIR / ".txo-source-revision").read_text(encoding="utf-8").strip()
    if marker != HINDSIGHT_COMMIT:
        raise RuntimeError(f"wrong Hindsight source revision: {marker}")
    if not (PROVIDER_DIR / "plugin.yaml").is_file():
        raise RuntimeError("Hindsight plugin manifest is missing")

    # Provider discovery/import must not need GitHub, PyPI, Hindsight, or any
    # credential.  Fail any accidental outbound socket connect during import.
    original_connect = socket.socket.connect

    def deny_connect(self, address):  # type: ignore[no-untyped-def]
        raise RuntimeError(f"network access attempted during provider import: {address}")

    socket.socket.connect = deny_connect
    try:
        from plugins.memory import find_provider_dir, import_memory_provider_module

        discovered = find_provider_dir("hindsight")
        if discovered is None or discovered.resolve() != PROVIDER_DIR.resolve():
            raise RuntimeError(f"Hindsight provider resolved to unexpected path: {discovered}")
        if not import_memory_provider_module("hindsight"):
            raise RuntimeError("Hindsight provider is not importable")
    finally:
        socket.socket.connect = original_connect

    if os.environ.get("HERMES_DISABLE_LAZY_INSTALLS") != "1":
        raise RuntimeError("Hermes lazy installs are not disabled in the runtime image")


if __name__ == "__main__":
    main()
