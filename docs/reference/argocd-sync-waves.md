# ArgoCD sync waves

Les sync waves ordonnancent les ressources qui ont une dépendance **réelle**. Elles ne doivent pas devenir une seconde orchestration globale du cluster.

## Principe

```yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "-2"
```

Une valeur plus basse est traitée avant une valeur plus haute. À wave égale, ArgoCD gère son ordre normal de synchronisation.

## Quand utiliser une wave

Une wave est justifiée lorsqu'un objet ne peut raisonnablement devenir sain avant un autre, par exemple :

```text
CRD
 ↓
operator/controller
 ↓
store/configuration gérée par ce controller
 ↓
workload consommateur
```

Exemples de dépendances :

- CRDs avant les Custom Resources qui les utilisent ;
- External Secrets Operator avant les `ExternalSecret` ;
- `ClusterSecretStore/openbao` avant les secrets applicatifs qui en dépendent ;
- cert-manager/webhook avant certains `Certificate`/issuers ;
- base de données partagée avant l'application lorsqu'une dépendance explicite est nécessaire.

Ne pas créer des waves uniquement pour "mettre de l'ordre" visuellement.

## Application vs ressource

Deux niveaux existent :

### Application ArgoCD

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: external-secrets
  annotations:
    argocd.argoproj.io/sync-wave: "-4"
```

Cela ordonne les Applications dans un app-of-apps.

### Ressource Kubernetes

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: openbao
  annotations:
    argocd.argoproj.io/sync-wave: "-3"
```

Cela ordonne les ressources au sein d'une synchronisation.

Choisir le niveau correspondant à la dépendance réelle ; ne pas dupliquer la même logique partout.

## Modèle secrets actuel

L'ancien ordre Infisical n'est plus applicable.

Le modèle actuel est :

```text
External Secrets Operator / CRDs
        ↓
ClusterSecretStore/openbao
        ↓
ExternalSecret
        ↓
Kubernetes Secret
        ↓
workload
```

Le nom exact des Applications et leurs waves courantes doivent être lus dans `argocd/` plutôt que copiés depuis ce document.

## Health et attente

Les waves ne remplacent pas la health assessment. Une dépendance critique doit exposer un état que ArgoCD ou son controller peut observer.

Éviter :

- `sleep` arbitraires ;
- Jobs uniquement destinés à ralentir une sync ;
- dépendances implicites non documentées ;
- mutations `kubectl` post-sync qui rendent Git incomplet.

## Validation

Après modification des waves :

1. construire/rendre les overlays concernés ;
2. vérifier que les annotations sont réellement présentes dans les objets rendus ;
3. tester d'abord sur dev ;
4. observer l'ordre et la santé dans ArgoCD ;
5. vérifier le comportement des dépendances, pas seulement `Synced` ;
6. documenter une dépendance inhabituelle près de la ressource concernée ou dans un ADR.

## Historique

Les vieux rapports d'incident peuvent mentionner l'Infisical Operator, Synology CSI ou d'anciennes waves. Ils décrivent l'état du cluster à leur date et ne constituent pas le plan de waves courant.
