# Proposition — Couches GitOps et contrats de plateforme neutres

**Statut : PROPOSITION À ÉTUDIER — non acceptée, non exécutable**  
**Date :** 2026-10-24  
**Sujet de suivi :** [vixens#3959](https://github.com/charchess/vixens/issues/3959)  
**Dépendance CI/release multi-repo :** [vixens#3960](https://github.com/charchess/vixens/issues/3960)

> Ce document conserve les orientations issues de la réflexion sur la refonte des environnements et le découpage de la plateforme. Il **ne remplace pas** l'architecture active décrite dans [docs/architecture.md](../architecture.md) et les manifests courants. Ce n'est ni un ADR accepté ni une décision de migration. La mise en attente explicite de #3959 reste applicable : **aucune migration, réorganisation des dépôts, modification de manifests, de ressources live ou de mécanismes de promotion n'est autorisée par cette note**.

## 1. Objectif et principes de conception

La cible envisagée sépare les responsabilités du bootstrap, de la plateforme Kubernetes, des services mutualisés et des applications. Chaque couche consomme les **interfaces et garanties publiées** par la couche inférieure, et non ses détails d'implémentation.

Objectifs :

- permettre de livrer plusieurs clusters sans recréer une architecture applicative différente par environnement ;
- neutraliser les noms et références de fournisseurs dans les workloads chaque fois qu'une abstraction stable et vérifiable est possible ;
- conserver un **petit catalogue explicite de contrats**, plutôt qu'une classe ou un overlay pour chaque usage ;
- limiter les dimensions globales du déploiement à **l'identité du cluster** fournie par Terraform et à son **environnement** logique ;
- assurer le bootstrap et la restauration sans dépendances circulaires ;
- préserver les garanties GitOps actuelles : état désiré versionné, propriétaires de ressources uniques, releases reproductibles, CI et promotion production explicitement autorisée.

**Règle directrice :** même contrat public sur les clusters qui supportent un composant, éventuellement avec des implémentations matérielles et des politiques d'environnement différentes. Les différences d'implémentation ne doivent pas fuiter dans les applications. Un cluster ne pouvant satisfaire un contrat requis échoue **explicitement**, sans substitution silencieuse.

## 2. Couches envisagées

```text
[ 3 — GitOps Applications ]
        consomme les contrats Core et les services Infra
                    ↑
[ 2 — GitOps Infra / Shared ]
        services transversaux : clusters PostgreSQL partagés,
        Redis, identité, observabilité, etc.
                    ↑
[ 1 — GitOps Core ]
        capacités Kubernetes : CSI/stockage, secrets,
        réseau, certificats, politiques, CRDs/operators essentiels
                    ↑
[ 0 — Terraform / TerraVixens ]
        machines/OS/réseau initial, Kubernetes, identité du
        cluster, environnement, prérequis du nœud,
        bootstrap minimal ArgoCD et accès nécessaires
```

| Couche | Fournit | Ne doit pas imposer aux couches supérieures |
|---|---|---|
| 0 / Terraform | Cluster exploitable et bootstrap autonome, nom et environnement | Détails machines comme des paramètres applicatifs |
| 1 / Core | Capacités Kubernetes et contrats publics | Fournisseur CSI, chemin NFS ou adresses NAS dans chaque application |
| 2 / Infra | Services mutualisés, endpoints et politiques de consommation | Topologie et mise en œuvre interne des services partagés |
| 3 / Applications | Workloads, usages et données spécifiques | Déploiement/maintenance du CSI ou de l'operator transverse |

La frontière des **operators** dépend du rôle réel du composant, et non de son emplacement historique. **Arbitrage actuel : l'operator CNPG et ses CRDs restent en couche Infra**, comme les instances PostgreSQL partagées ; cette affectation est **à réexaminer lors de la conception détaillée**, sans présumer de leur déplacement en Core. Kyverno reste également **Infra** par décision explicite, même s'il intervient dans l'admission du cluster.

Ces couches sont **logiques**, pas nécessairement quatre dépôts ni quatre applications ArgoCD. Le découpage physique en repositories demeure une décision distincte de #3959.

## 3. Identité et environnements : seulement deux dimensions globales

- **`cluster.name`** : identité stable du cluster, attribuée au bootstrap par Terraform (p. ex. `umi`).
- **`cluster.environment`** : politique logique de déploiement (`dev`, `test`, `staging`, `prod`).

**Ne pas créer un fichier YAML global répliquant** `capabilities.storage`, `capabilities.database`, `features.`, etc., cluster par cluster. Le catalogue de capacités relève des **contrats de plateforme** ; il est censé être le même partout où la plateforme est déployée.

Le choix du backend, les CIDR, les chemins de stockage, les certificats, les secrets, le dimensionnement ou la topologie demeurent des **paramètres internes aux couches responsables**, éventuellement spécifiques au cluster ou à l'environnement. Ils ne constituent pas des dimensions supplémentaires du *contrat public*.

### Effet sur les overlays (non arbitré)

L'organisation actuelle `base/overlays/dev` et `base/overlays/prod` doit être **réexaminée**, pas supprimée d'office.

- `environment` définit d'abord les **politiques** : promotion, sécurité, dimensionnement, nombre de réplicas, stratégies de disponibilité, etc.
- `cluster.name` identifie une instance de ces politiques et peut sélectionner un **binding technique local** (backend, endpoint, sous-réseau, secrets), sans redéfinir le contrat applicatif.
- Base/overlay Kustomize, Helm et ApplicationSet sont des outils d'assemblage possibles ; **le schéma de templating, de génération et de sélection reste ouvert**.
- Éviter à la fois la duplication massive de bases par environnement **et** la création d'un énorme fichier de variations qui centralise tous les détails matériels.

Aucun refactoring des overlays ne doit être lancé avant d'avoir choisi un mécanisme reproductible, validé le rendu et vérifié la propriété GitOps des ressources.

## 4. Conventions de nommage : neutralité et faible cardinalité

### Deux espaces à ne pas confondre

1. **Contrat de capacité / classe** : *quel type de ressource et quelles garanties ?*
2. **Instance de ressource / partage / claim** : *quelle donnée ou quel service précis est consommé ?*

Un type de stockage ne doit pas être dupliqué pour chaque répertoire ou chaque équipe. Inversement, plusieurs instances distinctes peuvent tout à fait consommer le même type.

### Proposition minimale de classes de stockage

Les noms suivants sont des **exemples à arbitrer**, pas des noms approuvés ni des ressources à créer immédiatement.

| Contrat candidat | Sémantique attendue |
|---|---|
| `storage-rwo-retain` | Volume à accès RWO, politique de conservation `Retain` |
| `storage-rwo-delete` | Volume à accès RWO, politique `Delete` |
| `storage-rwx-retain` | Volume partageable RWX, politique `Retain` |
| `storage-rwx-delete` | Volume partageable RWX, politique `Delete` |

**Ne pas supposer** que `RWO` signifie stockage bloc, ni que `RWX` implique NFS : ces termes expriment une **capacité Kubernetes**, pas un protocole. La sémantique précise des modes d'accès, de la rétention et du support `ReadWriteOncePod` doit être documentée pour chaque contrat.

Critères pour ajouter une nouvelle classe :

- la différence correspond à une **garantie utile, observable, testable et stable** (par ex. performances minimales, topologie, résilience, chiffrement, type de volume nécessaire à certains usages) ;
- on dispose de consommateurs nécessitant effectivement cette garantie ;
- la distinction peut être honorée dans les environnements concernés ;
- la différence ne représente **pas seulement** le backend choisi, l'application ou le nom d'un dossier.

Ainsi, `storage-slow-iscsi-retain` surqualifie probablement le mécanisme si `slow` et `iscsi` ne définissent pas des garanties contractuelles nécessaires. `storage-media-retain` confond **usage** et **classe**. Si des niveaux `fast`/`slow` sont un jour nécessaires, les définir d'abord de façon mesurable, sans inventer un produit cartésien d'options.

**Le catalogue attendu est explicite et réduit**, avec pour chaque contrat : nom, modèle d'accès, politique de rétention, expansion/snapshots si garantis, topologie et disponibilité, contraintes d'usage, obligations du fournisseur, tests de conformité et procédure de dépréciation. Une simple convention de nommage ne suffit pas à garantir l'équivalence réelle.

### Évolution sans migration accidentelle

Les classes existantes (par ex. `truenas-iscsi-retain`) et les PV/PVC liés sont des **ressources réelles**, pas des alias textuels. Créer de nouvelles classes neutres ne modifie pas les PV existants ; ne pas renommer, réaffecter ou reprovisionner automatiquement leurs volumes, IQN, handles, chemins ou stratégies `Retain`. Toute migration requiert inventaire, preuves d'intégrité et plan de retour arrière.

## 5. Les partages NFS : question ouverte, ne pas imposer une abstraction prématurée

### Problème constaté dans le dépôt

- `apps/20-media/sabnzbd/base/deployment.yaml` monte directement `nas.truxonline.com:/mnt/tank/scratch/downloads`.
- `apps/20-media/sonarr/base/deployment.yaml` utilise ce même partage et `/mnt/tank/data/media`.
- `apps/01-storage/nfs-storage/base/static-pvs.yaml` déclare déjà plusieurs **PV NFS statiques** pour des arbres existants (`downloads`, `media`, `documents`, etc.).
- `apps/01-storage/nfs-storage/README.md` explicite des bindings PVC→PV statiques.

L'application qui embarque `nas.truxonline.com` et `/mnt/tank/...` est directement couplée à la topologie physique. **Mais déplacer ce couplage ne justifie ni une StorageClass par partage, ni forcément un nouveau contrôleur/CRD.**

### Trois notions séparées

| Notion | Exemple candidat | Nature |
|---|---|---|
| Classe/capacité | `storage-rwx-retain` | Type de ressource et garanties |
| Partage/donnée identifiée | `downloads`, `media`, `documents` | **Instance** de donnée existante (identité, autorisations, cycle de vie) |
| Montage par application | `/downloads`, `/tv` | Chemin interne au conteneur, propre au consommateur |

Le nombre de **partages** peut naturellement être grand sans gonfler le catalogue de **contrats de classes**.

### Options à comparer — **aucun choix validé**

**Option A — PV/PVC statiques et noms logiques.** L'application référence un PVC connu dans son namespace, p. ex. `share-downloads`, et la plateforme matérialise le PV correspondant au vrai export. Cela neutralise le serveur/chemin dans le workload ; en revanche, le PVC est **namespace-scoped** et chaque PV ne se lie qu'à **un seul PVC** : si plusieurs namespaces consomment le même export, il faut définir une stratégie explicite de PV/bindings distincts pointant vers les mêmes données. Accès RWX, permissions, conflits d'écriture, rétention et ownership doivent être examinés. Un PV statique exposant un partage existant n'est **pas** une demande de provisionnement dynamique CSI.

Exemple d'expression côté application (**illustratif, pas prescription**) :

```yaml
volumes:
  - name: downloads
    persistentVolumeClaim:
      claimName: share-downloads
```

**Option B — Montages NFS natifs avec abstraction de configuration.** Conserver `volumes[].nfs` quand cela convient aux partages statiques existants, mais déplacer le mapping serveur/chemin dans une composition ou un patch de plateforme, sans duplication dans les bases applicatives. Moins d'objets PV/PVC, mais la frontière d'ownership et la réutilisation entre applications doivent rester lisibles et testables ; ne pas créer une « magie Kustomize » opaque.

**Option C — Abstraction de partage dédiée seulement si besoin avéré.** Un catalogue/contrat d'instances, éventuellement appuyé par génération de bindings ou un contrôleur, peut devenir pertinent si le nombre de consommateurs, la délégation, l'isolation ou le cycle de vie le justifient. **Ne pas créer un CRD/operator par anticipation.** L'abstraction doit réduire la complexité totale, pas la déplacer.

Les **PVC RWX provisionnés dynamiquement** constituent encore un autre cas : ils sont adaptés à la création de **nouveaux volumes**, mais ne sont pas automatiquement équivalents à des exports NFS préexistants partagés entre applications.

### Décision différée et critères de choix

**Le mécanisme NFS n'est PAS retenu à ce stade.** Il faudra confronter les options A/B/C à des cas physiques :

- un même partage consommé par SABnzbd/Sonarr/Radarr dans un namespace ;
- un partage consommé depuis plusieurs namespaces ;
- exports préexistants vs nouveaux volumes alloués ;
- isolation `dev`/`prod`, accès RO/RW, UID/GID et export ACL ;
- persistance et préservation des données, sécurité de suppression, complexité des manifests et expérience de récupération après panne.

On accepte le **principe de noms logiques neutres** et la distinction « classe ≠ partage », **pas encore la méthode d'exposition NFS**.

## 6. Bootstrap, dépendances et disponibilité réelle

- Terraform doit permettre le démarrage et la réparation du cluster sans dépendre d'un workload qui exige lui-même les CSI, DNS ou secrets indisponibles.
- Les CSI peuvent rester **gérés par GitOps** ; ils doivent aussi rester restaurables par maintenance lorsque l'état réel l'impose. Politique `prune` / finalizers / `selfHeal` à examiner séparément pour les composants critiques.
- Les sync waves ordonnent des objets, mais ne remplacent pas une **preuve fonctionnelle** : Secret réellement matérialisé, driver CSI enregistré, volume montable, CRDs et operator prêts avant leurs instances, etc.
- Le plan de récupération est un livrable architectural : accès aux nœuds, initiateur iSCSI, réseau, secrets bootstrap, intégrité des volumes, accès à l'API Kubernetes et capacité à suspendre/rétablir des réconciliations.
- Ne pas confondre **Git source de l'état désiré** et **Git unique moyen d'intervenir** : la maintenance de l'état réel est permise dans son périmètre (voir les modes proposés dans #4191 / PR #4192, sans supposer leur merge).

## 7. Cohérence GitOps, migration et non-objectifs

L'idée des quatre couches **ne modifie pas** à elle seule la mécanique de release multi-repo de #3959 : si les sources sont réparties, conserver une **BOM/lock de SHAs immuables**, permettant de reconstruire exactement un snapshot `dev-v*` ou `prod-v*`, et les gates de #3960. Les promotions en production restent explicitement autorisées.

Avant toute future bascule : inventaire de l'ownership ArgoCD, comparaison des rendus, preuve d'absence de suppression/recréation involontaire, sauvegardes/rollback, validation physique des volumes et services. Séparer la décision de **découpage des responsabilités** de celle de **découpage des dépôts**.

**Hors périmètre du présent document :** création de nouvelles StorageClasses, mise en place d'un catalogue de partages, remplacement des montages NFS, migration CSI, modification Terraform/ArgoCD, choix final des overlays, split multi-repo, synchronisation production. Aucun de ces gestes n'est déclenché par la rédaction ou le merge éventuel de cette note.

## 8. Questions de conception à reprendre lors du dégel

1. Arrêter un premier **catalogue contractuel** minimal, avec garanties vérifiables (stockage, ingress, secrets, certificats, éventuellement services de données).
2. Décider du **modèle de nommage** des contrats : préfixes, traits d'union, sémantique exacte de RWO/RWX, rétention, qualité de service, dépréciation.
3. Décider **après étude** de l'exposition des partages NFS statiques : PVC/PV, montage natif avec composition, autre ; arbitrer ownership, partage multi-namespace, ACL, accès concurrents et récupération.
4. Repenser la composition des overlays avec seulement les **deux entrées globales** `cluster.name` / `environment`, sans casser les particularités techniques légitimes.
5. Réévaluer seulement si nécessaire la frontière **Core/Infra** des operators et services mutualisés, **sans reclasser CNPG ou Kyverno par défaut** (CNPG actuellement Infra, qualification à approfondir ; Kyverno confirmé Infra).
6. Certifier l'**indépendance du bootstrap et de la restauration**, les health gates entre couches et la cohérence des snapshots GitOps.
7. Décider **seulement ensuite** des chemins de migration et des éventuels nouveaux dépôts selon #3959/#3960.

**Contexte de gouvernance :** #3959 est explicitement **PARK / DEFER** jusqu'à une nouvelle décision du propriétaire ; cette note est une trace de conception, non une instruction pour entreprendre des travaux.


## 9. Inventaire ArgoCD : décisions de classification et axes `layer` / `domain`

**Arbitrages de l'opérateur — 2026-10-24.** Les choix ci-dessous complètent la piste d'architecture, **sans déclencher de migration**. Les labels ArgoCD actuellement appliqués sont ceux de [#3805](https://github.com/charchess/vixens/issues/3805) : `vixens.io/domain: core|infra|fabric|personal`. Ils servent à **classer et filtrer** les Applications ; ils ne sont ni des `AppProject` ArgoCD ni des règles RBAC.

### 9.1 Affectations retenues et points laissés ouverts

| Composant | Domaine `vixens.io/domain` visé | Couche `vixens.io/layer` envisagée | Décision ou réserve |
|---|---|---|---|
| TrueNAS CSI, stockage, fondations GitOps | `core` | `core` | Cohérent avec la cible ; la compatibilité des contrats reste à démontrer |
| CNPG operator et CRDs | `infra` | `infra` | **Conserver en Infra pour le moment** ; la question de sa qualification est **à étudier**, sans transfert automatique vers Core |
| PostgreSQL partagé | `infra` | `infra` | Service transverse Infra confirmé |
| Kyverno | `infra` | `infra` | **Décision ferme : reste en Infra**, sans reclassement Core par sa seule fonction d'admission |
| Mosquitto | **`personal`** | `applications` | **À reclasser** : application/service personnel, pas service transversal global |
| `media-shared-secrets` | **`personal`** | `applications` | **À reclasser** : les consommateurs sont des applications personnelles ; « partagé » ne signifie pas « Infra » |
| AdGuard Home | **`infra`** | `infra` | **Conserver pour le moment** ; son **retrait éventuel** est à réfléchir séparément, sans décision immédiate |
| `whoami` | `infra` actuellement | Sans objet après retrait | **À retirer** du catalogue/déploiement ; prévoir une suppression GitOps contrôlée, pas une suppression immédiate des objets live |
| TXO Fabric / Hermes Fabric | `fabric` | `applications` par défaut | Domaine produit indépendant ; une exception de couche devra être motivée par une responsabilité réellement transverse |
| Home Assistant, Frigate, Trilium, etc. | `personal` | `applications` | Cohérent ; ne pas créer de nouvelle couche pour chaque usage |

**Différence essentielle :** une application *utilisée par plusieurs applications d'un même domaine* (par ex. `media-shared-secrets`) ne devient pas automatiquement une **infrastructure de plateforme**. On classe par **responsabilité et raison d'être**, pas simplement par présence du mot `shared` dans le nom.

Pour `whoami`, l'intention est de **retirer** le composant, pas de le rebaptiser. Son éventuelle désinstallation doit tenir compte de l'ownership ArgoCD, du `prune`, des finalizers et des conséquences réelles ; aucune action n'est exécutée par ce document.

### 9.2 Séparer deux axes de classification, sans imposer de refonte immédiate

La direction **`layer` + `domain`** est retenue **au niveau conception** : elle répond à deux questions distinctes.

- **`vixens.io/layer`** : *à quel niveau technique de la plateforme le composant appartient-il ?* Valeurs **candidates** : `core`, `infra`, `applications`. Terraform / couche 0 se situe hors des Applications Kubernetes et n'a pas à être artificiellement labellisé.
- **`vixens.io/domain`** : *pour quelle responsabilité ou finalité opérationnelle le composant existe-t-il ?* On conserve **provisoirement** le vocabulaire existant `core|infra|fabric|personal` pour ne pas casser le filtrage et la validation CI. Il mélange aujourd'hui technique et finalité, et sa normalisation à terme (`platform`, `fabric`, `personal` ou autre) **reste à concevoir**.

Exemple **illustratif**, non appliqué :

```yaml
metadata:
  labels:
    vixens.io/layer: applications
    vixens.io/domain: personal
```

**Ne pas transformer automatiquement** `domain=infra` en `layer=infra` ni `domain=fabric` en `layer=applications` sans inventorier les responsabilités concrètes : les axes doivent devenir véritablement indépendants. Le tableau précédent décrit les **affectations envisagées** pour les cas discutés, pas un algorithme universel de traduction.

Ne pas créer d'`AppProject` ArgoCD ni modifier les GitHub Projects à cette occasion. La question des **GitHub Projects et de leur organisation est explicitement hors périmètre** ; elle sera reprise lors d'un **point d'organisation distinct**.

### 9.3 Constat sur l'implémentation actuelle et conditions d'une future évolution

La classification actuelle est principalement ajoutée par les patches de `argocd/overlays/dev/kustomization.yaml` et `argocd/overlays/prod/kustomization.yaml`. La CI de `.github/workflows/validate.yaml` valide déjà les **quatre domaines actuels** ; changer les labels sans mettre à jour cette garde reviendrait à casser le contrat de validation.

Le diagnostic statique a relevé une **duplication de patches en dev** (certaines classifications sont répétées) et une **question distincte d'ownership de l'Application racine** `vixens-app-of-apps` déjà suivie dans [#3823](https://github.com/charchess/vixens/issues/3823).

Lors d'un futur chantier explicitement autorisé :

1. Valider le **vocabulaire exact et la sémantique** de `layer` et `domain`, ainsi que le système qui en possède la vérité.
2. Reclasser `mosquitto` et `media-shared-secrets` en `personal`, en préservant les ressources runtime ; laisser `CNPG`, `Kyverno` et `AdGuard Home` en `infra` tant qu'aucune autre décision n'est prise.
3. Préparer le retrait **contrôlé** de `whoami`, après inventaire de ses usages et évaluation du risque ArgoCD.
4. Éliminer les règles Kustomize dupliquées sans modifier la classification effective ; privilégier un propriétaire/générateur unique plutôt qu'une seconde copie divergente.
5. Étendre les tests de rendu et la CI au nouvel axe `layer`, lorsque son schéma sera validé. Ne **pas** convertir des labels de classification en mécanismes de sécurité sans décision et analyse dédiées.
6. Traiter l'Application racine avec #3823 et distinguer source Git, labels et ownership bootstrap.

**Non-objectifs immédiats :** modification des labels/manifests réels, suppression de `whoami`, changement de `AppProject`, changement des GitHub Projects, migration d'overlays ou de dépôts. #3959 reste **PARK / DEFER** jusqu'à décision explicite du propriétaire.
