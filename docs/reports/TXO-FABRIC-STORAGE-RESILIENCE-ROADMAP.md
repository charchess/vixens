# TXO Fabric — Long-Term Agent Storage Resilience and Fleet Recovery

**Status:** long-term architecture/roadmap note  
**Tracking issue:** #3760  
**Scope:** post-v0 planning; this document is not a current runtime contract  
**Scale horizon:** up to ~500,000 agents as an architectural stress case, not a committed capacity target

## 1. Why this document exists

TXO Fabric is moving from a small number of manually operated Hermes runtimes toward a platform that may eventually operate a large fleet of stateful agents.

The important scaling question is not only "can Kubernetes start many agents?". It is also:

> What happens when a storage path, node, CSI component, storage target, filesystem, or whole storage domain fails while many agents are active?

A design that requires an operator to inspect and repair one PVC at a time is acceptable for a lab. It is not an acceptable long-term operating model for a fleet.

The target invariant is therefore:

> **Loss or corruption of one agent runtime volume must eventually be recoverable without bespoke human intervention, and one storage-domain failure must have a bounded fleet impact.**

This does not mean every failure must be repaired automatically. Ambiguous or destructive cases may still require human approval. It means the platform must automate detection, correlation, fencing, recovery attempts, verification, and escalation rather than making manual PVC surgery the normal path.

## 2. Motivating incidents

### 2.1 2026-04-30 multi-volume iSCSI emergency_ro

The existing post-mortem documents a common storage/network interruption that pushed multiple ext4 filesystems into read-only/error state while pods could remain `Running`.

Key lessons already tracked:

- #3135 — detect read-only filesystems quickly;
- #3136 — increase iSCSI `replacement_timeout`;
- #3137 — document repeatable iSCSI recovery;
- #3138 — investigate multipath iSCSI;
- #3139 — evaluate XFS for new iSCSI PVCs.

See [2026-04-30 iSCSI emergency_ro multi-volume](../post-mortems/2026-04-30-iscsi-emergency-ro-multi-volume.md).

### 2.2 2026-10-03 UMI Hermes/Hindsight incident

On UMI, multiple iSCSI LUNs experienced I/O failures close together. Hermes private state and Hindsight PostgreSQL were both affected by the common storage path.

Observed behavior included:

- block I/O errors on multiple LUNs rather than one isolated application volume;
- Hermes ext4 entering `emergency_ro`;
- Hindsight PostgreSQL losing its block path but recovering with a clean filesystem after session recreation;
- a stale/zombie Hermes iSCSI session that had to be cleared before the block device could be trusted again;
- Hermes requiring offline `e2fsck` repair for real ext4 metadata inconsistencies;
- a pod/runtime that could appear alive at Kubernetes level while its persistent state was unusable;
- manual recovery requiring correlation of Kubernetes objects, CSI attachment, iSCSI IQN/session, block device, filesystem state, application state, and ArgoCD reconciliation.

The runtime was recovered successfully, but the operator workflow does not scale linearly with agent count.

This incident also reinforces that a filesystem change alone is not a complete solution. The primary fault class was loss/degradation of the underlying block path; ext4 reacted by protecting consistency.

## 3. Design principles

### 3.1 Agent identity is not a PVC

An `AgentIdentity` must remain the stable platform identity. A particular pod, block device, PV, PVC, filesystem UUID, or iSCSI session is replaceable infrastructure.

Long-term, the platform must know enough about durable agent state to recreate a runtime on replacement storage.

### 3.2 Bound the blast radius

No single node, network path, CSI controller, storage target, NAS, Kubernetes cluster, or future storage domain should be able to affect an unbounded fraction of the fleet.

The architecture must make failure domains explicit and measurable.

### 3.3 Recover failure domains, not N unrelated agents

If one common-path failure affects 500 agent volumes, the platform must recognize one storage-domain incident with 500 affected agents.

It must not generate 500 independent operator workflows that each rediscover the same cause.

### 3.4 Prefer reconstruction over filesystem surgery

Filesystem repair remains necessary for some failure modes, but it should become the exceptional path.

When a known-good snapshot/backup plus canonical platform state can safely recreate a runtime, replacement/restore is preferable to invasive repair of an uncertain filesystem.

### 3.5 Detection is part of storage correctness

A pod being `Running` is not proof that its persistent state is writable.

Storage health requires explicit signals such as writeability, filesystem health, CSI/session state and application-level persistence checks.

### 3.6 Storage products are implementation choices

TrueNAS, democratic-csi, ext4, XFS, multipath, Ceph/RBD or another future backend are not the TXO Fabric contract.

The Fabric contract is durability, isolation, bounded blast radius, recoverability and observable health.

## 4. Agent state classification

Before recovery can be automated, TXO Fabric must classify what state is authoritative and where it belongs.

The current and future model should distinguish at least:

| State class | Examples | Long-term expectation |
|---|---|---|
| Fabric intent | AgentIdentity, runtime profile, policy, integration bindings | Canonical in platform/Git/API state; reconstructible |
| Long-term memory | Hindsight bank/history | Durable external service; isolated from runtime PVC |
| Provider/application secrets | gateway credentials, application secrets | Durable secret manager; not dependent on agent PVC |
| Shared workspace | tenant/team/user collaborative files | Durable shared storage with explicit ownership |
| Private agent state | Hermes configuration, private local state, private/local skills | Explicit durability contract; backup/hydration path required |
| Cache/derived state | caches, generated indexes, transient runtime artifacts | Disposable/rebuildable unless explicitly promoted to durable state |
| Logs/telemetry | runtime logs and metrics | Exported centrally; loss of one PVC must not erase operational evidence |

The critical unresolved area is **private agent state**.

TXO Fabric must define which portions of Hermes `/opt/data` are:

1. canonical and must survive;
2. durable but restorable from another source;
3. derived/reconstructible;
4. disposable.

Private/local skills are especially important: an agent must retain its ability to create and improve its own skills, but those skills must not remain durable only because one particular ext4 filesystem survived.

## 5. Failure domains

TXO Fabric must model at least these failure domains:

- process/container;
- pod;
- node;
- block device attachment;
- iSCSI session/path;
- CSI node plugin;
- CSI controller;
- network/VLAN/path;
- storage target/controller;
- storage pool;
- Kubernetes cluster;
- future fleet cell/region.

Every persistent AgentIdentity runtime should be attributable to a storage/failure domain so that correlated faults can be grouped.

Example:

```text
Fleet
 ├─ Cell A
 │   ├─ Cluster A1
 │   │   ├─ Storage domain A
 │   │   └─ Storage domain B
 │   └─ Cluster A2
 └─ Cell B
```

A failure in Storage domain A should affect only its bounded population, not the whole fleet.

## 6. Prevention

### 6.1 Current practical baseline

The current ext4 + iSCSI + TrueNAS/democratic-csi stack remains an acceptable development baseline while resilience work is incremental.

Changing filesystem is not, by itself, a fix for transport loss.

### 6.2 iSCSI timeout tuning

Issue #3136 tracks increasing `replacement_timeout` so short storage/network interruptions do not become immediate failed I/O and filesystem shutdown.

The correct timeout must be validated against real restart/failover behavior rather than copied blindly across environments.

### 6.3 Path redundancy

Issue #3138 tracks multipath feasibility.

Where supported, independent storage paths should make loss of one network path transparent to the filesystem.

"Multiple IPs" only count as meaningful redundancy when they do not all share the same hidden single point of failure.

### 6.4 Production node stability

WSL2 is useful for development but must not become a production storage data-plane dependency.

Production storage nodes should use a predictable Linux/kernel/network/initiator environment with controlled suspend, networking and device lifecycle behavior.

### 6.5 Filesystem selection

Filesystem choice remains worth testing (#3139), but it is a secondary resilience layer.

Requirements for any candidate filesystem include:

- well-understood behavior on block I/O failure;
- mature offline/online repair tooling;
- CSI compatibility;
- predictable Kubernetes mount behavior;
- operational observability;
- acceptable repair/restore time.

A filesystem that merely reacts differently to a lost block path does not remove the path failure.

## 7. Detection and fleet-level correlation

### 7.1 Signals

The platform should consume and correlate:

- filesystem read-only state / `emergency_ro`;
- ext4/XFS/Btrfs filesystem errors where applicable;
- block I/O errors and SCSI command timeouts;
- iSCSI NOP/session recovery failures;
- CSI attach/mount/unmount errors;
- CSI node/controller restart or health anomalies;
- stale `VolumeAttachment`;
- application write failures;
- storage latency and saturation;
- snapshot/backup failures.

Issue #3135 is the minimum first step: a filesystem becoming read-only must alert quickly.

### 7.2 Storage canaries

Each storage/failure domain should eventually expose an active canary that performs a small recurring sequence such as:

```text
write -> fsync -> read -> checksum -> delete
```

The canary is not a replacement for application monitoring. It provides an early domain-level signal independent of any one agent.

### 7.3 Correlation

A fleet-level incident engine should recognize patterns such as:

```text
80 AgentIdentity volumes
same node/path/storage target
same 30-second window
same I/O failure signature

=> one StorageDomainIncident
   affectedAgents = 80
```

This avoids alert storms and enables one recovery policy to coordinate the affected population.

## 8. Recovery automation

Long-term recovery should be represented as an explicit state machine rather than ad-hoc shell commands.

Example:

```text
Healthy
  |
  | storage health signal
  v
Suspect
  |
  | confirmed persistent fault
  v
Fenced
  |
  | workload stopped, writable ownership released
  v
Recovering
  |
  +--> reattach/reconnect
  +--> restore/clone snapshot
  +--> controlled filesystem repair
  |
  v
Verifying
  |
  +--> mount writable
  +--> write/fsync/read test
  +--> runtime health
  +--> persistence check
  |
  +--> Healthy
  |
  +--> Escalated
```

### 8.1 Safety rules

Automated recovery must never:

- run filesystem repair on a mounted writable filesystem;
- blindly logout unrelated iSCSI sessions;
- force-delete workloads as a normal recovery step;
- assume a device name such as `/dev/sdd` is stable identity;
- repair a volume before mapping exact AgentIdentity -> PVC -> PV -> target -> attachment;
- destroy the last recoverable copy before snapshot/backup policy allows it.

### 8.2 Preferred recovery order

Where possible:

1. identify and correlate the failure domain;
2. stop/fence the affected runtime;
3. ensure old writable ownership is released;
4. retry clean transport/session recovery;
5. verify filesystem without modifying it;
6. if safe, repair or restore according to policy;
7. remount on controlled ownership;
8. perform RW/fsync verification;
9. start runtime;
10. perform application-level persistence/health check;
11. return to healthy or escalate with captured diagnostics.

### 8.3 Recovery controller

A future Fabric storage-recovery controller may own this workflow, but the controller design is not decided by this document.

Its important contract is that recovery is driven by stable identities and declared policy, not by mutable pod names or block-device letters.

## 9. Backup, snapshot and restore

A backup that has never been restored is not yet a proven recovery mechanism.

TXO Fabric needs explicit recovery classes for private agent state, including:

- required RPO;
- required RTO;
- snapshot frequency;
- backup independence from the primary storage failure domain;
- retention;
- encryption/access boundaries;
- periodic restore tests;
- restore into a replacement runtime without ambiguous dual writers.

Exact RPO/RTO values are product decisions and are intentionally not invented here.

## 10. Large-fleet architecture: cells

The "~500,000 agents" figure is a deliberate stress test.

It does **not** mean TXO Fabric should aim to create 500,000 valuable PVCs/LUNs behind one NAS or one Kubernetes cluster.

At large scale the expected architecture is cellular/sharded:

```text
                   TXO control plane
                         |
        +----------------+----------------+
        |                |                |
      Cell A           Cell B           Cell C
        |                |                |
   cluster(s)        cluster(s)        cluster(s)
   storage A         storage B         storage C
   bounded fleet     bounded fleet     bounded fleet
```

A cell should have:

- a bounded agent population;
- bounded compute and storage failure domains;
- independent health signals;
- independent recovery capacity;
- explicit placement metadata;
- a way for the control plane to stop placing new agents into a degraded cell.

The correct number of agents per cell is **not specified here**. It must come from benchmark data, storage latency, controller behavior, recovery concurrency and failure-injection tests.

## 11. Scale triggers

The platform should define measurable triggers that force architectural review before scale becomes an incident.

Examples of trigger categories:

- PVC/LUN count per storage target;
- attach/mount latency percentiles;
- storage IOPS/latency saturation;
- snapshot duration and backlog;
- recovery concurrency;
- mean time to detect storage faults;
- mean time to restore one agent;
- maximum number of agents in one failure domain;
- time to recover an entire cell;
- controller/API pressure caused by storage objects;
- backup/restore throughput.

Threshold values are intentionally TBD until measured.

## 12. Incremental roadmap

### Phase 0 — current hardening

Use existing issues rather than creating competing implementations:

- #3135 — read-only filesystem alerting;
- #3136 — iSCSI timeout resilience;
- #3137 — repeatable recovery runbook/post-mortem;
- #3138 — multipath investigation;
- #3139 — XFS experiment.

### Phase 1 — make state and recovery observable

- define the private AgentIdentity state classification;
- export storage-domain identity into operational metadata;
- add storage write/fsync/read canaries;
- correlate multi-volume incidents;
- define agent-state RPO/RTO classes;
- prove restore from backup/snapshot for a disposable test agent.

### Phase 2 — make one-agent recovery routine

- create replacement runtime storage;
- hydrate canonical/private state;
- restore snapshot/backup where needed;
- reconnect Hindsight and platform-managed secrets;
- verify persistence automatically;
- demonstrate recovery without operator block-device surgery.

### Phase 3 — automate failure-domain recovery

- storage recovery state machine/controller;
- concurrency controls and fencing;
- multi-agent incident recovery;
- degraded-domain placement prevention;
- controlled escalation and forensic capture.

### Phase 4 — cell-scale resilience

- benchmark cell size;
- benchmark storage backend limits;
- validate loss of one storage path;
- validate loss of one storage target;
- validate loss of one node;
- validate recovery of many agents concurrently;
- decide when a distributed storage backend or another topology is justified.

## 13. Non-goals

This document does not:

- require an immediate migration away from TrueNAS/democratic-csi;
- declare ext4, XFS, Btrfs, ZFS or any distributed storage product the final answer;
- promise 500,000-agent capacity;
- replace application-specific backup requirements;
- authorize automatic destructive `fsck` on arbitrary volumes;
- replace #3135–#3139;
- make legacy Hermes standalone storage semantics the permanent Fabric model.

## 14. Long-term acceptance invariants

TXO Fabric should not claim fleet-scale storage maturity until it can demonstrate that:

1. a filesystem becoming read-only is detected automatically;
2. a common-path failure is correlated as one incident;
3. one storage-domain failure has a documented and bounded blast radius;
4. private agent state has a declared durability/reconstruction class;
5. a lost runtime PVC can be replaced and an agent restored without bespoke per-volume shell surgery;
6. backups/snapshots are periodically restored in tests;
7. recovery automation fences writers before repair/restore;
8. platform placement avoids known-degraded storage domains;
9. cell sizing is based on failure-injection and capacity tests;
10. storage backend replacement does not change the stable AgentIdentity contract.

## References

- #3760 — tracking issue for this long-term architecture note
- #3135 — read-only filesystem alert
- #3136 — iSCSI replacement timeout
- #3137 — iSCSI recovery runbook/post-mortem
- #3138 — multipath investigation
- #3139 — XFS StorageClass experiment
- #3673 — migrate existing Hermes agents to Fabric
- [Storage & Backup Strategy](STORAGE-STRATEGY.md)
- [2026-04-30 iSCSI emergency_ro post-mortem](../post-mortems/2026-04-30-iscsi-emergency-ro-multi-volume.md)
