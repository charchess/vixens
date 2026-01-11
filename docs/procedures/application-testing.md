# Application Testing Process

## Overview

This document describes the process for testing and evaluating new applications before deploying them to production.

## Testing Workflow

### Phase 1: Multi-Application Comparison

When evaluating multiple similar applications (e.g., comparing different note-taking apps, media servers, or monitoring tools), follow this process:

1. **Deploy to `apps/99-test/`**
   - Create minimal manifests for each candidate application
   - Use simple configurations (no complex PVC, no databases)
   - Use temporary storage if needed (emptyDir or simple PVC)
   - Each app gets its own subdirectory in `apps/99-test/`

2. **Configure Basic Access**
   - Set up ingress with test subdomain: `<app-name>-test.dev.truxonline.com`
   - Use Let's Encrypt staging certificates
   - No authentication required for initial testing

3. **Deploy to Dev Cluster**
   ```bash
   # Applications in apps/99-test/ will be auto-synced by ArgoCD
   git add apps/99-test/<app-name>
   git commit -m "test: add <app-name> for evaluation"
   git push
   ```

4. **User Evaluation**
   - User tests each application via WebUI
   - User evaluates features, UI/UX, performance
   - User decides which application to keep

5. **Cleanup Test Applications**
   ```bash
   # Remove unused applications from test directory
   git rm -r apps/99-test/<rejected-app>
   git commit -m "test: remove rejected <app-name>"
   git push
   ```

### Phase 2: Proper Deployment

Once the user has selected an application:

1. **Clean Deployment to Dev**
   - Move application from `apps/99-test/` to proper category (e.g., `apps/60-services/`)
   - Create complete manifests with proper configuration
   - Configure persistent storage if needed
   - Set up database if required
   - Configure monitoring (ServiceMonitor)
   - Set up proper authentication

2. **Documentation**
   - Create application documentation in `docs/applications/<category>/<app-name>.md`
   - Document configuration choices
   - Document access URLs
   - Document backup requirements (if applicable)

3. **Validation in Dev**
   - Test functionality thoroughly
   - Verify monitoring is working
   - Test backup/restore (if applicable)
   - Validate resource usage

4. **Promotion to Production**
   - Follow standard GitOps promotion workflow
   - Update production overlay with prod-specific config
   - Tag with `prod-stable` tag
   - Validate in production

---

## Test Directory Structure

```
apps/99-test/
├── <app-name-1>/
│   ├── base/
│   │   ├── kustomization.yaml
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   └── overlays/
│       └── dev/
│           ├── kustomization.yaml
│           └── ingress.yaml
│
├── <app-name-2>/
│   └── ...
│
└── <app-name-3>/
    └── ...
```

---

## Test Configuration Guidelines

### Minimal Deployment

For testing purposes, use minimal configuration:

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: app-test
  template:
    metadata:
      labels:
        app: app-test
    spec:
      containers:
      - name: app
        image: app-image:latest
        ports:
        - containerPort: 8080
        resources:
          requests:
            cpu: 50m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
        # No health checks for testing
        # No persistent storage initially
```

### Temporary Storage

If the app requires storage for testing:

```yaml
# Use emptyDir (data lost on restart - OK for testing)
volumes:
- name: data
  emptyDir: {}

# Or simple PVC (will be deleted after testing)
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: app-test-data
spec:
  accessModes:
  - ReadWriteOnce
  storageClassName: synology-iscsi-retain  # or synology-iscsi-delete
  resources:
    requests:
      storage: 1Gi
```

---

## When to Use Test Directory

**Use `apps/99-test/` for:**
- Comparing multiple similar applications
- Quick proof-of-concept deployments
- Evaluating new applications before commitment
- Testing application compatibility with infrastructure

**Do NOT use for:**
- Long-term deployments
- Applications already decided and ready for production
- Production services (even in dev environment)

---

## Example: Testing Note-Taking Apps

```bash
# Deploy 3 note-taking apps for comparison
mkdir -p apps/99-test/{joplin,trilium,obsidian}

# Create minimal manifests for each
# ... (deployment, service, ingress)

git add apps/99-test/
git commit -m "test: compare note-taking applications"
git push

# User tests each:
# - https://joplin-test.dev.truxonline.com
# - https://trilium-test.dev.truxonline.com
# - https://obsidian-test.dev.truxonline.com

# User selects Trilium, cleanup others:
git rm -r apps/99-test/{joplin,obsidian}
git commit -m "test: remove rejected note-taking apps"

# Proper deployment of Trilium:
mkdir -p apps/60-services/trilium
# ... create production-grade manifests
git add apps/60-services/trilium
git commit -m "feat(trilium): deploy note-taking application"
git push
```

---

## Cleanup Checklist

After testing is complete:

- [ ] Remove test applications from `apps/99-test/`
- [ ] Delete test PVCs (if any)
- [ ] Remove test DNS records (if custom)
- [ ] Clean up test secrets (if any)
- [ ] Verify ArgoCD has removed test applications

---

## Related Documentation

- [Adding New Applications](../guides/adding-new-application.md)
- [GitOps Workflow](../guides/gitops-workflow.md)
- [Quality Standards](../reference/quality-standards.md)

---

**Last Updated:** 2026-01-11
