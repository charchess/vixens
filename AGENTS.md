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

## GitHub Project / roadmap

Le GitHub Project personnel **`vixens roadmap`** (owner `charchess`) est la vue de planification active du travail. L'issue reste la source du besoin et de ses critères d'acceptation ; Git et les manifests restent la source du comportement exécutable ; le Project porte la planification (`Target`, `Status` et autres champs de pilotage).

Lorsqu'un chantier touche une issue planifiée :

- vérifier que l'issue est présente dans `vixens roadmap` avant de commencer un travail significatif ;
- si une nouvelle issue est créée pendant l'analyse ou l'implémentation, l'ajouter au Project dans le même flux de travail plutôt que la laisser hors roadmap ;
- maintenir le champ de cible de release (`Target`, par exemple `v0`, `v0.1`, `v1`) cohérent avec la décision produit courante ; ne pas confondre une cible de release avec une itération/sprint temporel ;
- maintenir `Status` cohérent avec l'état réel du chantier lorsque l'environnement permet de le faire ; une PR ouverte, un merge ou une CI verte ne remplacent pas une validation physique requise ;
- lorsqu'une découverte change réellement le scope ou la cible, mettre à jour l'issue et le Project ensemble, avec une justification traçable ;
- ne pas déplacer silencieusement un ticket vers une release ultérieure pour contourner un critère d'acceptation ; toute sortie de scope doit être une décision explicite ;
- avant de considérer un objectif/release terminé, vérifier que les issues correspondantes et la roadmap reflètent l'état réellement accepté, pas seulement l'état du code.

Le Project ne doit pas devenir une seconde spécification technique. Éviter d'y recopier les détails d'architecture ou les critères d'acceptation déjà portés par les issues et la documentation.

Les outils GitHub Project disponibles peuvent varier selon l'agent. Préférer l'API/connector disponible ; sinon utiliser `gh project` ou l'API GraphQL GitHub. Ne pas faire dépendre les règles du dépôt d'une version particulière du CLI. Si l'environnement ne permet pas de modifier le Project, ne pas ignorer la mise à jour : signaler précisément la carte/le champ à modifier ou fournir une commande/action reproductible pour terminer la synchronisation.

Ne pas coder en dur le numéro du Project dans une règle durable lorsque son titre permet de le retrouver. Les opérations CLI sur les Projects peuvent nécessiter le scope OAuth `project`.

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

## Tests TXO Fabric Operator

Pour toute intervention sur le TXO Fabric Operator :

- considérer les tests unitaires/reconciliation comme une partie du livrable, pas comme une étape optionnelle après implémentation ;
- avant de modifier un comportement, identifier les invariants existants qui pourraient régresser et vérifier qu'ils sont déjà couverts ; compléter les TU si nécessaire ;
- toute fonctionnalité nouvelle doit avoir des TU dédiés couvrant ses chemins principaux et, lorsqu'ils existent, ses chemins d'échec/fail-closed ;
- toute correction de bug doit ajouter un test de régression reproduisant le défaut corrigé ;
- ajouter des cas de non-régression multi-tenant/multi-agent pour toute logique susceptible de partager noms, statuts, credentials, policies, ConfigMaps, outposts, workspaces ou autres ressources ;
- tester l'idempotence des reconcile paths : un second reconcile sans changement ne doit pas muter inutilement les objets ni provoquer de rollout ;
- privilégier les assertions de contrat observable (ressource dérivée, condition/status, permission, absence de secret, isolation, conservation/suppression) plutôt que les détails internes fragiles ;
- maximiser la couverture utile des branches critiques, en particulier finalizers/delete, stockage Retain/Delete, IAM/human access, Hindsight, gateway credentials, capabilities/toolsets, integrations et status transitions ;
- ne jamais supprimer/affaiblir un test uniquement pour faire passer une nouvelle implémentation : si le contrat change, documenter explicitement ce changement dans l'issue/PR ;
- traiter `go test -race` comme un gate de l'operator ; une race détectée est un défaut bloquant ;
- consulter les artefacts de couverture CI pour repérer les chemins non testés, sans poursuivre un pourcentage au détriment de scénarios significatifs.

Ces TU complètent, mais ne remplacent pas, envtest/kind et les validations physiques exigées par les dépendances réelles.

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
