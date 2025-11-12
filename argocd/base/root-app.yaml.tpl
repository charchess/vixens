# App-of-Apps Root Application
# This Application manages all other Applications in the cluster
# It watches argocd/overlays/${environment}/ and deploys all Applications defined there
#
# This file is a Terraform template - DO NOT apply manually!
# Rendered by: terraform/environments/${environment}/argocd.tf
#
# Variables:
#   - environment: ${environment}
#   - target_revision: ${target_revision}
#   - overlay_path: ${overlay_path}

apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: vixens-app-of-apps
  namespace: argocd
  labels:
    vixens.lab/environment: ${environment}
    vixens.lab/managed-by: terraform
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: https://github.com/charchess/vixens.git
    targetRevision: ${target_revision}
    path: ${overlay_path}
  destination:
    server: https://kubernetes.default.svc
    namespace: argocd
  syncPolicy:
    automated:
      prune: true
      selfHeal: true