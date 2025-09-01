# Vixens - Infrastructure GitOps

## Description
Infrastructure Kubernetes homelab gérée via ArgoCD avec pattern GitOps.

## Architecture
- **OS**: Talos Linux v1.10.7
- **Kubernetes**: v1.30.0
- **GitOps**: ArgoCD (App-of-Apps pattern)

## Démarrage rapide
```bash
# Installation
kubectl apply -f clusters/vixens/root-app.yaml