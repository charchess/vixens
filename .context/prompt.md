# Contexte système

## Architecture C4
- C1: Système Talos + K8s + ArgoCD + Cilium + Traefik
- C2: 3 environnements (dev/staging/prod)
- En cours: reconstruction infrastructure

## Dernières commandes
$(cat .context/last-commands.txt 2>/dev/null || echo "Aucune")

## Étact terraform
$(terraform show -no-color 2>/dev/null | head -20 || echo "Pas d'état")

## À faire
$(cat docs/spec.md 2>/dev/null || echo "Spéc non définie")
