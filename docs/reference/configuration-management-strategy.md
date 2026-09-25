# Configuration management strategy

**Status:** Active  
**Last Updated:** 2026-09-25  
**Related:** [ADR-013](../adr/013-layered-configuration-disaster-recovery.md), [ADR-014](../adr/014-litestream-backup-profiles-and-recovery-patterns.md), [ADR-018](../adr/018-openbao-external-secrets-and-nas-fqdn.md)

Vixens sépare les données selon leur nature au lieu d'utiliser un mécanisme unique pour tout.

## Modèle

| Type | Source de vérité | Projection / persistance | Récupération |
|---|---|---|---|
| Secrets / credentials | OpenBao | ESO → Kubernetes Secret | resync depuis OpenBao |
| Config statique non sensible | Git | ConfigMap / fichier manifest | ArgoCD |
| Config applicative mutable | stockage persistant | pattern de backup de l'app | restore testé |
| SQLite | stockage persistant | DataAngel/Litestream lorsqu'utilisé | restore Litestream |
| PostgreSQL | CloudNativePG | volumes CNPG | mécanismes CNPG |
| Fichiers partagés / médias | NAS / stockage partagé | NFS/CSI selon l'app | stratégie NAS |
| Cache / scratch | aucune | `emptyDir` ou équivalent | recréation |

Le chantier stockage TrueNAS/CSI peut faire évoluer les StorageClass et backends ; les manifests courants restent la source de vérité pour ce détail.

## Couche 1 — Secrets : OpenBao + External Secrets

Les valeurs sensibles vivent dans OpenBao et sont projetées via External Secrets Operator.

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: myapp-secrets
  namespace: myapp
spec:
  refreshInterval: 60s
  secretStoreRef:
    name: openbao
    kind: ClusterSecretStore
  target:
    name: myapp-secrets
    creationPolicy: Owner
  dataFrom:
    - extract:
        key: vixens/prod/apps/60-services/myapp
```

Règles :

- zéro valeur sensible dans Git ;
- une ressource `ExternalSecret` décrit seulement le contrat de synchronisation ;
- le workload consomme le `Secret` Kubernetes généré ;
- les différences dev/prod portent de préférence sur le chemin OpenBao, pas sur deux manifests presque identiques ;
- toute credential exposée publiquement doit être révoquée/rotatée, même si elle est ensuite supprimée de Git.

Voir [`docs/guides/secret-management.md`](../guides/secret-management.md).

## Couche 2 — Config statique

Une configuration qui est :

- non sensible ;
- reproductible ;
- revue avec le code ;

appartient dans Git, généralement via ConfigMap ou fichier monté depuis une ressource déclarative.

Ne pas pousser dans OpenBao une configuration non sensible uniquement pour éviter de la versionner.

## Couche 3 — Config applicative mutable

Les applications qui modifient elles-mêmes leur configuration ont besoin d'un stockage persistant et d'une stratégie de restauration.

Principe :

```text
PVC / stockage applicatif
        ↓
backup adapté au format
        ↓
stockage de sauvegarde
        ↓
restore testé
```

Pour les applications utilisant le pattern DataAngel du dépôt, l'overlay configure les chemins et credentials tandis que le composant partagé fournit la mécanique commune. Chercher une application active utilisant DataAngel avant d'en créer une nouvelle intégration.

## SQLite

SQLite nécessite une sauvegarde consciente du WAL ; une copie de fichier périodique n'est pas suffisante pendant que la base est active.

Lorsque le pattern Litestream/DataAngel est utilisé :

- préciser le chemin réel de la DB ;
- stocker les credentials S3 via OpenBao/ESO ;
- tester un restore sur une copie/instance contrôlée ;
- ne pas mélanger les fichiers SQLite actifs avec un `rclone sync` générique.

## PostgreSQL

Les bases gérées par CloudNativePG utilisent les mécanismes CNPG pour réplication, sauvegarde et restauration.

Ne pas ajouter DataAngel/Litestream autour d'un PostgreSQL CNPG : le moteur possède déjà les primitives appropriées.

## Fichiers partagés et médias

Les gros volumes partagés ne doivent pas être recopiés dans les PVC applicatifs par défaut. Utiliser le backend partagé/CSI prévu par l'application et la stratégie du NAS.

Les chemins, permissions et StorageClass peuvent changer pendant le chantier TrueNAS/CSI ; éviter de figer dans la documentation générale une adresse ou un chemin qui n'est pas un contrat durable.

## Secrets de backup

Les clés S3/MinIO nécessaires aux sauvegardes suivent exactement la même règle que tout autre secret :

```text
OpenBao → ExternalSecret → Kubernetes Secret → DataAngel/Litestream/CNPG
```

Pas de `Secret` avec valeurs réelles dans Git, pas de credentials dans un script ou une documentation.

## Matrice de décision

| Situation | Pattern |
|---|---|
| API key / password / token | OpenBao + ExternalSecret |
| configuration statique non sensible | Git / ConfigMap |
| configuration mutable | PVC + backup adapté |
| SQLite | PVC + Litestream/DataAngel si pattern applicable |
| PostgreSQL | CloudNativePG |
| gros fichiers partagés | stockage partagé / NAS / CSI |
| cache | `emptyDir` / recréation |

## Validation d'une stratégie de persistance

Pour une nouvelle application, répondre explicitement à :

1. Qu'est-ce qui est la source de vérité ?
2. Qu'est-ce qui peut être recréé ?
3. Qu'est-ce qui doit survivre à la perte d'un pod ? d'un nœud ? d'un PVC ? du cluster ?
4. Où sont les credentials du mécanisme de backup ?
5. Quel est le RPO/RTO attendu ?
6. Quand le restore a-t-il été testé pour la dernière fois ?
7. Le mécanisme est-il représenté dans Git ou dépend-il d'une procédure manuelle non documentée ?

Un backup non restauré en test n'est pas une garantie de récupération.

## Historique

L'ancien guide `backup-restore-pattern.md` décrivait une orchestration rclone/CronJob et Infisical. Il est conservé sous forme de stub historique uniquement et ne doit plus être utilisé comme template.
