# Guide : Mode Scout (V-scout) 🛰️

## Concept
Le mode **Scout** (`V-scout`) est un profil de dimensionnement temporaire utilisé pour les applications dont les besoins réels en ressources sont inconnus ou instables. Il permet de "faire entrer" une application sur un cluster saturé tout en lui donnant une marge de manœuvre quasi illimitée pour observer son comportement réel via le VPA ou Prometheus.

## Caractéristiques
*   **Request minimal (ex: 128Mi RAM) :** Pour garantir que l'ordonnanceur (Scheduler) accepte le pod, même sur des nœuds presque pleins.
*   **Limit maximal (ex: 8Gi RAM) :** Pour éviter tout crash de type `OOMKilled` pendant la phase d'analyse.
*   **Non-conformité assumée :** Le label `V-scout` n'est volontairement **pas inclus** dans les politiques Kyverno. Cela marque automatiquement le pod comme non conforme dans les audits de maturité, servant de rappel technique qu'un "vrai" profil (`G-`, `B-`, `SB-` ou `V-`) doit être défini plus tard.

## Utilisation
Pour activer le mode Scout sur un conteneur, modifiez le déploiement manuellement :

1.  Appliquez le label : `vixens.io/sizing.<container>: V-scout`
2.  Ajoutez manuellement le bloc `resources` (puisqu'aucune mutation automatique ne se fera) :
    ```yaml
    resources:
      requests:
        cpu: 10m
        memory: 128Mi
      limits:
        cpu: 1000m
        memory: 8Gi
    ```

## Sortie du mode Scout
Une fois que l'application a tourné pendant au moins 48h, consultez les recommandations Goldilocks/VPA. Supprimez ensuite le bloc `resources` manuel et remplacez le label `V-scout` par le profil T-Shirt standard correspondant à la consommation observée.
