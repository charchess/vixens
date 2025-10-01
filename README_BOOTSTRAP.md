vixens bootstrap auto-generated
--------------------------------

Structure:
- bootstrap/{dev,stg,prod}/ : kustomize + Application argocd-self + apps-root
- bootstrap/argocd-values.yaml : used by run-bootstrap.sh for helm template values
- bootstrap/run-bootstrap.sh : render ArgoCD chart and apply core manifests then bootstrap
- apps/argocd/ : App-of-Apps entries (base + overlays)
- apps/metallb/ : example application (base + overlays per env)

Git workflow:
- All changes are pushed to branch 'dev'.
- Branches 'stg' and 'prod' are created on remote (from dev) and are targets for PRs.
  To promote to staging/prod: open PR dev -> stg (or dev -> prod), review, merge.

Bootstrap usage:
1) From your machine with kubectl/helm configured to the target cluster:
   bash bootstrap/run-bootstrap.sh dev

2) This will:
   - render ArgoCD manifests (helm template) and apply them (CRDs + controllers)
   - then apply kustomize in bootstrap/dev which creates Applications: argocd-self and apps-root
   - ArgoCD (self-managed) will then sync apps-root -> apps/argocd/overlays/dev

Security:
- Default config enables INSECURE HTTP and ANONYMOUS ADMIN (for bootstrap only).
- Do not expose ArgoCD publicly until you configure TLS & authentication & tighten RBAC.
