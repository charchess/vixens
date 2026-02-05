# Secret Management Guide (Infisical)

This guide covers secrets management, synchronization, and access within the Vixens infrastructure using **Infisical**.

---

## 🏗️ Architecture

1.  **Infisical Server:** Self-hosted at `http://192.168.111.69:8085`.
2.  **Infisical Operator:** Runs in the cluster and synchronizes secrets from Infisical to Kubernetes `Secret` resources.
3.  **Machine Identity:** Uses **Universal Auth** for secure, non-interactive access.

---

## 💻 CLI Access (Critical for Automation)

When using the Infisical CLI with Machine Identity (Universal Auth), there is a known issue where the project slug (`vixens`) might not resolve correctly (returning 404).

**Standard Rule:** Always use the **Project ID (UUID)** instead of the slug.

### 🔑 Credentials
- **Project ID:** `47aca60e-543b-4fd6-b646-8ebd5a7b3433`
- **Machine Identity Secret:** Stored in Kubernetes under `argocd/infisical-universal-auth`.

### 🔓 Login Procedure
```bash
infisical login --method=universal-auth \
  --client-id=<YOUR_CLIENT_ID> \
  --client-secret=<YOUR_CLIENT_SECRET> \
  --domain http://192.168.111.69:8085
```

### 📦 Fetching Secrets
```bash
# Use --projectId instead of --project
infisical secrets --projectId 47aca60e-543b-4fd6-b646-8ebd5a7b3433 --env prod --path /apps/00-infra/velero
```

---

## 🔄 Synchronization Protocol

### 1. In Infisical
- Add your secret in the web UI.
- Use paths like `/apps/<category>/<app-name>`.

### 2. In Kubernetes (GitOps)
Create an `InfisicalSecret` resource:

```yaml
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: my-app-secrets
spec:
  hostAPI: http://192.168.111.69:8085
  authentication:
    universalAuth:
      credentialsRef:
        secretName: infisical-universal-auth
        secretNamespace: argocd
      secretsScope:
        projectSlug: vixens
        envSlug: prod
        secretsPath: /apps/my-category/my-app
  managedSecretReference:
    secretName: my-app-secrets
    creationPolicy: Owner
```

---

## 🛠️ Troubleshooting

### Error: "404 Project Not Found"
- **Cause:** Using slug `vixens` with Universal Auth in CLI.
- **Fix:** Use the UUID `47aca60e-543b-4fd6-b646-8ebd5a7b3433`.

### Secrets not updating
- Check operator logs: `kubectl logs -n infisical-operator-system -l app.kubernetes.io/name=secrets-operator`.
- Force sync via annotation: `kubectl annotate infisicalsecret <name> secrets.infisical.com/force-sync=$(date +%s) --overwrite`.

---

**Last Updated:** 2026-02-05 (Fix for CLI 404 issue)
