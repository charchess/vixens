# CoreDNS

CoreDNS is the Kubernetes DNS control plane in `kube-system`. Terraform may create an UDM-only bootstrap Corefile before ArgoCD exists; after the documented handoff, this Vixens Application is the sole runtime owner of the ConfigMap.

## Runtime forwarding policy

- `internal.truxonline.com` → domain controllers `192.168.200.21` and `192.168.200.22`, sequentially.
- `truxonline.com` → UDM `192.168.201.1`.
- All other names → AdGuard Home at the declared `10.110.196.136` ClusterIP, then UDM `192.168.201.1` sequentially when AdGuard has no ready endpoint.

CoreDNS must not use the AdGuard Service FQDN as an upstream: `forward` accepts
an address or a file, and resolving that FQDN through CoreDNS would create a
bootstrap dependency. AdGuard remains the normal filtered resolver; the UDM
fallback preserves CoreDNS, ArgoCD and External Secrets recovery when the
recoverable AdGuard workload is unavailable.

AdGuard Home must be Ready before this Application is reconciled. The app-of-apps applies it at sync wave 6, after AdGuard Home at wave 5.

## Ownership handoff

The Terraform bootstrap resource must be removed from Terraform state **only after** this Application is `Synced/Healthy` and DNS has been verified. Disabling the Terraform resource before the state handoff would cause Terraform to delete the live ConfigMap.

## Verification

From a Pod using cluster DNS, verify all three resolution paths:

```text
nas.truxonline.com              # UDM zone
<directory-host>.internal.truxonline.com  # DC zone
example.com                     # AdGuard/general upstream
```

| Environment | Declared | Configured | Tested | Version |
|-------------|----------|------------|--------|---------|
| Prod | [ ] | [ ] | [ ] | pending GitOps promotion |
