"""Immutable TXO role composition for the pinned Hermes v2026.9.24 gateway.

Adds an approved tenant role to the user's existing gateway/channel/personality
context at API-call time. Never rewrites private SOUL.md, memory or skills.
Actual permissions remain enforced independently by Fabric and the gateway.
"""
from __future__ import annotations

import os

ROLE_ENV = "TXO_FUNCTIONAL_SYSTEM_PROMPT"


def append_role_instructions(personal_prompt: str) -> str:
    role = os.environ.get(ROLE_ENV, "").strip()
    if not role:
        return personal_prompt
    if len(role) > 8192:
        raise ValueError("managed TXO functional instructions exceed approved length")
    return (personal_prompt + "\n\n" + role).strip()
