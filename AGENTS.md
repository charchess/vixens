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
