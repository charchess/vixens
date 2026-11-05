# OpenBao Fabric recovery acceptance contract (#4049)

**Status:** source-controlled recovery contract. It does not schedule, execute or authorize a backup, export, restore, unseal, audit activation, credential write, Kubernetes auth role, ESO store or production promotion.

## Purpose and gate

OpenBao Fabric may hold real runtime/tenant credentials only after a custody-approved physical recovery drill has passed every acceptance condition in this document. A PVC with `Retain`, a ZFS snapshot, an OpenBao health response, a green CI job or a backup object merely existing is not a restore proof.

This contract covers the single-node Raft OpenBao Fabric state currently declared by GitOps. It does not replace the separate Talos/cluster recovery, Infra OpenBao custody, application-native database recovery or future Shamir-to-Transit seal migration contracts.

## Authority and separation

- Git/Argo owns the declarative OpenBao Fabric server shape and this contract.
- Infra custody owns backup identity, encryption keys, independent destination, recovery shares, restore authorization and evidence storage.
- Fabric workloads, ESO, browsers and tenant operators never receive backup, restore, unseal, audit or administrative OpenBao authority.
- All secret values, root/recovery/unseal material, tokens, TLS private keys, snapshot payloads, encrypted object URLs and audit payloads remain outside Git, CRDs, issue/PR comments, CI output and chat.
- The historical OpenBao Infra server is not a runtime fallback. Its break-glass authority does not become a Fabric workload identity.

## Required recovery artefacts

Each drill must produce, in the approved custody system rather than Git:

1. a logical Raft snapshot taken with the supported OpenBao administrative procedure;
2. a cryptographic integrity record for that snapshot;
3. a local ZFS snapshot for rapid rollback, explicitly labelled as a secondary fast-recovery layer rather than the sole backup;
4. an encrypted copy in an independent failure domain, with retention; and
5. a non-secret evidence record identifying the drill, source revision, snapshot timestamp, restore target class, restore result and custodian approval.

The independent target must not depend on the same primary TrueNAS dataset, the source PVC or the live Fabric Pod. Encryption/decryption authority must be recoverable independently of Fabric's runtime credentials.

## Preconditions

Before a drill, the authorized custodian verifies and records only non-secret facts:

- source Pod is healthy and uses the expected retained PVC/PV identity;
- Raft health and initialized/sealed state are known;
- selected OpenBao binary/image version is restore-compatible with the source state;
- snapshot destination, retention and encryption recovery path were tested or are explicitly marked unavailable;
- an isolated restore environment exists, has no route from production workload namespaces, and has a fresh dedicated retained dataset/PVC;
- approved recovery/unseal custody is available without creating a new root key or reinitializing a cluster.

Abort rather than improvise if the storage identity, compatibility, custody, independent destination or isolation boundary is missing.

## Drill procedure

An authorized custodian performs the sensitive commands outside Git/CI/chat. The runbook must preserve this ordering:

1. establish a bounded maintenance/quiescence window and capture source health metadata;
2. take the supported logical Raft snapshot and integrity record;
3. take the local ZFS rollback snapshot, then copy the logical snapshot to the encrypted independent destination;
4. verify the encrypted copy's integrity at the destination without logging its contents;
5. provision or select an isolated target with fresh retained storage; never attach, delete or modify the production PVC/PV;
6. restore the logical snapshot only into that isolated target;
7. initialize no new cluster and generate no replacement root/recovery material;
8. unseal only through the approved custody process; validate compatible Raft state and perform a read-only logical health check; and
9. destroy ephemeral restore compute only according to a separately approved retention procedure, preserving drill evidence and retained recovery artefacts.

## Required acceptance evidence

The #4049 evidence record must prove, without sensitive data:

- source and restored instances report expected initialized/sealed readiness transitions;
- the restored instance reaches healthy Raft status after the approved unseal operation;
- a read-only logical status check succeeds on the restored instance;
- snapshot integrity matches at the independent destination and before restore;
- production Fabric identity/PVC/PV remains unchanged;
- production workload namespaces cannot route to the isolated restore endpoint;
- no credential, audit record content, token, recovery share, key, endpoint secret or snapshot bytes appears in public evidence.

## Failure and rollback

A failed snapshot, copy, decrypt, integrity check, restore, unseal, health check or isolation check fails this gate. It does not authorize retrying against production, restoring over production, deleting/recreating the retained PVC, running `sys/init`, issuing a runtime credential, activating ESO or falling back to OpenBao Infra.

Keep the source instance unchanged, preserve non-secret failure evidence, retain artefacts according to custody retention and open/continue an incident or #4049 subtask for the failed condition. A production recovery is a distinct, explicitly authorized maintenance operation.

## Gate outcome

Only a successful isolated drill permits the next, separately reviewed activation work: secret-safe audit custody, exact Kubernetes-auth tuple, closed tenant-provisioning boundary and namespace-scoped Fabric ESO store. It does not itself authorize any of those changes or production promotion.
