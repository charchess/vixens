# Issue: Cilium DNS "operation not permitted" after destroy/recreate

**Date**: 2025-11-14
**Environment**: dev
**Severity**: High
**Status**: Open - Requires investigation

## Summary

After a complete `terraform destroy` followed by `terraform apply` of the dev environment, ArgoCD applications fail to sync with DNS resolution errors showing "operation not permitted" when trying to contact CoreDNS (kube-dns).

## Symptoms

### Primary Error

ArgoCD repo-server cannot resolve DNS queries:

```
connection error: desc = "transport: Error while dialing: dial tcp: lookup argocd-repo-server on 10.96.0.10:53: dial udp 10.96.0.10:53: connect: operation not permitted"
```

### Affected Components

1. **ArgoCD Applications**:
   - `cert-manager-webhook-gandi`: OutOfSync/Degraded
   - Unable to generate manifests from Helm charts

2. **cert-manager webhook resources**:
   - Certificate `cert-manager-webhook-gandi-ca`: SyncFailed/Missing
   - Certificate `cert-manager-webhook-gandi-webhook-tls`: SyncFailed/Missing
   - Issuer `cert-manager-webhook-gandi-ca`: SyncFailed/Missing
   - Issuer `cert-manager-webhook-gandi-selfsign`: SyncFailed/Missing

3. **cert-manager-webhook-gandi pod**:
   - Status: `ContainerCreating` (stuck for 30+ minutes)
   - Error: `MountVolume.SetUp failed for volume "certs" : secret "cert-manager-webhook-gandi-webhook-tls" not found`
   - Cannot start because TLS certificate secret doesn't exist
   - TLS certificate cannot be created because ArgoCD cannot sync the Certificate resources

### Cascade Effect

```
ArgoCD DNS blocked
  ↓
Cannot generate Helm manifests
  ↓
Cannot sync Certificate resources
  ↓
cert-manager cannot create TLS certificate
  ↓
webhook-gandi pod cannot mount TLS secret
  ↓
webhook-gandi pod stuck in ContainerCreating
```

## Environment Details

### Cluster Configuration

- **Environment**: dev
- **Nodes**: obsy (192.168.111.162), onyx (192.168.111.164), opale (192.168.111.163)
- **Kubernetes**: v1.34.0
- **Talos**: v1.11.0
- **CNI**: Cilium v1.18.3
- **VIP**: 192.168.111.160
- **VLANs**: 111 (internal), 208 (services)

### Cilium Configuration

```yaml
enable-policy: default
enable-k8s-networkpolicy: true
bpf-policy-map-max: 16384
enable-endpoint-lockdown-on-policy-overflow: false
```

### Network Policies

- **Count**: 0 (no NetworkPolicy resources in cluster)
- **Cilium enforcement**: Enabled by default

### CoreDNS Service

- **IP**: 10.96.0.10
- **Port**: 53 (UDP/TCP)
- **Namespace**: kube-system

## What Works vs. What Doesn't

### ✅ Working

1. **Test environment**: Same configuration, deployed weeks ago, works perfectly
2. **cert-manager base**: Deployed, Running, Healthy
3. **cert-manager-webhook**: Deployed, Running (after restart)
4. **ArgoCD applications sync status**: Most apps are Synced
5. **Cilium connectivity**: Basic pod-to-pod communication works
6. **DNS resolution from regular pods**: Works (tested with busybox)

### ❌ Not Working

1. **ArgoCD to CoreDNS**: DNS lookups fail with "operation not permitted"
2. **cert-manager-webhook to CoreDNS**: Certificate validation fails with same error
3. **UDP traffic to 10.96.0.10:53**: Blocked by Cilium/kernel

## Investigation Steps Performed

### 1. Verified cert-manager pods

```bash
kubectl get pods -n cert-manager
```

Result:
- `cert-manager`: Running ✅
- `cert-manager-webhook`: Running ✅ (after restart)
- `cert-manager-webhook-gandi`: ContainerCreating ❌ (30+ minutes)

### 2. Checked ArgoCD application status

```bash
kubectl get applications -n argocd
```

Result:
- Most apps: Synced/Healthy ✅
- `cert-manager-webhook-gandi`: OutOfSync/Degraded ❌

### 3. Examined application sync error

```bash
kubectl get application cert-manager-webhook-gandi -n argocd -o yaml
```

Found DNS resolution error with "operation not permitted"

### 4. Verified Cilium configuration

```bash
kubectl get cm -n kube-system cilium-config -o yaml
```

Result: Policy enforcement enabled, no obvious misconfigurations

### 5. Checked for NetworkPolicies

```bash
kubectl get networkpolicies -A
```

Result: No NetworkPolicy resources exist

### 6. Restarted cert-manager-webhook

```bash
kubectl delete pod -n cert-manager -l app.kubernetes.io/name=webhook
```

Result: New pod started Running, but error persists

## Root Cause Hypothesis

### Primary Hypothesis: Cilium eBPF Policy Enforcement

When Cilium is deployed fresh on a destroy/recreate cluster, it may be enforcing stricter default policies that block UDP traffic to CoreDNS (10.96.0.10:53) from certain namespaces or pods.

**Evidence**:
1. Error message: "operation not permitted" is typical of eBPF policy denial
2. Same configuration works in test environment (deployed incrementally, not destroyed)
3. No NetworkPolicy resources, so blocking comes from Cilium defaults
4. DNS queries specifically fail (UDP port 53)

### Secondary Hypothesis: Timing/Race Condition

During initial deployment, ArgoCD and cert-manager may attempt to sync before Cilium has fully initialized its datapath and policy rules.

**Evidence**:
1. Problem appears immediately after fresh deployment
2. Test environment (never destroyed) doesn't have this issue
3. Cilium policies take time to propagate

### Tertiary Hypothesis: Host Firewall / iptables

Talos Linux may have host-level firewall rules that interfere with Cilium's operation on fresh deployments.

**Evidence**:
1. Talos is immutable and may have different initial state after recreate
2. "operation not permitted" can come from kernel-level filtering
3. Cilium kube-proxy replacement may interact with host firewall

## Not Related To

❌ **ArgoCD DRY Optimization PR #88**: This is NOT a regression from the Phase 1 migration
- Same app manifests in test (working) and dev (not working)
- PR only reorganized file structure (moved apps to `apps/` subdirectory)
- No changes to app configuration, sync waves, or behavior
- Test environment uses old structure, works fine
- Dev environment uses new structure, fails only after destroy/recreate

## Recommended Next Steps

### Immediate (Workaround)

1. **Restart Cilium agents** to force policy recomputation:
   ```bash
   kubectl rollout restart ds/cilium -n kube-system
   ```

2. **Create explicit NetworkPolicy** allowing DNS traffic:
   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: allow-dns
     namespace: argocd
   spec:
     podSelector: {}
     policyTypes:
     - Egress
     egress:
     - to:
       - namespaceSelector:
           matchLabels:
             kubernetes.io/metadata.name: kube-system
       ports:
       - protocol: UDP
         port: 53
       - protocol: TCP
         port: 53
   ```

3. **Temporarily disable Cilium policy enforcement**:
   ```bash
   kubectl patch cm cilium-config -n kube-system \
     --type merge \
     -p '{"data":{"enable-policy":"never"}}'
   kubectl rollout restart ds/cilium -n kube-system
   ```

### Medium Term (Investigation)

1. **Compare Cilium state** between test (working) and dev (broken):
   ```bash
   # Test env
   cilium status --wait
   cilium policy get
   cilium endpoint list

   # Dev env (same commands)
   ```

2. **Enable Cilium debug logging**:
   ```bash
   kubectl patch cm cilium-config -n kube-system \
     --type merge \
     -p '{"data":{"debug":"true"}}'
   ```

3. **Capture Cilium monitor** during ArgoCD sync attempt:
   ```bash
   cilium monitor --type drop --type policy-verdict
   ```

4. **Check Hubble for flow visibility**:
   ```bash
   hubble observe --namespace argocd --protocol UDP --port 53
   ```

5. **Review Cilium eBPF maps**:
   ```bash
   cilium bpf policy get <endpoint-id>
   ```

6. **Compare Terraform state** between fresh deploy and working cluster:
   - Cilium Helm values
   - kube-proxy replacement settings
   - Host firewall rules in Talos config

### Long Term (Fix)

1. **Add Cilium DNS egress policy** to Terraform/ArgoCD base configuration
2. **Document required NetworkPolicies** for all core services
3. **Add validation script** to check DNS connectivity after deployment
4. **Investigate Talos + Cilium** host firewall interaction
5. **Consider Cilium policy pre-population** before ArgoCD sync

## References

### Log Snippets

**ArgoCD repo-server error**:
```
Failed to load target state: failed to generate manifest for source 1 of 1:
rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing:
dial tcp: lookup argocd-repo-server on 10.96.0.10:53: dial udp 10.96.0.10:53: connect: operation not permitted"
```

**cert-manager-webhook-gandi pod**:
```
Events:
  Type     Reason       Age                  From               Message
  ----     ------       ----                 ----               -------
  Warning  FailedMount  103s (x18 over 22m)  kubelet            MountVolume.SetUp failed for volume "certs" :
  secret "cert-manager-webhook-gandi-webhook-tls" not found
```

**cert-manager-webhook TLS errors**:
```
http: TLS handshake error from 10.244.0.187:41020: remote error: tls: bad certificate
http: TLS handshake error from 10.244.0.187:41030: remote error: tls: bad certificate
```

### Related Issues

- Cilium GitHub: https://github.com/cilium/cilium/issues?q=is%3Aissue+dns+operation+not+permitted
- cert-manager webhook bootstrap: https://cert-manager.io/docs/concepts/webhook/
- ArgoCD DNS resolution: https://argo-cd.readthedocs.io/en/stable/operator-manual/tls/

### Cluster Context

- **Last working state**: Test environment (never destroyed)
- **Breaking change**: `terraform destroy && terraform apply` on dev
- **Terraform modules**: Using shared module for Cilium v1.18.3 deployment
- **ArgoCD sync waves**: cert-manager (0), webhook-gandi (1), cert-manager-config (2)

## Timeline

- **23:22**: Fresh terraform apply completed
- **23:22**: cert-manager deployed (wave 0)
- **23:23**: cert-manager-webhook-gandi deployed (wave 1)
- **23:23**: DNS errors start appearing in ArgoCD
- **23:23**: Certificate resources fail to sync with "operation not permitted"
- **23:24**: cert-manager-webhook-gandi pod stuck in ContainerCreating
- **23:43**: Investigation started
- **23:48**: cert-manager-webhook restarted (now Running)
- **23:48**: Problem persists - ArgoCD still cannot resolve DNS

## Impact

### Current Impact

- ✅ **ArgoCD self-management**: Working
- ✅ **Most applications**: Syncing correctly
- ❌ **cert-manager-webhook-gandi**: Cannot deploy
- ❌ **Let's Encrypt DNS-01 challenges**: Not functional
- ❌ **TLS certificate automation**: Broken for Gandi domains

### Future Risk

If this issue is not resolved:
1. Any destroy/recreate of dev will reproduce the problem
2. Test/staging/prod environments may face same issue if recreated
3. Fresh cluster deployments will fail
4. Automation/GitOps workflows will be unreliable

## Owner

- **Team**: Infrastructure
- **Contact**: See vixens project README

## Related Documents

- `/docs/adr/004-cilium-cni.md` - Cilium CNI selection
- `/docs/architecture/network-diagram.md` - Network architecture
- `/terraform/modules/cilium/` - Cilium Terraform module
- `/argocd/overlays/dev/apps/cert-manager-webhook-gandi.yaml` - App manifest

---

**Note**: This issue was discovered during validation of PR #88 (ArgoCD DRY Phase 1) but is NOT caused by that PR. It is an infrastructure issue that appears on fresh cluster deployments.

Note importante : il est psosible que ca soit jsute du a une lenteur de mise en place, l'ensemble semble fonctionner apres une heure sans intervention notable.