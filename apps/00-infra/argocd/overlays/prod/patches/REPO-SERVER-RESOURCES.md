# ArgoCD Repository Server Resources

## Problem
ArgoCD repo-server crashes with `CrashLoopBackOff` due to insufficient resources.

Kyverno policies enforce "micro" sizing (128Mi memory) which is insufficient for ArgoCD repo-server.

## Solution Applied (Manual)
Direct patch applied to cluster:
```bash
kubectl patch deployment argocd-repo-server -n argocd --patch-file repo-server-resources.yaml
```

## Required Resources
```yaml
spec:
  template:
    metadata:
      labels:
        vixens.io/sizing: "large"
        vixens.io/priority: "critical"
    spec:
      containers:
        - name: repo-server
          resources:
            limits:
              cpu: "2"
              memory: 2Gi
            requests:
              cpu: 250m
              memory: 512Mi
```

## Long-term Fix
Update Terraform Helm release values for ArgoCD to include these resources.
