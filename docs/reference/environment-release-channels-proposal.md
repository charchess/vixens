# Proposition — Canaux d'environnement, tags de déploiement et validation des contrats

**Statut : PROPOSITION / ÉTUDE — aucune décision de mise en œuvre**  
**Date :** 2026-10-25  
**Sujet parent :** [#3959 — Refactorisation GitOps](https://github.com/charchess/vixens/issues/3959)  
**Prérequis release multi-repos :** [#3960](https://github.com/charchess/vixens/issues/3960)  
**Architecture envisagée :** [Couches et contrats neutres](platform-layer-contracts-proposal.md)  
**Workflow actuel faisant autorité :** [WORKFLOW.md](../../WORKFLOW.md)

> L'enjeu est de **réexaminer en profondeur** les conventions de tags, la relation entre environnements, le mécanisme de promotion, le degré d'automatisation, les overlays et les validations des contrats de plateforme. Cette note est une base de discussion, **pas** une approbation de modification des tags, GitHub Actions, ArgoCD ou clusters. Le chantier #3959 reste explicitement **PARK / DEFER** jusqu'à nouvelle décision du propriétaire. Ne pas réinterpréter les exemples ci-dessous comme un workflow exécutable.

## 1. État actuel vérifié dans les sources

Les conventions actives se trouvent dans `WORKFLOW.md`, `docs/guides/promotion-workflow.md`, et les workflows `.github/workflows/auto-tag-dev.yaml`, `promote-prod.yaml` et `mark-prod-working.yaml` :

| Référence actuelle | Nature | Signification actuelle |
|---|---|---|
| `main` | Branche de développement | Dernier état désiré dev ; ArgoCD dev suit cette branche |
| `dev-vYYYY.MM.<PR-or-sha>` | Tag versionné immuable | **Identité du snapshot**, créé automatiquement après push sur `main` ; pas une preuve de validation dev |
| `dev-latest` | Tag alias mutable | Dernier snapshot de dev (auto-déplacé) |
| `prod-vYYYY.MM.<PR-or-sha>` | Tag versionné immuable | Release de production créée/vérifiée à la promotion autorisée |
| `prod-stable` | Tag alias mutable | Révision source suivie par ArgoCD prod ; déplacée par `promote-prod.yaml` après ses contrôles |
| `prod-working` | Tag alias mutable **manuel** | Dernière release explicitement confirmée connue-bonne par l'opérateur ; ne bouge jamais automatiquement |

L'appellation `work-stable` a été évoquée oralement mais **n'a pas été trouvée** dans les fichiers GitOps consultés ; le mécanisme équivalent semble être `prod-working`. Ne pas confondre les **canaux Hermes** (`hermes-edge`, `hermes-canary`, `hermes-stable`, objets `AgentRuntimeProfile`) avec les **canaux de déploiement de la plateforme** : ils représentent des responsabilités différentes.

La promotion production actuelle part d'un `dev-v*` immuable, vérifie sa présence dans l'historique `main` et les contraintes de provenance d'artefacts, puis crée une release `prod-v*` et met à jour `prod-stable`. **La validation dev est une étape distincte du tag**, et l'autorisation de prod est explicitement humaine. Aucune chaîne active et complète `dev → test → staging → prod` n'a été établie par cette lecture.

## 2. Séparer quatre concepts avant d'inventer de nouveaux tags

1. **Identité d'un artefact / snapshot** : SHA Git immuable ; dans l'architecture multi-repos, un **manifest de release / BOM immuable** résolvant les SHAs des couches, charts, images et leurs digests.
2. **Environnement logique** : `dev`, `test`, `staging`, `prod` ; définit politiques et exigences de validation, pas les capacités fonctionnelles.
3. **Cible de déploiement physique** : cluster identifié par Terraform (p. ex. `umi`, `perso-charchess` ou plusieurs clusters dans le même environnement). **Ne pas confondre** nom du cluster, propriétaire/cible, domaine fonctionnel et environnement.
4. **État de qualification / approbation** : preuves de rendu, CI, tests contractuels, déploiement et acceptation physique, autorisation éventuelle ; **un tag seul n'atteste rien**.

Question structurante : une référence `perso-charchess` est-elle un **environnement autonome**, un **cluster cible personnel**, un **profil/périmètre d'applications** ou un **canal de releases distinct** ? Éviter de créer un cinquième environnement sur la seule base d'un nom personnel. Le cluster peut porter `environment=prod` tout en étant sélectionné comme cible/périmètre personnel. Décision à instruire.

## 3. Chaîne de promotion envisagée

### Intention de l'opérateur

Étudier un flux ordonné :

```text
     main + snapshots immuables
               │
            dev
               │  qualification
               ▼
             test
               │  validation / preuve
               ▼
           staging
               │  recette préproduction / autorisation
               ▼
             prod
```

**Règle candidate :** aucun environnement aval ne consomme un snapshot qui n'a pas été **qualifié et promu explicitement** par la chaîne amont. Chaque environnement suit un snapshot/release *déterministe*, jamais implicitement une branche mutable non validée (hors dev si ce comportement est conservé).

La formule « prod toujours en retard sur staging, staging sur test, test sur dev » doit être **définie sémantiquement** :

- **Pas d'ordre numérique fiable des tags** : `vYYYY.MM.PR` suit la date/PR de création, pas un numéro séquentiel du code ni le niveau de preuve.
- **Pas de retard strict obligatoire** : après une promotion, **prod et staging peuvent légitimement pointer vers le même snapshot** ; il faut autoriser l'égalité. Le flux doit plutôt certifier la relation de promotion et la traçabilité.
- Pour un **monorepo strictement linéaire**, on peut contrôler que la révision d'un environnement aval est un **ancêtre ou égale** de l'amont (`merge-base --is-ancestor`), et qu'elle possède les preuves nécessaires. **L'ascendance seule ne prouve pas la qualification**.
- Pour un **multi-repo**, deux releases peuvent combiner des SHAs différents ; l'ascendance globale d'un commit unique ne suffit plus. Comparer les manifestes immuables/BOM et des **relations de promotion explicites** plutôt qu'un numéro de version ou l'heure du tag.
- Un **rollback**, un **hotfix d'urgence**, ou une indisponibilité d'un environnement amont peut casser temporairement la relation d'âge apparente. Prévoir une voie de dérogation **auditée et autorisée**, jamais un mouvement automatique pour « réaligner » artificiellement les environnements.
- L'aval ne doit pas être promu si l'amont a testé un **contenu différent** de celui effectivement proposé (images, chart deps, BOM, valeurs critiques ou contrats).

Question ouverte : promotion de **snapshot complet de plateforme** entre environnements ou promotion **par couche/composant** ? Une granularité fine accélère les évolutions mais augmente le risque de combinaison incohérente. Conserver au minimum **un verrou de compatibilité inter-couches** et une BOM cohérente ; éviter un tag mutable par application sans contrôle de composition.

## 4. Convention de références : hypothèses, pas prescription

**Option A — Garder la convention actuelle et l'étendre :**
- Tags immuables : `dev-v*`, puis éventuels `test-v*`, `staging-v*`, `prod-v*`.
- Aliases : `dev-latest`, `test-stable`, `staging-stable`, `prod-stable`.
- `prod-working` conserve un rôle **distinct de « déployé »** : bookmark known-good manuel, pas un nouveau niveau du pipeline.

**Option B — Séparer clairement « release » et « canal de déploiement » :**
- Une seule famille de **releases immuables** (ex. `release/v...` ou release BOM à SHA), indépendante des environnements.
- Quelques pointeurs mutables explicitement gouvernés (ex. `deploy/dev`, `deploy/test`, `deploy/staging`, `deploy/prod`) vers ces releases.
- **Registres de qualification/receipts** en dehors du tag : telle release a réussi tels gates sur tel environnement/cluster, à telle date, avec tels artefacts.
- Un bookmark manuel known-good distinct pour la production, si jugé utile (`prod-working` ou nouveau nom discuté).

**Option C — GitOps promotion pilotée par fichiers déclaratifs :**
- Pour chaque cible, un manifeste de déploiement versionné/pinné (`environments/<env>/release.yaml` ou équivalent), référençant la BOM immuable.
- Les tags peuvent servir de **raccourcis de consultation**, et non de source directe pour ArgoCD. Les changements de pointeurs sont des PRs auditées et contrôlées.
- Avantages possibles : meilleure traçabilité, contrôles et rollback par PR ; inconvénients : plus de commits, risque de boucle de réconciliation et davantage de glue CI.

**À comparer sérieusement :** atomicité d'une bascule, gestion des refs Git mutables et du cache ArgoCD, révision exacte visible dans l'UI, concurrence entre promotions, idempotence des workflows, preuves signées/attestées, protection contre le déplacement non autorisé des tags, rollback, et résilience si GitHub ou le pipeline est indisponible.

Ne pas créer d'emblée toutes les familles de tags. Choisir d'abord le **modèle d'état** puis **un nommage qui lui corresponde**.

## 5. Automatisation progressive : décisions distinctes du nommage

Proposition d'étude des permissions par transition :

| Transition | Politique candidate, NON approuvée |
|---|---|
| `main → dev` | Automatique après CI de merge, si l'on maintient le comportement actuel |
| `dev → test` | Possiblement automatique **sur une release complète connue** et après gates de CI pertinents |
| `test → staging` | Possiblement automatique sous conditions plus strictes (tests contractuels + fonctionnels, disponibilité, contrôle des dépendances) |
| `staging → prod` | **Autorisation humaine explicite maintenue** par défaut, portant sur le snapshot exact et ses preuves |
| `prod → known-good` | Marqueur manuel après constat réel que la version fonctionne, **jamais** un simple effet d'une CI verte |

Pour chaque transition, conserver : candidat SHA/BOM exact, source/amont, destination, validation requise, preuve associée, acteur autorisé, horodatage, rollback, prise en compte des fenêtres de maintenance et tests de non-régression. Les promoteurs automatiques ne doivent pas transformer « tests verts » en « cluster prêt », ni redéployer des volumes persistants incompatibles.

Il faut traiter les canaux de qualification à l'échelle **d'un nombre maîtrisé de clusters/cibles**, pas générer un cluster par commit ni déployer systématiquement toute la plateforme si l'on ne teste qu'un composant. Les stratégies de preview sélectives (#3588, #3590) méritent d'être réexaminées avec cette proposition, sans se substituer aux preuves sur les environnements complets.

## 6. Supprimer les overlays : hypothèse à tester, pas acquis

L'existence de **contrats entre couches** peut supprimer beaucoup d'overlays devenus nécessaires uniquement pour traduire un fournisseur particulier (`truenas-iscsi-*`, NFS hardcodé, chemins de secrets, etc.). Elle **ne supprime pas automatiquement** les différences de déploiement réelles :

- DNS, domaines, ingress et routes ; endpoints externes propres à un cluster ;
- références de secrets, certificats, rotation, identité, politiques réseau, isolation des données ;
- dimensionnement, quotas, réplication, backup et règles de disponibilité ;
- activation délibérée d'un workload, canary, hibernation, PVC existants, schémas de données et stratégie de migration ;
- contraintes de bootstrap du Core et wiring système.

**Comparer au moins trois variantes** avant de conclure :
1. **Bases + overlays minimaux** : retirer les patches fournisseur/duplicatifs, conserver uniquement les vrais écarts de politique et de cible.
2. **Bases paramétrées / composition explicite** : ApplicationSet + Kustomize components/Helm values, profils d'environnement et bindings matériels internes aux couches, avec rendu final reproductible.
3. **Génération déterministe sans overlays applicatifs** : toutes les applications consomment les mêmes contrats, et l'orchestrateur génère les différences autorisées à partir de `cluster.name` et `environment` et de l'état désiré approuvé.

**Critère de décision :** coût total de maintenance, lisibilité de ce qu'ArgoCD applique réellement, testabilité des rendus, sécurité, impossibilité de cross-mount `dev`/`prod`, facilité de reprise manuelle et capacité à migrer sans recréer les ressources stateful. Éviter une génération opaque plus difficile à auditer que quelques overlays.

`perso-charchess` doit être testé ici comme **cible d'affectation / sélection d'applications**, éventuellement avec des politiques d'accès propres, **sans imposer** une variante complète de configuration fonctionnelle à toute la plateforme.

## 7. Faire respecter les contrats dans CI : objectif ferme à étudier dans sa mise en œuvre

La **garantie de conformité** ne peut pas être seulement documentaire : concevoir des tests qui refusent les changements cassant l'interface Core → Infra → Applications.

### Niveau A — Contrats et rendus statiques (PR / CI, obligatoire à terme)

- **Catalogue de contrats versionné** et schémas validés : nom/garanties des classes de stockage, access modes/RWO/RWX, `Retain/Delete`, expansion, snapshots si garantis, IngressClass, ClusterIssuer, stores secrets, endpoints et interfaces des services partagés ; propriété et compatibilité de chaque contrat.
- **Rendu effectif complet** pour chaque environnement/cible supporté : aucune référence à une classe/issuer/service absent, aucune collision de noms, CRD avant Custom Resource, namespace et propriétaire explicites.
- **Interdiction de fuite d'implémentation** dans les couches supérieures sauf exemption documentée ; p. ex. classes TrueNAS et adresses NAS dans une application, sans imposer aux données préexistantes une migration automatique.
- **Compatibilité des révisions** et cohérence BOM : tags, images et chart versions épinglés, aucune combinaison de couches non validée, aucune dépendance circulaire bootstrap.
- **Tests négatifs** : régression d'un nom public, changement destructif de politique `Retain`, absence de capacité requise, backend de secours non autorisé, secret de production référencé depuis `dev`, cross-namespace interdit, permissions trop ouvertes, perte de protection contre `prune`.
- **Publication d'un rapport** : par snapshot et cible, conformité des contrats, diffs de rendus, gates passés, preuves manquantes. Un check absent, annulé ou échoué est un échec, pas un faux vert.

### Niveau B — Intégration sur cluster jetable (CI ou lab)

- CRDs/operators, ArgoCD/ApplicationSet et ordonnancement ; création des ressources des couches inférieures avant leurs consommateurs ;
- création/déploiement/accès sur des ressources de **test isolées**, idempotence, retour arrière, permissions ;
- simulateurs ou backends substituables uniquement lorsqu'ils vérifient effectivement le **contrat** ; pas de validation trompeuse d'un driver réel via un faux CSI.

### Niveau C — Conformité physique réelle (hors CI standard, avant promotions concernées)

- stockage réel : provisionnement, attach/mount, données, multi-attach autorisé/refusé, reboot, rétention, snapshots/restauration et panne réseau ;
- accès TrueNAS/OpenBao/Cilium, ACL et certificats, disponibilité des services, recovery test, smoke applicatifs représentatifs ;
- état ArgoCD `Synced` / `Healthy` **et** preuve fonctionnelle ; archivage des résultats par **snapshot exact + cible réelle**.

**La CI GitHub ne possède pas nécessairement d'accès à ces clusters physiques** : ne jamais présenter un rendu Kustomize ou un test de simulation comme preuve NFS/iSCSI ou de restauration. La promotion doit demander explicitement la preuve physique lorsque le changement impacte cette garantie ; si la preuve manque, bloquer ou requérir une exception humaine tracée, pas l'inventer.

### Versionnement des contrats et compatibilité

Une modification d'un nom ou des garanties d'un contrat public est un changement d'API : prévoir **versionnement, période de coexistence, tests de consommateurs et plan de migration**, puis suppression contrôlée. Un contrat standard n'est utile que s'il est réellement fourni dans la destination visée et compatible avec les données existantes.

## 8. Questions ouvertes et critères d'acceptation pour la future étude

1. `perso-charchess` représente quoi exactement : environnement, cible, scope personnel ou canal ? Plusieurs clusters du même environnement ont-ils vocation à pointer sur la même release ?
2. Promotion **globale** d'une BOM cohérente ou progression **par couche** avec verrou de compatibilité ? Définir l'objet source de vérité (tag, fichier Git, BOM/manifest).
3. « En retard » = ancêtre/égal, chaîne de preuves, âge maximal, autorisation et rollback ? Préciser les exceptions et éviter les classements par noms de tags.
4. Quelle différence entre **déployé**, **testé**, **qualifié**, **approuvé**, **known-good** et **rollback-safe** ? Quels reçus/digests persistants pour chacune ?
5. Quels gates peuvent avancer **automatiquement** et sous quelles permissions ? Maintenir par défaut l'autorisation humaine sur toute promotion de production.
6. Les overlays sont-ils réellement supprimables, ou seulement **réductibles** aux bindings et politiques utiles ? Comparer le diff de rendu d'au moins deux applications dont une stateful.
7. Quels contrats sont opposables en CI, quel système les publie, et quels tests ont besoin de ressources physiques ? Quel bootstrap indépendant rend l'ensemble testable et récupérable ?
8. Comment déplacer le mode actuel `main`/`dev-v*`/`dev-latest`/`prod-v*`/`prod-stable`/`prod-working` **sans rupture** des ArgoCD Applications, anciens tags, release receipts ou rollback ?
9. Comment conserver les garde-fous et les snapshots cohérents en cas de séparation multi-repos (#3959/#3960) ?

**Critères pour valider le design avant de lancer une migration** : schéma de références et états précis, graphe des autorisations, preuves requises, matrice de compatibilité des contrats, étude d'impact des overlays, tests de rendu multi-cibles, plan de migration/rollback, séparation claire cible/environnement/domaine, risques de sûreté des données, et preuve par un prototype jetable.

**Rappel :** étude seulement ; pas de nouveaux tags, pas de nouveau pipeline, pas de retrait d'overlays, pas de modification de stockage ou de clusters et pas de promotion automatique créée par ce document. Les [arbitrages `layer` / `domain` en cours](https://github.com/charchess/vixens/pull/4197) demeurent un chantier documentaire séparé.
