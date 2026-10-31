# TXO Fabric — finalize a durable RWO active volume (#4384)

The immutable `spec.runtime.storage.activeVolume` pin is the permanent
steady-state identity contract from [ADR-044](../adr/044-fabric-durable-active-volume-bindings.md).
It is not a migration, copy proof, PVC creation request, or operator-generated
default. No tenant is automatically opted in.

## Evidence before declaring a pin

Read each **actual** tenant namespace and agent from the cluster, not from old
manifests or migration logs:

```bash
kubectl -n tenant-<tenant> get pvc <active-claim> -o yaml
kubectl get pv <pvc.spec.volumeName> -o yaml
kubectl get agentidentity <agent> -o yaml
```

Record the live PVC **metadata.uid**, bound PV name and **claimRef.uid**, class,
access mode, phase, Fabric tenant/agent labels, retention label, actual PV
reclaim policy and current Deployment claim. Confirm the volume is the approved
existing data volume; `Bound` or a historical source-UID label does **not**
prove copied data. In particular, a prepare-only agent must not be pinned to
a replacement lacking genuine copy verification.

Once independently verified, declare the following *actual observed values*:

```yaml
spec:
  runtime:
    storage:
      retentionPolicy: Retain
      activeVolume:
        claimName: <actual-pvc-name>
        pvcUid: <actual-pvc-uid>
        volumeName: <actual-pv-name>
        storageClassName: <actual-storage-class>
        accessMode: ReadWriteOnce
        retentionPolicy: Retain
```

No placeholder is a valid value. `Delete` is allowed only when the agent's
existing Delete policy, its PVC retention label, and the actual PV reclaim
policy all agree. This binding must be delivered as a reviewed GitOps change
after the physical identity baseline is recorded in #4357.

## Reconciliation and recovery

With an active binding set, the agent controller checks the *same* PVC UID,
PV name and UID-bound claimRef, Bound phases, class, mode, owner/tenant labels
and retention before changing the Deployment. It does **not** create or
reconcile legacy source or target migration PVCs, and it does **not** read
transient Jobs/ConfigMaps/legacy attestations. Historical audit status is
not deleted. A missing or replaced claim causes a fail-closed degraded state.

The pin is immutable once set, including removal. Recovery from a missing
or replaced PV/PVC requires a separately authorized restore/new migration
and a reviewed change to the binding contract. Reverting the operator code
alone cannot restore a lost physical volume, and reverting a pinned spec
may require a separately reviewed CRD transition strategy.

Existing unpinned agents continue on the prior migration path pending verified
RWO inventory and rollout. hAIrem is not a special case.

After code review and CI, validate from dev GitOps revision through operator
image pin and two reconciles to real runtime PVC/PV UID and Deployment mounts.
No physical check was performed by the source change alone.
