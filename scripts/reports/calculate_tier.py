#!/usr/bin/env python3
"""
Goldification Tier Calculator

Evaluates the maturity level of applications based on the 7-tier system:
Bronze → Silver → Gold → Platinum → Emerald → Diamond → Orichalcum

Each level requires all prerequisites from lower levels plus new requirements.

Note: This script evaluates based on STATE-ACTUAL.md fields.
Some criteria may not be available in the current state file.
"""

import argparse
import os
import sys

sys.path.append(os.path.join(os.path.dirname(__file__), "..", "lib"))

from report_utils import parse_markdown_table, save_markdown_table


TIER_ORDER = [
    "🥉 Bronze",
    "🥈 Silver",
    "🥇 Gold",
    "💎 Platinum",
    "🟢 Emerald",
    "💠 Diamond",
    "🌟 Orichalcum",
]

TIER_DESCRIPTIONS = {
    "🥉 Bronze": "L'application existe et fonctionne en dev",
    "🥈 Silver": "Prête pour la production",
    "🥇 Gold": "Observable et optimisable",
    "💎 Platinum": "Fiable avec gestion des ressources",
    "🟢 Emerald": "Données protégées et récupérables",
    "💠 Diamond": "Sécurisée et intégrée",
    "🌟 Orichalcum": "Irréprochable et autonome",
}


def is_defined(value):
    """Check if a value is defined (not N/A or empty)."""
    if value is None:
        return False
    val = str(value).strip()
    return val not in ["", "N/A", "None", "n/a"]


def evaluate_bronze(actual):
    """Level 1: Bronze - App exists"""
    checks = {
        "CPU Request": is_defined(actual.get("CPU Req")),
    }
    return all(checks.values()), checks


def evaluate_silver(actual):
    """Level 2: Silver - Production Ready"""
    checks = {
        "CPU Limit": is_defined(actual.get("CPU Lim")),
        "Memory Limit": is_defined(actual.get("Mem Lim")),
    }
    return all(checks.values()), checks


def evaluate_gold(actual):
    """Level 3: Gold - Standard Quality"""
    checks = {
        "VPA Target": is_defined(actual.get("VPA Target")),
    }
    return all(checks.values()), checks


def evaluate_platinum(actual):
    """Level 4: Platinum - Reliability"""
    checks = {
        "PriorityClass": is_defined(actual.get("Priority")),
        "Sync Wave": is_defined(actual.get("Wave")),
    }
    return all(checks.values()), checks


def evaluate_emerald(actual):
    """Level 5: Emerald - Data Durability"""
    checks = {
        "Backup configuré": actual.get("Backup", "").strip()
        not in ["", "N/A", "None", "n/a"],
    }
    return all(checks.values()), checks


def evaluate_diamond(actual):
    """Level 6: Diamond - Security & Integration"""
    checks = {
        "PSA Labels": "NON-ÉVALUÉ (champ manquant dans STATE-ACTUAL)",
        "NetworkPolicies": "NON-ÉVALUÉ (champ manquant dans STATE-ACTUAL)",
    }
    return False, checks  # Cannot evaluate without the data


def evaluate_orichalcum(actual):
    """Level 7: Orichalcum - Perfection"""
    checks = {
        "Guaranteed QoS": actual.get("QoS", "").strip() == "Guaranteed",
    }
    return all(checks.values()), checks


def calculate_tier(actual):
    """
    Calculate the current maturity tier based on ACTUAL state.
    Returns tuple: (current_tier, next_tier, next_tier_requirements, notes)
    """
    tiers = [
        ("🥉 Bronze", evaluate_bronze),
        ("🥈 Silver", evaluate_silver),
        ("🥇 Gold", evaluate_gold),
        ("💎 Platinum", evaluate_platinum),
        ("🟢 Emerald", evaluate_emerald),
        ("💠 Diamond", evaluate_diamond),
        ("🌟 Orichalcum", evaluate_orichalcum),
    ]

    current_tier = "🥉 Bronze"
    next_tier = None
    next_tier_reqs = []
    notes = []

    # Find current tier by checking each level
    for tier_name, eval_func in tiers:
        passed, checks = eval_func(actual)
        if passed:
            current_tier = tier_name
        elif next_tier is None:
            # This is the next level to achieve
            next_tier = tier_name
            next_tier_reqs = [k for k, v in checks.items() if not v]

    # If already at highest level
    if current_tier == "🌟 Orichalcum":
        next_tier = None
        next_tier_reqs = []

    return current_tier, next_tier, next_tier_reqs, notes


def main():
    parser = argparse.ArgumentParser(
        description="Evaluate goldification maturity tier for applications"
    )
    parser.add_argument(
        "--actual", default="docs/reports/STATE-ACTUAL.md", help="Actual state file"
    )
    parser.add_argument(
        "--output", default="docs/reports/TIER-EVALUATION.md", help="Output report file"
    )
    parser.add_argument(
        "--json", action="store_true", help="Output JSON instead of markdown"
    )
    args = parser.parse_args()

    if not os.path.exists(args.actual):
        print(f"Error: File not found: {args.actual}")
        sys.exit(1)

    actual_rows = parse_markdown_table(args.actual, header_contains="App")

    if not actual_rows:
        print("Warning: Could not parse actual state file")
        actual_rows = []

    actual_map = {
        row["App"].replace("**", ""): row for row in actual_rows if "App" in row
    }

    results = []
    tier_counts = {tier: 0 for tier in TIER_ORDER}
    tier_counts["Non déployée"] = 0

    for app_name in sorted(actual_map.keys()):
        actual = actual_map[app_name]

        if not actual:
            tier = "Non déployée"
            next_tier = None
            missing = ["Application absente"]
        else:
            tier, next_tier, missing, _ = calculate_tier(actual)

        tier_counts[tier] = tier_counts.get(tier, 0) + 1

        results.append(
            {
                "App": f"**{app_name}**",
                "Tier": tier,
                "Prochain": next_tier if next_tier else "-",
                "Manquant": ", ".join(missing[:3])
                + ("..." if len(missing) > 3 else ""),
            }
        )

    if args.json:
        import json

        output = {"tier_counts": tier_counts, "applications": results}
        print(json.dumps(output, indent=2))
        return

    content = "# Évaluation des Niveaux de Maturité\n\n"
    content += "## Résumé\n\n"

    for tier in TIER_ORDER:
        count = tier_counts.get(tier, 0)
        desc = TIER_DESCRIPTIONS.get(tier, "")
        content += f"- **{tier}**: {count} applications - {desc}\n"

    content += f"\n**Total:** {sum(tier_counts.values())} applications\n\n"
    content += "---\n\n"
    content += "## Détail par Application\n\n"

    headers = ["App", "Tier", "Prochain", "Manquant"]
    content += save_markdown_table(headers, results)
    content += "\n\n## Définitions\n\n"

    for tier in TIER_ORDER:
        desc = TIER_DESCRIPTIONS.get(tier, "")
        content += f"### {tier}\n\n{desc}\n\n"

    os.makedirs(os.path.dirname(args.output), exist_ok=True)
    with open(args.output, "w", encoding="utf-8") as f:
        f.write(content)

    print(f"Évaluation générée: {args.output}")
    print("\nRépartition:")
    for tier in TIER_ORDER:
        count = tier_counts.get(tier, 0)
        if count > 0:
            print(f"  {tier}: {count}")


if __name__ == "__main__":
    main()
