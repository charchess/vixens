"""Source-pinned and fail-closed Hermes gateway patch (v2026.9.24 only)."""
from __future__ import annotations

from pathlib import Path

SOURCE = Path("/opt/hermes/gateway/run_turn_runner.py")
ANCHOR = """        return combined

    def _append_auto_media_tags(self, final_response: str, result, agent_history, history_media_paths) -> str:
"""
REPLACEMENT = """        # TXO managed business instructions remain additive even when a
        # user selects /personality or a channel-specific prompt.
        from txo_functional_overlay import append_role_instructions
        return append_role_instructions(combined)

    def _append_auto_media_tags(self, final_response: str, result, agent_history, history_media_paths) -> str:
"""


def main() -> None:
    old = SOURCE.read_text(encoding="utf-8")
    if old.count(ANCHOR) != 1 or "from txo_functional_overlay" in old:
        raise RuntimeError("Hermes source is not the expected pinned gateway: refusing patch")
    new = old.replace(ANCHOR, REPLACEMENT, 1)
    compile(new, str(SOURCE), "exec")
    SOURCE.write_text(new, encoding="utf-8")


if __name__ == "__main__":
    main()
