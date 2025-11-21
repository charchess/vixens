# Enforce HTTPS Redirect on All Ingress Resources

## Why

**Current State:**
- Some ingress resources have HTTP → HTTPS redirect (argocd, whoami via separate ingress)
- Others have TLS but no automatic redirect (homeassistant, mail-gateway)
- Inconsistent approach: some use separate redirect ingress, others rely on browser behavior

**Problems:**
- ❌ **Insecure HTTP access**: Users can access apps via http:// (unencrypted)
- ❌ **No consistency**: Different redirect patterns across apps
- ❌ **Manual configuration**: Each app must implement redirect logic

**Vision:**
All ingress resources SHALL automatically redirect HTTP → HTTPS using Traefik middleware, providing consistent security across all environments and applications.

## What Changes

### Traefik Middleware Approach

Use Traefik's built-in redirect middleware via ingress annotations:

```yaml
annotations:
  traefik.ingress.kubernetes.io/router.middlewares: default-redirect-https@kubernetescrd
```

### Centralized Middleware

Create a global middleware in Traefik namespace:

```yaml
# apps/traefik/base/middleware-redirect-https.yaml
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: redirect-https
  namespace: traefik
spec:
  redirectScheme:
    scheme: https
    permanent: true  # 301 redirect
```

### Application Integration

**Option 1: Annotation on every ingress (CHOSEN)**
- Add annotation to each ingress resource in base/
- Simple, explicit, easy to audit

**Option 2: IngressClass default**
- Configure Traefik IngressClass with default middlewares
- Implicit, but requires Traefik configuration changes

We choose **Option 1** for clarity and control.

### Example Changes

**Before:**
```yaml
# apps/homeassistant/base/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: homeassistant-ingress
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-staging
spec:
  ingressClassName: traefik
  rules:
    - host: homeassistant.dev.truxonline.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: homeassistant-svc
                port:
                  number: 8123
  tls:
    - hosts:
        - homeassistant.dev.truxonline.com
      secretName: homeassistant-tls-dev
```

**After:**
```yaml
# apps/homeassistant/base/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: homeassistant-ingress
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-staging
    traefik.ingress.kubernetes.io/router.middlewares: traefik-redirect-https@kubernetescrd
spec:
  ingressClassName: traefik
  rules:
    - host: homeassistant.dev.truxonline.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: homeassistant-svc
                port:
                  number: 8123
  tls:
    - hosts:
        - homeassistant.dev.truxonline.com
      secretName: homeassistant-tls-dev
```

### Affected Applications

- ✅ homeassistant (add annotation)
- ✅ mail-gateway (add annotation)
- ⚠️ argocd (already has redirect ingress - consolidate)
- ⚠️ whoami (already has redirect ingress - consolidate)
- ✅ traefik-dashboard (add annotation)

## Non-Goals

- **Not blocking HTTP entirely** - Still allow HTTP for health checks, redirects
- **Not using cert-manager HTTP-01** - Keep DNS-01 challenge (no change)
- **Not adding HSTS headers** - Can be added later as separate change

## Testing Strategy

### Phase 1: Deploy Middleware
1. Create `apps/traefik/base/middleware-redirect-https.yaml`
2. Apply to dev environment via ArgoCD
3. Verify middleware exists: `kubectl get middleware -n traefik`

### Phase 2: Test with One Application (homeassistant)
1. Add annotation to `apps/homeassistant/base/ingress.yaml`
2. Commit and push to dev
3. Wait for ArgoCD sync
4. Test redirect: `curl -I http://homeassistant.dev.truxonline.com`
   - Should return `301 Moved Permanently`
   - Location header should be `https://homeassistant.dev.truxonline.com`
5. Test HTTPS still works: `curl -I https://homeassistant.dev.truxonline.com`

### Phase 3: Rollout to All Applications
1. Add annotation to remaining ingress resources
2. Remove separate redirect ingress resources (argocd, whoami)
3. Commit and push
4. Test all applications for redirect functionality

### Phase 4: Multi-Environment Validation
1. Test in test environment
2. Test in staging environment
3. Test in prod environment

## Success Criteria

- ✅ Traefik Middleware `redirect-https` deployed in traefik namespace
- ✅ All ingress resources have annotation `traefik.ingress.kubernetes.io/router.middlewares`
- ✅ HTTP requests to all apps return 301 redirect to HTTPS
- ✅ HTTPS access remains functional for all apps
- ✅ No separate HTTP redirect ingress resources (cleanup argocd, whoami)
- ✅ Consistent behavior across all 4 environments

## Rollback Plan

1. **Remove annotations** from ingress resources
2. **Delete middleware**: `kubectl delete middleware -n traefik redirect-https`
3. **Restore separate redirect ingresses** for apps that had them (git revert)
4. ArgoCD will automatically resync to previous state
