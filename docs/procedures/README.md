# Procedures

Procédures opérationnelles Vixens. Elles décrivent **comment opérer** le système ; le desired state reste dans Git et `WORKFLOW.md` définit le cycle de changement.

## Procédures disponibles

- **[Deployment Standard](deployment-standard.md)** — conventions de déploiement applicatif.
- **[Application Testing](application-testing.md)** — validation fonctionnelle après changement.
- **[Adding a New Talos Node](adding-new-talos-node.md)** — procédure Talos ; pour les changements d'infrastructure déclaratifs, respecter le repo Terraform/Talos faisant autorité.
- **[Dev Hibernation](dev-hibernation.md)** — opérations de mise en veille/réveil dev.
- **[Scout Mode Sizing](scout-mode-sizing.md)** — observation/tuning des ressources.

## Secrets

La rotation et le diagnostic des secrets suivent l'architecture courante :

```text
OpenBao → ClusterSecretStore/openbao → ExternalSecret → Secret → workload
```

Voir :

- `docs/guides/secret-management.md`
- `docs/adr/018-openbao-external-secrets-and-nas-fqdn.md`

Ne pas utiliser une ancienne procédure Infisical comme runbook actif.

## Procédure vs guide

- **Procedure** : action opérateur séquencée, avec prérequis et validation.
- **Guide** : explication/pattern pour construire ou modifier le système.
- **ADR** : décision durable et rationale.
- **Post-mortem/audit** : historique d'un état ou d'un incident ; pas une procédure courante.

## Maintenance

Lorsqu'une procédure change :

1. vérifier le `main` actuel et les PR concurrentes ;
2. mettre à jour la procédure dans la même PR que le contrat qu'elle décrit lorsque pertinent ;
3. éviter les commandes mutantes qui contournent GitOps pour un changement persistant ;
4. inclure une étape de validation observable ;
5. retirer les références aux outils/architectures retirés.

Pour une opération qui n'a pas encore de procédure durable, créer un fichier dédié plutôt que laisser un bloc « Coming Soon » ambigu dans cet index.

**Last Updated:** 2026-09-25
