# TXO Fabric — shared skill libraries

## Intent

Hermes agent-local skills remain the primary personalization and self-improvement
mechanism. TXO Fabric must not turn skills into centrally imposed immutable
packages or prevent an individual agent from creating and improving its own
procedural knowledge.

Fabric adds **shared skill libraries** beside those local skills so knowledge can
also move between agents, teams, companies and eventually trusted providers.

The resulting model has three trust/ownership modes:

1. **Local agent skills** — private, writable and agent-owned under the agent's
   existing `/opt/data` profile home. Hermes can create, modify and delete these
   normally.
2. **Collaborative shared skills** — writable libraries for an authorized tenant
   population. An agent may publish or improve a skill there and other authorized
   agents see the shared result.
3. **Reference shared skills** — read-only libraries for consumers. They contain
   validated company/team material or, later, trusted provider material.

Business files and shared skills are distinct trust domains even when they use the
same TrueNAS RWX storage profile. A writable document share must never
implicitly become behavioral input.

## Filesystem contract

Agent-local skills stay in the normal Hermes profile under `/opt/data`.

Shared libraries use a separate root:

```text
/workspace/skills/
├── organization/
│   ├── collaborative/
│   └── reference/
├── groups/
│   └── sales/
│       ├── collaborative/
│       └── reference/
└── users/
    └── bertrand/
        ├── collaborative/
        └── reference/
```

The visible path is the contract. Each enabled scope/mode is backed by its own
RWX PVC rather than one tenant-wide writable filesystem. Reference mounts use
`readOnly: true` in the Hermes container; collaborative mounts are writable.

PVC names intentionally distinguish the skill domain from business workspaces,
for example:

```text
ws-group-sales-rw       # business files
skills-group-sales-rw   # shared skills
skills-org-ref          # organization reference skills
```

Shared skill PVC lifecycle follows the tenant shared-workspace retention policy,
not any individual `AgentIdentity` lifecycle.

## v1alpha1 API

The existing organization/group/user scope is also the authorization population
for shared skills. No second IAM model is introduced.

Each scope can independently enable business-file and skill-library modes:

```yaml
spec:
  workspace:
    profileRef: shared-nfs
    retentionPolicy: Retain
    organization:
      reference: true
      skillsReference: true
    groups:
      - name: sales
        collaborative: true
        skillsCollaborative: true
```

The four booleans mean:

- `reference`: business files mounted read-only under `/workspace/shared/...`;
- `collaborative`: business files mounted writable under `/workspace/shared/...`;
- `skillsReference`: skill library mounted read-only under `/workspace/skills/...`;
- `skillsCollaborative`: skill library mounted writable under `/workspace/skills/...`.

`AgentIdentity.spec.access.groups` and `userRef` resolve both domains. A sales
agent receives only the declared sales file/skill scopes; an unrelated agent gets
no sales mount at all. Organization scopes remain tenant-wide.

## Hermes integration

Fabric does not replace Hermes' local skill system. The operator keeps creating
profiles with `--no-skills` so no template skills are injected, while local skill
creation remains available afterward in the private profile.

At pod bootstrap the operator writes one stable external skill root to the named
Hermes profile:

```yaml
skills:
  external_dirs:
    - /workspace/skills
```

Authorization does **not** depend on giving Hermes a dynamic list of paths. It is
enforced by Kubernetes mount topology: only the organization/group/user skill PVCs
resolved for that `AgentIdentity` are mounted below `/workspace/skills`. For
example, a sales agent may physically have:

```text
/workspace/skills/organization/reference
/workspace/skills/groups/sales/collaborative
```

while an unrelated agent has no sales subtree at all. Hermes can scan the stable
root recursively, but cannot discover content Kubernetes did not mount. If an
agent has no shared-skill scopes, `/workspace/skills` is absent and Hermes simply
has no shared library content to discover.

Using one stable root also avoids depending on a dynamic path list. Hermes
`v2026.9.24` validates `skills.external_dirs` as a list, so the CLI value must be
passed as a YAML/JSON list literal rather than as a scalar string. The configuration
is written explicitly to the agent's named profile with:

```text
hermes -p <agentKey> config set skills.external_dirs '["/workspace/skills"]'
```

The Hermes host gateway continues to use its profile multiplexing model.

Hermes `v2026.9.24` does **not** apply silent local-over-external precedence for a
bare skill name when a local skill and an `external_dirs` skill collide. It reports
both matching paths and refuses to guess. Fabric therefore treats same-name
collisions as an explicit, observable condition: callers must use an unambiguous
categorized/full relative path, or one of the colliding skills must be renamed.

An agent may still import/copy a shared skill locally and personalize it without
mutating the common copy, but the personalized copy should use an unambiguous
name/path rather than relying on implicit shadowing.

The POC does not invent an additional precedence policy beyond Hermes' actual
runtime behavior.

## Sharing workflow

The first implementation provides the storage/discovery boundary, not a final
human or agent-facing publication API.

The intended workflow is:

```text
local agent skill
      |
      | explicit publish/copy
      v
group or organization collaborative library
      |
      | review / curation
      v
reference library
```

A dedicated skill-curator agent may later be authorized to collaborative areas to
merge, improve and propose shared skills. Human review can promote validated
content to reference libraries.

Likewise, a commercial agent may publish a useful technique to the sales
collaborative library, while another commercial agent may import that shared skill
locally and tune it without changing the original. The local copy must remain
unambiguous with respect to visible shared skills.

## Trusted provider library

A platform/provider reference library is a valid next layer: TXO/hAIrem or another
trusted third party may offer reusable skills to tenants, and downloaded external
skills may enter that catalog after validation.

It is deliberately **not** implemented as a customer-name special case in the
generic controller. The future contract must be platform-owned, opt-in per tenant
or population, read-only to consuming agents, and separate from tenant-writable
collaborative storage.

OCI/Git/Hermes Hub or another source may later be used as an ingestion mechanism,
but immutable package distribution is not the core skill model. The core model is
local autonomy plus shared libraries with explicit trust and write boundaries.

## Initial physical acceptance

`fabric-smoke` has physically accepted the first useful slice:

1. [x] `skills-org-ref` exists on RWX storage and is mounted at
   `/workspace/skills/organization/reference` read-only to Hermes;
2. [x] `skills-group-sales-rw` exists on RWX storage and is mounted at
   `/workspace/skills/groups/sales/collaborative` writable to `sales-probe`;
3. [x] the outsider `probe` receives no sales skill mount;
4. [x] Hermes' sales profile uses `/workspace/skills` as its external root and
   lists/discovers a skill created in the sales collaborative library;
5. [x] a management/curator write can seed the organization reference library and the
   agent can read/discover it but cannot modify it;
6. [x] agent-local skills remain writable and private under `/opt/data`;
7. [x] a same-named local/shared pair produces an explicit collision and Hermes
   refuses ambiguous bare-name resolution rather than silently selecting one;
8. [x] shared skill data survives deletion/recreation of an individual
   `AgentIdentity`; with `runtime.storage.retentionPolicy: Delete`, the private
   runtime PVC is recreated and the old local skill disappears while the shared
   PVC identity and shared skill data remain intact.

The shared-skill slice is therefore physically accepted for the v0 POC contract.
