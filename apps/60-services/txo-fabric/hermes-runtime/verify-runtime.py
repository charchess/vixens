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


def verify_txo_functional_overlay() -> None:
    """Exercise the actual patched gateway with independent per-turn preferences."""
    from types import SimpleNamespace
    from unittest.mock import patch

    from gateway.run_turn_runner import TurnRunner
    from txo_functional_overlay import append_role_instructions

    turn = object.__new__(TurnRunner)
    turn._ctx = SimpleNamespace(
        context_prompt="Session-scoped context",
        channel_prompt="Channel hint",
        source=SimpleNamespace(
            platform="telegram", chat_id="thread1", thread_id=None,
            parent_chat_id=None,
        ),
    )
    current_personality = ["Personal profile /personality"]
    turn._runner = SimpleNamespace(
        _get_system_prompt_for_channel=lambda *a, **kw: current_personality[0]
    )
    with patch.dict(os.environ, {"TXO_FUNCTIONAL_SYSTEM_PROMPT": "Approved sales role"}):
        first = turn._combined_ephemeral_prompt()
        for fragment in (
            "Session-scoped context", "Channel hint",
            "Personal profile /personality", "Approved sales role",
        ):
            if fragment not in first:
                raise RuntimeError("missing prompt layer: " + fragment)
        if first.index("Approved sales role") < first.index("Personal profile"):
            raise RuntimeError("personal channel instructions overrode managed role")
        current_personality[0] = "Updated personal preference"
        second = turn._combined_ephemeral_prompt()
        if "Updated personal preference" not in second or "Approved sales role" not in second:
            raise RuntimeError("managed role does not compose with a changed /personality")
        if "Personal profile" in second:
            raise RuntimeError("stale channel personality survived next-turn recomposition")

    with patch.dict(os.environ, {}, clear=True):
        baseline = turn._combined_ephemeral_prompt()
        if "Approved sales role" in baseline or "Updated personal preference" not in baseline:
            raise RuntimeError("role-less Hermes gateway behavior changed")
        if append_role_instructions("plain prompt") != "plain prompt":
            raise RuntimeError("unbound role did not preserve baseline prompt")

def main() -> None:
    # Prove the exact Hermes runtime line and executable are present.
    require_version("hermes-agent", HERMES_VERSION)
    verify_txo_functional_overlay()
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
