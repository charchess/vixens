# Exemption des Métriques (Gold Tier)

## Contexte

Selon l'**ADR-022**, toutes les applications au niveau **Gold** doivent exposer des métriques Prometheus (`prometheus.io/scrape`).

Cependant, certaines applications (comme `whoami`) sont des applications de test qui n'ont pas besoin de monitoring avancé. La question était : comment une application peut-elle atteindre Gold sans métriques ?

## Problème initial

L'approche initiale était circulaire :
- Si annotations Prometheus présentes → OK
- Si pas d'annotations → FAIL

Cela ne permettait pas d'exempter des applications légitimes tout en maintenant le standard Gold.

## Solution adoptée

### Annotation d'exemption

Nous avons introduit l'annotation **`vixens.io/nometrics: "true"`** qui permet à une application de déclarer explicitement qu'elle n'expose pas de métriques.

### Logique du script

```python
def check_metrics(pod):
    annotations = pod.get("metadata", {}).get("annotations", {})
    
    # Exemption explicite
    if annotations.get("vixens.io/nometrics") == "true":
        return None  # N/A - exempté
    
    # Check standard
    has_metrics = (
        "prometheus.io/scrape" in annotations
        or "prometheus.io/port" in annotations
    )
    
    return has_metrics  # True ou False
```

### Résultats possibles

| Annotation | `check_metrics()` | Gold Status |
|------------|-------------------|-------------|
| `prometheus.io/scrape: "true"` | `True` | ✅ Pass |
| `vixens.io/nometrics: "true"` | `None` | ✅ N/A (exempté) |
| Aucune des deux | `False` | ❌ ÉCHEC |

## Quand utiliser l'exemption ?

### ✅ Cas légitimes

- **Applications de test** (whoami, debug, etc.)
- **Apps sans interface réseau** (batch jobs, sidecars)
- **Apps temporaires** (one-shot)

### ❌ Cas interdits

- Applications de production
- Services exposés aux utilisateurs
- Apps avec SLA/SLO
- Bases de données, caches, brokers

## Exemple : whoami

### Avant (faux positif)

```yaml
annotations:
  prometheus.io/scrape: "true"  # Faux - whoami n'a pas de /metrics
  prometheus.io/port: "80"
```

Prometheus scrapait et recevait 404 ❌

### Après (honnete)

```yaml
annotations:
  vixens.io/nometrics: "true"  # Déclaration explicite
  goldilocks.fairwinds.com/enabled: "true"
  autoscaling.k8s.io/vpa: "true"
```

whoami passe Gold avec exemption valide ✅

## Usage

### Sans exemption (métriques requises)

```bash
$ python3 scripts/reports/evaluate_maturity.py myapp -n myns
Silver  # Échec Gold - pas de métriques

Missing Gold prerequisites:
  - Metrics exposed
```

### Avec exemption

```bash
$ python3 scripts/reports/evaluate_maturity.py whoami -n whoami
Gold  # ✅ Avec exemption N/A

Namespace: whoami
```

## Standard de qualité maintenu

L'annotation `vixens.io/nometrics` doit être **positionnée manuellement** et justifiée dans la review. Elle n'est pas automatique.

Cela garantit que :
1. Le standard Gold reste strict pour la majorité des apps
2. Les exceptions sont **explicites** et **tracées**
3. Pas de "faux positifs" avec des annotations Prometheus vides

## Références

- ADR-022 : 7-Tier Goldification System
- PR #1538 : Implementation de l'exemption
- whoami : Exemple de référence (`apps/99-test/whoami/base/deployment.yaml`)
