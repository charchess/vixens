# TXO Fabric — Use cases et attentes produit

Ce dossier est le **registre des usages attendus**, racontés du point de vue de leurs acteurs. Il sert à ne pas perdre les idées, les parcours clients, les contraintes métier et les critères observables au fil des ateliers.

**Il ne remplace pas les sources de vérité du dépôt** :

| Nature | Source faisant foi |
|---|---|
| Besoin accepté, scope, critères de livraison | GitHub Issues et commentaires |
| Planification (Status / Priority / Target) | GitHub Project `vixens roadmap` |
| Décision architecturale approuvée et justification | `docs/adr/` et documentation active |
| Implémentation effective | `main`, tests, manifests et CI |
| Comportement physique prouvé | Preuves de recette et observations dans les issues |
| **Scénario métier, exemple, piste, questions ouvertes** | **`usecase/`** |

`WORKFLOW.md` et `AGENTS.md` restent obligatoires. Les exemples ci-dessous sont **des scénarios**, jamais des déclarations de fonctionnement live.

## Ajouter un cas

1. Déposer l'idée brute dans [INBOX.md](INBOX.md) si elle n'est pas encore clarifiée ; ne pas lui attribuer un statut « validé » sans décision.
2. Copier [TEMPLATE.md](TEMPLATE.md) en `UC-NNN-slug.md`, garder un ID stable et écrire **qui veut faire quoi, dans quel contexte et pourquoi**.
3. Décrire le comportement visible, les permissions, les refus et les cas limites sous forme de critères testables.
4. Distinguer explicitement : **besoin confirmé par le propriétaire**, **option/recommandation technique**, **question ouverte**, **preuve d'implémentation**.
5. Lier une issue qui porte la livraison ; si le cas entraîne une décision structurante, proposer ou réviser une ADR. Un agent ne change pas silencieusement les contrats techniques depuis une fiche.
6. Maintenir les liens quand une issue/ADR est remplacée ; ne pas effacer les hypothèses rejetées sans laisser de trace.

Statuts possibles pour une fiche : `Exploration`, `Besoin confirmé`, `Contrat à préciser`, `En implémentation`, `Validé physiquement`, `Historique`. **La validation physique requiert des preuves et ne découle jamais d'une CI verte.**

## Catalogue initial

| Cas | Objectif métier | État |
|---|---|---|
| [UC-001 — Politiques hiérarchiques et verrouillages](UC-001-hierarchical-policies.md) | Administrer sans permettre une escalade d'override | Besoin confirmé, contrat à préciser |
| [UC-002 — Provider managé par Fabric et refacturé](UC-002-fabric-managed-provider.md) | Service prêt à l'emploi et chargeback tenant | Besoin confirmé, tarification à concevoir |
| [UC-003 — Bring Your Own Key du tenant](UC-003-tenant-byok.md) | Le client administre ses propres comptes/providers | Besoin confirmé, UI à livrer |
| [UC-004 — Comptes personnels utilisateur](UC-004-user-owned-accounts.md) | OAuth, device auth ou API personnelles sans mutualisation implicite | Besoin confirmé, isolation par principal à concevoir |
| [UC-005 — WebUI conversationnelle multi-agent](UC-005-agent-chat-webui.md) | Parler à ses agents via une interface unique | Besoin confirmé, implémentation à livrer |
| [UC-006 — Séparation chat / services Hindsight](UC-006-inference-purpose-boundaries.md) | Éviter le mélange des routes, identités, droits et coûts | Contrat produit confirmé, sous-capacités à éprouver |

## Glossaire de travail

- **Fabric Admin** : opérateur de la plateforme et de ses politiques, éventuellement pour un tenant ou un utilisateur ciblé.
- **Tenant Admin** : administrateur explicitement autorisé d'une organisation cliente, jamais déduit de la seule appartenance à un groupe de travail.
- **User** : humain authentifié, dont les choix et credentials personnels sont distincts des credentials du tenant.
- **Provider managé / Tenant BYOK / User BYOK** : mode d'approvisionnement du credential ; **distinct** de qui a le droit de l'employer et de qui paie.
- **Endpoint Fabric** : API logique accessible aux workloads ; pour l'IA, LiteLLM est la façade tenant, CPA le broker interne pour **toutes** les conversations Hermes.
- **Politique effective** : résultat calculé côté serveur de la configuration et des restrictions applicables ; les préférences ne constituent pas des autorisations.

Création initiale : 2026-10-09 · Traçabilité : [#3982](https://github.com/charchess/vixens/issues/3982).
