# Application Scoring Model — SUPERSEDED

**Status:** SUPERSEDED depuis le 2026-09-26

Ce document décrivait un ancien modèle de notation sur 100 points. Il ne doit plus être utilisé pour évaluer la maturité d'une application.

Le modèle actuel est le système de maturité à 7 tiers :

```text
Bronze → Silver → Gold → Platinum → Emerald → Diamond → Orichalcum
```

## Sources actuelles

- [`app-golden-standard.md`](app-golden-standard.md) — conventions applicatives ;
- [`quality-standards.md`](quality-standards.md) — critères de qualité ;
- [ADR-023](../adr/023-7-tier-goldification-system-v2.md) et [ADR-029](../adr/029-align-maturity-with-current-platform.md) — décisions de maturité ;
- [`../../scripts/reports/evaluate_maturity.py`](../../scripts/reports/evaluate_maturity.py) — évaluateur exécutable courant ;
- [`../guides/secret-management.md`](../guides/secret-management.md) — architecture secrets OpenBao + External Secrets Operator.

L'ancienne version mentionnait notamment Infisical et des exigences qui ne correspondent plus au modèle courant. Elle reste consultable dans l'historique Git pour provenance.
