# Secret management

Vixens utilise **OpenBao** comme source de valeurs secrètes et **External Secrets Operator (ESO)** pour les projeter dans Kubernetes.

La chaîne canonique est :

```text
OpenBao
  ↓
ClusterSecretStore/openbao
  ↓
ExternalSecret
  ↓
Kubernetes Secret
  ↓
workload
```

Git contient le **contrat** (où lire, quel Secret produire, quelles clés consommer), jamais les valeurs sensibles.

## Sources de vérité

- Store canonique : `apps/00-infra/openbao/base/cluster-secret-store.yaml`
- External Secrets Operator : applications ArgoCD `external-secrets`
- Secrets applicatifs : ressources `external-secrets.io/v1`, `kind: ExternalSecret`
- Décision d'architecture : `docs/adr/018-openbao-external-secrets-and-nas-fqdn.md`
- Ancienne architecture Infisical : `docs/adr/011-infisical-secrets-management.md` (**historique / superseded**)

Toujours relire les manifests courants avant de supposer l'endpoint OpenBao, la méthode d'authentification ou un chemin KV : ces détails peuvent évoluer.

## Pattern standard

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: my-app-secrets
  namespace: my-app
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
        key: vixens/prod/apps/60-services/my-app
```

L'application consomme ensuite le `Secret` Kubernetes normalement :

```yaml
env:
  - name: API_TOKEN
    valueFrom:
      secretKeyRef:
        name: my-app-secrets
        key: API_TOKEN
```

Ou via `envFrom` lorsque cela correspond au contrat de l'application.

## Convention des chemins

Les chemins actuellement utilisés suivent en général :

```text
vixens/<env>/apps/<category>/<app>
```

Exemples :

```text
vixens/dev/apps/00-infra/cert-manager-webhook-gandi
vixens/prod/apps/20-media/my-app
```

Ne pas inventer une autre arborescence sans vérifier les applications similaires et le contenu actuel du dépôt.

## Ajouter un secret applicatif

1. Vérifier un pattern existant dans la même catégorie.
2. Créer/mettre à jour la valeur dans OpenBao par un canal opérateur autorisé.
3. Déclarer un `ExternalSecret` dans Git.
4. Référencer le Secret Kubernetes généré depuis le workload.
5. Rendre le chemin dev/prod explicite dans l'overlay si nécessaire.
6. Construire les overlays Kustomize et laisser CI valider.
7. Après merge, vérifier ESO puis le comportement de l'application.

Aucune valeur secrète ne doit apparaître dans :

- Git ;
- Issues ou PR GitHub ;
- documentation ;
- skills/templates d'agent ;
- logs ou captures partagées ;
- exemples de test versionnés.

## Diagnostic

Les commandes d'observation sont compatibles avec le workflow GitOps :

```bash
kubectl get clustersecretstore openbao
kubectl describe clustersecretstore openbao

kubectl get externalsecret -A
kubectl -n <namespace> describe externalsecret <name>

kubectl -n <namespace> get secret <name>
kubectl -n <namespace> get secret <name> -o jsonpath='{.data}' | jq 'keys'

kubectl -n external-secrets get pods
kubectl -n external-secrets logs deploy/external-secrets --tail=200
```

Pour diagnostiquer, préférer l'existence des clés, les conditions ESO, les `resourceVersion`, les timestamps et le comportement applicatif plutôt que d'afficher les valeurs.

## Rotation

Procédure générique :

1. générer/rotater la nouvelle credential chez le fournisseur ;
2. mettre à jour la valeur correspondante dans OpenBao ;
3. attendre ou déclencher une réconciliation ESO compatible avec la version déployée ;
4. vérifier que l'`ExternalSecret` est `Ready` et que le `Secret` Kubernetes a changé ;
5. redémarrer/reconcilier uniquement les workloads qui ne relisent pas automatiquement le Secret ;
6. tester l'application ;
7. révoquer l'ancienne credential chez le fournisseur.

Une credential qui a été committée dans un dépôt public est **compromise** : supprimer le fichier courant ne suffit pas, car l'historique Git subsiste. Elle doit être révoquée ou rotatée.

## Bootstrap OpenBao

Le `ClusterSecretStore` peut lui-même nécessiter une identité bootstrap. Cette identité est une dépendance de plateforme et non un secret applicatif.

Règles :

- ne jamais la versionner ;
- limiter ses privilèges ;
- préférer une identité courte / native Kubernetes lorsque l'architecture le permet ;
- documenter sa procédure de récupération sans documenter sa valeur.

Le passage de l'authentification bootstrap actuelle vers une auth Kubernetes OpenBao est un chantier de sécurité séparé.

## Infisical

Infisical n'est plus le backend actif de Vixens. Les mentions de `InfisicalSecret`, Universal Auth ou de l'ancien endpoint dans les ADR, audits ou rapports datés sont **historiques**.

Ne jamais copier ces exemples dans un nouveau manifest.
