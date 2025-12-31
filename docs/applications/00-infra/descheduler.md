# Descheduler

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 0.32.2  |
| Prod          | [ ]     | [ ]       | [ ]   | 0.32.2  |

## Politique de Rééquilibrage
Le Descheduler est configuré en mode **CronJob** pour s'exécuter toutes les **4 heures**. La politique est volontairement conservative pour éviter des mouvements incessants de pods.

### Stratégies activées :
1.  **RemoveDuplicates** : S'assure qu'un seul pod d'un même ReplicaSet s'exécute sur un nœud.
2.  **RemovePodsViolatingInterPodAntiAffinity** : Respecte les règles d'anti-affinité.
3.  **RemovePodsViolatingNodeAffinity** : Respecte les règles d'affinité des nœuds.
4.  **HighNodeUtilization** : Expulse les pods des nœuds dont l'utilisation (CPU/Mémoire) dépasse les seuils définis pour les redistribuer sur des nœuds moins chargés.
    *   Seuils bas : 20%
    *   Seuils cibles : 50%

## Validation
### Méthode Automatique (Logs)
```bash
# Vérifier l'exécution du CronJob
kubectl -n descheduler get jobs
# Vérifier les logs du dernier job
kubectl -n descheduler logs -l job-name=<job-name>
# Attendu: Liste des pods expulsés ou "No pods to evict"
```

## Notes Techniques
- **Namespace :** `descheduler`
- **Fréquence :** `0 */4 * * *`
- **Tolérations :** Control-plane (pour analyser tous les nœuds).
- **Impact :** Les applications avec `strategy: Recreate` subiront une brève interruption (10-30s) en cas d'expulsion.
