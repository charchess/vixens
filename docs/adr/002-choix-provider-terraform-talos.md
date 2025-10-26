# 2. Choix du Provider Terraform pour Talos

- **Statut**: Accepté
- **Date**: 2025-10-26

## Contexte

Pour configurer et amorcer un cluster Talos de manière programmatique et reproductible, nous avons besoin d'un provider Terraform capable d'interagir avec l'API de Talos en mode maintenance.

## Décision

Nous choisissons d'utiliser le provider officiel `siderolabs/talos`.

Nous adoptons une stratégie de versioning souple mais sécurisée en spécifiant la version `~> 0.9.0` dans notre fichier `versions.tf`. Cela nous permet de bénéficier des corrections de bugs (versions `0.9.x`) tout en nous protégeant contre des changements majeurs potentiellement déstabilisants (versions `0.10.0` ou `1.0.0`).

## Conséquences

- Toute la logique de configuration de Talos sera gérée via les ressources de ce provider.
- Nous devons maintenir ce provider à jour au fil du temps pour bénéficier des nouvelles fonctionnalités et des correctifs de sécurité.
- La dépendance à ce provider est centralisée dans le fichier `versions.tf` du module, conformément à nos conventions.
