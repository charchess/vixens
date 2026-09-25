# Secret Management Guide

Vixens utilise **OpenBao** comme source de vérité pour les valeurs secrètes et **External Secrets Operator (ESO)** pour les projeter vers des `Secret` Kubernetes.

## Architecture canonique

```text
OpenBao (KV v2)
    ↓
ClusterSecretStore/openbao
    ↓
ExternalSecret
    ↓
Secret Kubernetes
    ↓
Pod
```

Les valeurs secrètes ne sont jamais stockées en clair dans Git.

## ClusterSecretStore

Le store canonique est :

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: openbao
```

Il est défini dans `apps/00-infra/openbao/` et pointe vers le moteur KV v2 d'OpenBao.

## Convention des chemins

Utiliser des chemins explicites par environnement et application :

```text
vixens/dev/apps/<category>/<app>
vixens/prod/apps/<category>/<app>
```

Exemple :

```text
vixens/prod/apps/00-infra/cert-manager-webhook-gandi
```

## ExternalSecret

Pattern standard :

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: my-app-secrets
  namespace: my-namespace
spec:
  refreshInterval: 60s
  secretStoreRef:
    name: openbao
    kind: ClusterSecretStore
  target:
    name: my-app-secrets
    creationPolicy: Owner
    deletionPolicy: Retain
  dataFrom:
    - extract:
        key: vixens/dev/apps/<category>/<app>
```

Le chemin est adapté dans l'overlay prod pour pointer vers `vixens/prod/...`.

## Utilisation dans un workload

Le workload consomme uniquement le `Secret` Kubernetes matérialisé par ESO :

```yaml
env:
  - name: API_KEY
    valueFrom:
      secretKeyRef:
        name: my-app-secrets
        key: API_KEY
```

Une application ne doit pas connaître les détails d'authentification OpenBao.

## Validation

Observer l'état sans mutation persistante :

```bash
kubectl get clustersecretstore openbao
kubectl get externalsecret -A
kubectl describe externalsecret -n <namespace> <name>
kubectl get secret -n <namespace> <target-secret>
```

Ne jamais afficher ou copier la valeur d'un secret dans un ticket, une PR, un commit ou un log partagé.

## Bootstrap OpenBao

L'authentification ESO → OpenBao est actuellement un détail d'infrastructure géré séparément du contenu des `ExternalSecret`. Le contrat applicatif reste :

```text
ExternalSecret → ClusterSecretStore/openbao
```

Toute évolution du mécanisme d'authentification (par exemple token statique vers auth Kubernetes) doit préserver ce contrat autant que possible.

## Ancien modèle Infisical

L'ancien `InfisicalSecret` et l'Infisical Operator sont retirés du modèle actif. Les ADR, audits ou rapports historiques peuvent encore les mentionner pour conserver l'historique du projet ; ils ne constituent pas une procédure actuelle.

Pour tout nouveau secret, utiliser **OpenBao + External Secrets Operator**.
