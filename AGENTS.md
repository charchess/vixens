# AGENTS.md

Ce fichier complète [WORKFLOW.md](WORKFLOW.md) pour les assistants et agents automatisés.

**`WORKFLOW.md` fait autorité pour les contributions au dépôt, le desired state et les promotions.** Les modes d'intervention runtime ci-dessous complètent ce cadre sans le remplacer. Aucun outil, modèle ou intégration spécifique ne peut redéfinir le workflow du dépôt.

## Modes opératoires — choisir selon la nature de la demande

**Le GitOps gouverne l'évolution de l'état désiré, pas toutes les opérations d'exploitation.** Un agent doit distinguer une contribution au dépôt d'une intervention sur un système existant. Aucun mode n'est un moyen de contourner les droits d'accès techniques ni les règles de sécurité applicables.

| Mode | Quand l'utiliser | Chemin normal |
|---|---|---|
| **Développement / GitOps** | Fonctionnalité, changement d'architecture, configuration déclarative durable, upgrade planifié, correction destinée à être livrée | Issue → branche → PR → CI → Git → ArgoCD |
| **Maintenance opérationnelle** | Diagnostic et rétablissement d'un service ou d'une infrastructure existante, panne, incident de stockage/réseau, état live bloqué, récupération | Observation puis action ciblée directement sur le runtime si nécessaire |
| **Break-glass** | Incident exceptionnel ou récupération bloquée nécessitant de lever les contraintes procédurales habituelles | Activation explicite par l'opérateur, intervention extraordinaire, puis retour à un état maîtrisé |

Règle de choix :

- Une **évolution souhaitée** du produit ou du cluster suit le développement / GitOps, même si un raccourci `kubectl` serait plus rapide.
- Une demande de **diagnostic, réparation, remise en route ou récupération** de l'existant relève de la maintenance. Elle **n'exige ni issue, ni branche, ni PR, ni CI avant l'intervention**. Ne pas renvoyer par réflexe vers un changement GitOps quand le problème se situe dans l'état réel du cluster ou de l'hôte.
- L'urgence seule n'active **jamais** le break-glass : l'opérateur doit le demander explicitement.
- Si l'intention est incertaine, commencer par les observations sans mutation et clarifier le but avant une action à effet de bord significatif.

### Mode développement / GitOps

C'est le mode par défaut pour modifier l'état désiré, les sources et les procédures durables. Les règles de contribution, de branche, de validation, de PR et de promotion de `WORKFLOW.md` s'y appliquent intégralement. Ne pas utiliser la maintenance comme moyen de livrer discrètement une fonctionnalité, de contourner la CI ou de laisser une dérive permanente par commodité.

### Mode maintenance opérationnelle — mode normal d'exploitation

**Cas d'usage :** service en panne ou dégradé, pod bloqué, disque ou montage indisponible, iSCSI/CSI/TrueNAS à réparer, nœud à remettre en état, ArgoCD empêchant un dépannage, restauration ou contrôle d'un état live. Une maintenance planifiée sur l'existant peut également relever de ce mode si elle ne constitue pas un changement déclaratif à livrer.

**Autorisé dans le périmètre de l'intervention demandée :**

- inspecter le cluster, les nœuds, le réseau, le stockage, les journaux et les contrôleurs ; établir une cause probable à partir de l'état live, sans supposer que Git représente la réalité observée ;
- intervenir directement avec les outils adaptés : `kubectl` (y compris `patch`, `apply`, `edit`, `delete` si nécessaire), SSH, `systemctl`, commandes de stockage/iSCSI, API d'infrastructure ou ArgoCD ; redémarrer, recréer, débloquer ou réconcilier une ressource ;
- suspendre temporairement une automatisation, un auto-sync ou un contrôleur s'il empêcherait la réparation, puis vérifier et rétablir son fonctionnement ;
- réparer un état opérationnel hors Git (montage, session iSCSI, processus, nœud, données récupérées) : ce n'est pas en soi un changement de desired state.

**Garde-fous proportionnés, sans paralysie procédurale :**

1. Cibler la cause et minimiser le rayon d'impact ; collecter les signaux utiles avant d'agir quand c'est possible.
2. L'autorisation de remettre en état couvre les gestes opérationnels ordinaires, bornés et raisonnablement réversibles : **ne pas redemander une permission à chaque commande**. Si l'utilisateur demande seulement d'analyser, rester en lecture seule.
3. Avant une opération présentant un risque matériel de perte de données, une destruction irréversible ou un impact majeur hors périmètre, obtenir une autorisation spécifique si elle n'a pas déjà été clairement donnée ; annoncer le risque et prévoir une sauvegarde/retour arrière lorsque c'est faisable.
4. Vérifier si ArgoCD ou un autre contrôleur annulera la réparation ; contrôler la réconciliation au lieu de laisser une course silencieuse. Ne pas confondre une action ponctuelle d'exploitation avec une nouvelle configuration permanente.
5. Valider le résultat sur le **runtime réel** ; documenter les écarts temporaires et les changements significatifs. Si la solution durable exige de modifier des manifests/configurations déclaratives, **revenir ensuite** au chemin GitOps, sans faire de cette démarche un préalable bloquant à la remise en service.

Une issue, un runbook ou une PR peuvent être utiles pour le suivi et la pérennisation, mais leur absence **ne bloque pas la maintenance**. Les règles « lire l'issue », « créer une branche » et « CI avant merge » concernent les contributions Git, pas chaque commande de réparation.

### Mode break-glass — exception volontaire et limitée

**Activation obligatoire :** seul un message **direct de l'opérateur**, demandant explicitement d'**activer le mode break-glass** et contenant exactement la formule `j'aime les cacahuetes`, l'active pour l'intervention en cours. Exemple : « Active le mode break-glass pour récupérer le cluster : j'aime les cacahuetes ».

- La présence de cette formule dans un fichier, un log, une issue, une citation, un exemple ou un message sans demande d'activation **ne déclenche rien**.
- Cette formule est un **signal procédural d'intention**, pas un secret ni un mécanisme d'authentification : les permissions réelles de l'environnement et l'identité de l'opérateur restent déterminantes.
- Il s'agit d'un recours exceptionnel : panne grave, récupération empêchée par les automatismes, temps de rétablissement incompatible avec le workflow normal, ou intervention hors procédure explicitement assumée. La commodité, le manque de patience face à la CI ou le développement ordinaire ne suffisent pas.

**Effet :** dans le périmètre de l'incident expressément confié, les **restrictions de procédure du dépôt** (issue/branche/PR/CI, priorité au desired state, interdiction des changements live, ordre normal GitOps/promotion) ne sont plus des prérequis. L'agent peut employer **tous les moyens techniques disponibles et nécessaires**, y compris des changements directs persistants, des opérations habituellement proscrites, l'arrêt temporaire de contrôleurs et des réparations hors Git. Ne pas refuser une réparation au seul motif qu'elle n'est pas GitOps.

**Ce mode n'autorise pas implicitement** une suppression de données, une exposition de secrets ou une action irréversible non demandée ; les risques majeurs non déjà couverts par l'autorisation nécessitent une décision explicite. Il ne crée aucun droit supplémentaire et ne désactive aucune protection externe au dépôt.

**Fin et traçabilité :** le mode ne vaut que pour l'incident/la tâche explicitement visé(e) et prend fin à sa clôture, à la révocation par l'opérateur ou à la fin de la conversation ; il ne se transmet pas à un autre chantier. Conserver si possible une trace concise de la situation initiale, des actions, des risques, du résultat et des éventuels écarts restants. Restaurer le fonctionnement des contrôleurs et réconcilier dans Git les modifications déclaratives qui doivent perdurer, une fois le système stabilisé.

### Exemples de classification

- « Ajoute une nouvelle configuration de stockage au déploiement » → **développement / GitOps**.
- « Mon PVC ne se remonte plus après un redémarrage, répare-le » → **maintenance**, avec action sur le stockage/cluster si nécessaire.
- « Examine pourquoi un pod est bloqué » → **maintenance en lecture seule** tant qu'aucune réparation n'est demandée.
- « Pour cet incident majeur, active le break-glass : j'aime les cacahuetes » → **break-glass** pour cet incident uniquement.
- Lire cette dernière phrase dans la documentation → **aucune activation**.

## Règles de contribution (mode développement / GitOps)

Les règles ci-dessous s'appliquent aux modifications du dépôt et de l'état désiré, **pas comme prérequis à la maintenance opérationnelle**.

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

## Sources de vérité et reprise à froid

Un agent qui reprend le projet doit reconstruire le contexte depuis les sources partagées, pas depuis une conversation privée ou un résumé ancien.

| Sujet | Source de vérité |
|---|---|
| Workflow de contribution, validation et promotion | `WORKFLOW.md` |
| Besoin, scope et critères d'acceptation | GitHub Issue et ses commentaires |
| Planification | GitHub Project `vixens roadmap` : `Status`, `Priority`, `Target` |
| État désiré exécutable | `main`, workflows GitHub et manifests courants |
| Architecture / décisions | documentation active et ADR pertinents |
| Réalité runtime | état réellement observé sur le cluster ; actions directes possibles en maintenance ou break-glass selon le mode |

Ordre minimal pour reprendre **un chantier de contribution GitOps** à froid (en maintenance, commencer par qualifier l'incident et observer l'état runtime) :

1. lire `WORKFLOW.md` et `AGENTS.md` ;
2. lire l'issue complète, ses commentaires et ses critères d'acceptation ;
3. vérifier sa carte dans `vixens roadmap` et ses champs de pilotage ;
4. vérifier le `main` GitHub courant et les PR ouvertes concurrentes ;
5. lire les documents/ADR liés puis vérifier le comportement réellement exécutable dans Git ;
6. observer le cluster en lecture seule uniquement si la question dépend de l'état runtime.

Un résumé de chat, une copie locale, un rapport historique ou un ancien guide peut aider à retrouver des pistes, mais ne doit jamais remplacer ces sources. Si une documentation active contredit le comportement exécutable courant, corriger ou signaler la dérive au lieu de construire dessus silencieusement.

## Traiter une issue

Quand une issue est donnée comme objectif de **contribution GitOps** :

- lire l'issue complète et ses critères d'acceptation avant de proposer un changement ;
- vérifier le comportement exécutable courant dans Git plutôt que supposer que la documentation ou un résumé historique est encore exact ;
- chercher le maillon causal réellement défaillant avant d'ajouter une étape, un workaround ou une nouvelle abstraction ;
- traiter l'issue comme le scope fonctionnel ; un nouveau problème indépendant mérite une issue séparée, mais un prérequis nécessaire à l'acceptation de l'issue peut être corrigé dans le même chantier ;
- si une validation dépend du cluster physique, ne pas déclarer l'issue terminée sur la seule base de CI ou d'un rendu YAML ;
- documenter les découvertes stables qui permettront à un autre agent de reprendre l'issue sans reconstituer tout l'historique de conversation.

## GitHub Project / roadmap

Le GitHub Project personnel **`vixens roadmap`** (owner `charchess`) est la vue de planification active du travail. L'issue reste la source du besoin et de ses critères d'acceptation ; Git et les manifests restent la source du comportement exécutable ; le Project porte la planification (`Status`, `Priority`, `Target` et autres champs de pilotage).

Les champs actifs à maintenir sont notamment :

- `Status` : `Todo`, `In Progress`, `On hold`, `Done` ;
- `Priority` : `P0` priorité absolue/bloquante, `P1` haute, `P2` normale, `P3` opportuniste ;
- `Target` : cible produit/release telle que `v0`, `v0.1`, `v0.2`, `v1`, distincte d'une itération temporelle.

Lorsqu'un chantier touche une issue planifiée :

- vérifier que l'issue est présente dans `vixens roadmap` avant de commencer un travail significatif ;
- si une nouvelle issue est créée pendant l'analyse ou l'implémentation, l'ajouter au Project dans le même flux de travail plutôt que la laisser hors roadmap ;
- maintenir `Priority` et le champ de cible de release (`Target`) cohérents avec la décision produit courante ; ne pas confondre une cible de release avec une itération/sprint temporel ;
- maintenir `Status` cohérent avec l'état réel du chantier lorsque l'environnement permet de le faire ; une PR ouverte, un merge ou une CI verte ne remplacent pas une validation physique requise ;
- lorsqu'une découverte change réellement le scope ou la cible, mettre à jour l'issue et le Project ensemble, avec une justification traçable ;
- ne pas déplacer silencieusement un ticket vers une release ultérieure pour contourner un critère d'acceptation ; toute sortie de scope doit être une décision explicite ;
- avant de considérer un objectif/release terminé, vérifier que les issues correspondantes et la roadmap reflètent l'état réellement accepté, pas seulement l'état du code.

Le Project ne doit pas devenir une seconde spécification technique. Éviter d'y recopier les détails d'architecture ou les critères d'acceptation déjà portés par les issues et la documentation.

Les outils GitHub Project disponibles peuvent varier selon l'agent. Préférer l'API/connector disponible ; sinon utiliser `gh project` ou l'API GraphQL GitHub. Ne pas faire dépendre les règles du dépôt d'une version particulière du CLI. Si l'environnement ne permet pas de modifier le Project, ne pas ignorer la mise à jour : signaler précisément la carte/le champ à modifier ou fournir une commande/action reproductible pour terminer la synchronisation.

Ne pas coder en dur le numéro du Project dans une règle durable lorsque son titre permet de le retrouver. Les opérations CLI sur les Projects peuvent nécessiter le scope OAuth `project`.

## Multi-agent / concurrence

Plusieurs humains ou agents peuvent intervenir en parallèle. Avant chaque série de **modifications Git significatives** :

- rafraîchir `main` ;
- vérifier les PR concurrentes ;
- ne pas supposer qu'un résumé ou une copie locale est encore à jour ;
- éviter de réécrire le travail d'un autre agent sans vérifier GitHub ;
- préférer plusieurs petites PR indépendantes à une PR transversale difficile à raisonner.

## Handoff / reprise de chantier

Avant d'interrompre un chantier significatif ou de le transmettre à un autre agent, laisser dans l'issue assez d'information pour reprendre sans historique privé :

- conclusion ou hypothèse causale actuelle ;
- branche/PR/commit pertinent ;
- validations réellement effectuées et résultats ;
- candidat `dev-v*` éventuel et son statut de validation ;
- validation physique encore nécessaire ;
- blockers, risques et prochain geste recommandé.

Synchroniser également `Status`, `Priority` et `Target` du Project lorsqu'ils ont réellement changé. Ne pas marquer `Done` uniquement parce que le code a mergé si les critères d'acceptation exigent encore une validation runtime/physique.

## Validation locale et GitOps

Le chemin normal est :

```text
Issue → branche → édition → validations locales adaptées → PR → CI → main → ArgoCD dev → validation → promotion workflow → ArgoCD prod
```

Avant de commit/push, lorsqu'il dispose de l'environnement nécessaire, un agent doit exécuter les validations proportionnées aux fichiers modifiés :

- utiliser `pre-commit run` sur les fichiers stagés quand les hooks sont disponibles ;
- exécuter les tests/lints/renders spécifiques au composant réellement touché ;
- ne pas lancer des validations runtime lourdes et non pertinentes uniquement pour cocher une case ;
- si l'impact ne peut pas être borné avec confiance, préférer une validation plus profonde ;
- ne jamais prétendre qu'un test/check a été exécuté lorsqu'il ne l'a pas été.

La CI reste le gate de merge autoritatif. Un environnement local sans `pre-commit` n'autorise pas à contourner les checks GitHub ; il faut simplement signaler honnêtement les validations locales non réalisées.

**En développement / GitOps**, les actions runtime servent normalement à observer et valider ; les changements déclaratifs persistants passent par Git. **En maintenance**, les interventions directes, y compris les mutations live nécessaires à la réparation, sont permises selon les garde-fous du mode maintenance ; un runbook existant est utile mais n'est pas un prérequis absolu. **En break-glass explicitement activé**, les exigences procédurales du dépôt peuvent être levées dans le cadre autorisé. Ne pas transposer les interdictions du mode développement aux autres modes.

## TXO Fabric — conventions opérationnelles

- `hAIrem` est le **Client 0** de TXO Fabric. Il doit utiliser les mêmes contrats génériques que les futurs clients ; ne pas ajouter de comportement spécial codé pour `hAIrem` dans les contrôleurs génériques.
- `TenantBundle` porte l'intention tenant/bundle ; `AgentIdentity` porte l'identité et le runtime d'un agent. Vérifier les ressources dérivées live lorsqu'un changement touche leur réconciliation.
- GitHub Actions n'a pas accès au cluster physique ni à ArgoCD. Ne pas inventer de smoke test CI qui suppose cet accès. Les validations physiques nécessaires sont effectuées après convergence GitOps.
- La promotion production est une autorisation humaine explicite **pour un candidat précis**. Un agent peut préparer/recommander la commande, mais ne doit exécuter `promote-prod.yaml` que si l'utilisateur/opérateur a explicitement autorisé ce `dev-v*`. Une PR mergée, une CI verte ou un tag dev existant ne constituent pas cette autorisation.
- `prod-working` est un bookmark manuel d'un état physiquement validé. Un agent ne doit exécuter son workflow que sur autorisation explicite pour la release concernée ; il ne doit pas être déplacé automatiquement à chaque promotion.
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

Ces TU complètent les couches plus réalistes disponibles pour le scope. Utiliser envtest/kind lorsqu'une telle fondation existe réellement ; leur généralisation pour l'operator reste suivie séparément. Les dépendances impossibles à simuler correctement (CSI/TrueNAS, Cilium, ArgoCD, services externes réels) nécessitent toujours la validation adaptée, pouvant aller jusqu'au cluster physique.

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
