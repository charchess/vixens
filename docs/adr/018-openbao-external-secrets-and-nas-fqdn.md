# ADR 018: OpenBao / External Secrets and NAS FQDN

**Date:** 2026-09-12
**Status:** Accepted
**Deciders:** Vixens maintainers
**Tags:** security, secrets, storage, dns, portability

---

## Decision

- Retire the deprecated **Infisical Operator** from the Argo CD desired state.
- Use **External Secrets Operator** with the UMI OpenBao `ClusterSecretStore` for managed secret material.
- Use `nas.truxonline.com` as the canonical endpoint for the current TrueNAS services (NFS, iSCSI templates, OpenBao, and MinIO/S3), rather than embedding a mutable IPv4 address in GitOps manifests.

## Rationale

The Infisical controller is deprecated in this platform and duplicates the OpenBao/External Secrets path already reconciled by Argo CD. Removing the controller avoids two competing secret-control planes.

A stable NAS FQDN keeps manifests portable across NAS restoration or address changes. At the time of this decision, `nas.truxonline.com` resolves on UMI to the current TrueNAS address. DNS must be restored before workloads that mount or contact the NAS are reconciled after disaster recovery.

## Consequences

- Argo CD prunes the obsolete `infisical-operator` Applications and their supporting manifests after this change reaches the tracked revision.
- Existing filenames containing `infisical` may remain temporarily where they are only historical names for ExternalSecret manifests; runtime objects must use External Secrets/OpenBao, not `InfisicalSecret` resources.
- `192.168.111.69` references are legacy Synology/documentation or separately scoped endpoints and are not rewritten by this decision.

## Validation

Before merging, verify no Argo CD Application, Kubernetes kind, or API group references the Infisical Operator, and build the affected OpenBao, NFS storage, and workload overlays.
