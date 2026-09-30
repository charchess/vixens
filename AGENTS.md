# AGENTS.md

Ce fichier complète [WORKFLOW.md](WORKFLOW.md) pour les assistants et agents automatisés.

**`WORKFLOW.md` fait autorité.** Aucun outil, modèle ou intégration spécifique ne peut redéfinir le workflow du dépôt.

## Règles obligatoires

1. Vérifier le `main` GitHub courant avant toute modification.
2. Vérifier les PR ouvertes qui peuvent toucher le même scope.
3. Lire l'issue et les documents/ADR pertinents avant d'écrire.
4. Créer ou utiliser une branche dédiée depuis le `main` courant.
5. Limiter les changements au scope explicite ; créer une issue séparée pour un nouveau chantier.
6. Ne jamais pousser directement sur `main`.
7. Ne jamais utiliser `kubectl apply`, `kubectl edit` ou `kubectl delete` comme mécanisme persistant lorsque GitOps s'applique.
8. Ne jamais stocker de valeur secrète en clair dans Git.
9. Laisser CI valider la PR, puis ArgoCD réconcilier le cluster.
10. Mettre à jour la documentation lorsque le contrat, l'architecture ou la procédure change.

## Traiter une issue

Quand une issue est donnée comme objectif de travail :

- lire l'issue complète et ses critères d'acceptation avant de proposer un changement ;
- vérifier le comportement exécutable courant dans Git plutôt que supposer que la documentation ou un résumé historique est encore exact ;
- chercher le maillon causal réellement défaillant avant d'ajouter une étape, un workaround ou une nouvelle abstraction ;
- traiter l'issue comme le scope fonctionnel ; un nouveau problème indépendant mérite une issue séparée, mais un prérequis nécessaire à l'acceptation de l'issue peut être corrigé dans le même chantier ;
- si une validation dépend du cluster physique, ne pas déclarer l'issue terminée sur la seule base de CI ou d'un rendu YAML ;
- documenter les découvertes stables qui permettront à un autre agent de reprendre l'issue sans reconstituer tout l'historique de conversation.

## Multi-agent / concurrence

Plusieurs humains ou agents peuvent intervenir en parallèle. Avant chaque série de modifications significatives :

- rafraîchir `main` ;
- vérifier les PR concurrentes ;
- ne pas supposer qu'un résumé ou une copie locale est encore à jour ;
- éviter de réécrire le travail d'un autre agent sans vérifier GitHub ;
- préférer plusieurs petites PR indépendantes à une PR transversale difficile à raisonner.

## GitOps

Le chemin normal est :

```text
Issue → branche → PR → CI → main → ArgoCD dev → validation → promotion workflow → ArgoCD prod
```

Les actions runtime sont acceptables pour **observer et diagnostiquer** (`get`, `describe`, `logs`, requêtes API, tests réseau). Les changements persistants doivent être encodés dans Git.

## TXO Fabric — conventions opérationnelles

- `hAIrem` est le **Client 0** de TXO Fabric. Il doit utiliser les mêmes contrats génériques que les futurs clients ; ne pas ajouter de comportement spécial codé pour `hAIrem` dans les contrôleurs génériques.
- `TenantBundle` porte l'intention tenant/bundle ; `AgentIdentity` porte l'identité et le runtime d'un agent. Vérifier les ressources dérivées live lorsqu'un changement touche leur réconciliation.
- GitHub Actions n'a pas accès au cluster physique ni à ArgoCD. Ne pas inventer de smoke test CI qui suppose cet accès. Les validations physiques nécessaires sont effectuées après convergence GitOps.
- La promotion production est une autorisation humaine explicite. Le workflow de promotion crée le `prod-v*` immuable et déplace `prod-stable` ; ne pas promouvoir automatiquement parce qu'une PR a mergé.
- `prod-working` est un bookmark manuel d'un état physiquement validé. Il ne doit pas être déplacé automatiquement à chaque promotion.
- Pour l'opérateur Fabric, les artefacts générés (`CRD`, RBAC, etc.) ne sont utiles que s'ils sont à la fois correctement générés **et effectivement inclus** dans les rendus/Kustomizations consommés par ArgoCD.
- Une validation doit distinguer un vrai résultat fonctionnel d'un prérequis absent. Exemple : un test d'écriture n'est pas un PASS si le pod, le mount ou le PVC attendu n'existe pas.
- Pour les commandes destinées à être copiées dans un shell interactif, ne pas utiliser `set -euo pipefail` au niveau supérieur. Si un strict mode est utile, l'isoler dans un sous-shell ou un script. Les diagnostics doivent autant que possible collecter plusieurs signaux avant de sortir.
- Privilégier les preuves causales : intention Git → rendu ArgoCD → ressource live → comportement fonctionnel.

## Validation

Choisir les validations adaptées au changement :

- YAML / Kustomize render pour les manifests ;
- checks CI et sécurité ;
- ArgoCD `Synced` + `Healthy` après merge ;
- test fonctionnel ciblé ;
- validation prod après promotion lorsqu'elle est nécessaire.

Ne pas fermer une issue uniquement parce qu'un fichier a été modifié : vérifier le résultat attendu.

## Outils

Les agents peuvent utiliser les outils disponibles dans leur environnement (éditeur, shell, API GitHub, recherche documentaire, navigateur, etc.).

Les outils sont **optionnels et interchangeables**. Les règles du dépôt ne doivent jamais dépendre de Serena, Just, Claude, Gemini, OpenCode, Archon ou d'un autre client particulier.

Les fichiers spécifiques à un agent ne doivent contenir que des adaptations minimales et renvoyer vers `WORKFLOW.md` / `AGENTS.md` au lieu de recopier les règles.

## Documentation

Ordre de priorité :

1. `WORKFLOW.md` — workflow canonique ;
2. `AGENTS.md` — contraintes agents ;
3. workflows GitHub et manifests — comportement exécutable ;
4. architecture / ADR / guides actifs ;
5. rapports et documents historiques.

Un document historique peut mentionner une technologie retirée sans redevenir une instruction active.
