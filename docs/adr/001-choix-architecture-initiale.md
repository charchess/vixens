# ADR-001: Choix de l'Architecture Initiale


> **⚠️ HISTORICAL DOCUMENT - REQUIRES REVIEW**
>
> This ADR was restored from archived state (commit fe1e1cab, 2025-12-21) for historical purposes.
> The status and content need to be reviewed and updated to reflect current architecture.
> 
> Related task: vixens-0jt2

---

## Statut
✅ Accepté

## Contexte

Le projet Vixens vise à construire une infrastructure Kubernetes robuste et automatisée pour un homelab. Nous devons choisir une architecture de base qui sera la fondation de tous les développements futurs.

### Alternatives Évaluées

1. **Architecture Monolithique (VM unique)**
   - ✅ Simplicité de déploiement initial
   - ❌ Point de défaillance unique
   - ❌ Pas de scalabilité
   - ❌ Difficile à maintenir et à faire évoluer

2. **Architecture Multi-VMs (Kubernetes sur VMs)**
   - ✅ Haute disponibilité (HA)
   - ✅ Scalabilité horizontale
   - ✅ Flexibilité (choix de l'OS, CNI, etc.)
   - ❌ Complexité de gestion des VMs
   - ❌ Overhead de ressources (OS complet par VM)

3. **Architecture Bare-Metal (Kubernetes sur serveurs physiques)**
   - ✅ Performance maximale
   - ✅ Contrôle total sur le hardware
   - ❌ Coût élevé (achat/maintenance serveurs)
   - ❌ Moins flexible pour les tests et le développement

## Décision

**Adopter une architecture Multi-VMs avec Kubernetes sur des machines virtuelles.**

### Justifications

1. **Haute Disponibilité** : Permet de résister à la défaillance d'une VM ou d'un nœud physique sous-jacent.
2. **Scalabilité** : Facilite l'ajout ou la suppression de nœuds Kubernetes selon les besoins.
3. **Flexibilité** : Offre un bon équilibre entre contrôle et facilité de gestion, permettant d'expérimenter avec différentes configurations (CNI, stockage, etc.).
4. **Coût-efficacité** : Utilise les ressources existantes de l'hyperviseur de manière plus efficace qu'une solution bare-metal dédiée pour un homelab.

## Conséquences

### Positives
- ✅ Infrastructure résiliente et tolérante aux pannes.
- ✅ Possibilité de créer des environnements de développement et de test isolés.
- ✅ Facilite l'implémentation de pratiques GitOps et Infrastructure as Code.

### Négatives
- ⚠️ **Complexité de gestion** : Nécessite des outils pour gérer les VMs (Terraform, Ansible, etc.).
  - **Mitigation** : Utilisation de Terraform pour automatiser le provisionnement des VMs.
- ⚠️ **Overhead de ressources** : Chaque VM consomme des ressources (CPU, RAM, disque) même si elle est minimale.
  - **Mitigation** : Utilisation d'un OS minimaliste comme Talos Linux.

## Références

- [Kubernetes Documentation](https://kubernetes.io/docs/concepts/architecture/)
- [Talos Linux](https://www.talos.dev/)

---

**Date** : 2025-10-25
**Auteur** : Infrastructure Team
**Révisé** : N/A
