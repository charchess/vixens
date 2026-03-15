# Registre de la Dette Technique (Technical Debt Registry) 📒

Ce document répertorie les dérives architecturales, les exceptions de configuration et les compromis temporaires effectués pour maintenir la stabilité opérationnelle du cluster. Chaque entrée doit être associée à une tâche **Beads** pour son suivi et sa résolution.

## 🔴 Dette Critique (Mise en conformité requise)

| ID Tâche | Date | Composant | Description de la dérive | Impact |
| :--- | :--- | :--- | :--- | :--- |
| `vixens-pgk8` | 2026-03-14 | `booklore`, `lazylibrarian` | **Mode Scout** : Ressources CPU/RAM définies en dur pour forcer le scheduling sur un cluster saturé. | Non-conformité aux politiques Kyverno. |
| `vixens-qkyq` | 2026-03-14 | `frigate` | Ressources en dur pour les sidecars/initContainers de restauration (RAM migration). | Violation du principe DRY (Kyverno managed sizing). |

## 🟡 Dette Modérée (Optimisation requise)

| ID Tâche | Date | Composant | Description de la dérive | Impact |
| :--- | :--- | :--- | :--- | :--- |
| `vixens-mg4n` | 2026-03-14 | `Kyverno Policies` | Erreurs JMESPath dans les règles complexes (Audit mode). | Logs pollués, audit de maturité partiellement faussé. |
| `vixens-pf76` | 2026-03-14 | `Sizing Grid` | Grille de profil `B-*` (Burstable) avec des Requests trop hauts pour la capacité actuelle. | Famine de ressources (Pods en Pending). |

## 🟢 Dette Légère (Maintenance différée)

| ID Tâche | Date | Composant | Description de la dérive | Impact |
| :--- | :--- | :--- | :--- | :--- |
| `vixens-avrb` | 2026-03-14 | `Networking` | Gestion manuelle des NetworkPolicies pour les flux inter-apps (Whisparr -> Sabnzbd). | Manque de scalabilité, violation DRY. |

---
*Dernière mise à jour : 2026-03-14*
