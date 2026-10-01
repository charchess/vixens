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

At pod bootstrap the operator resolves only the shared skill mounts authorized for
that agent and writes those paths to the named Hermes profile as
`skills.external_dirs`. An empty resolved set is written as `[]`, so removing a
binding also removes stale shared-library discovery on the next rollout.

Example resolved configuration for a sales agent:

```yaml
skills:
  external_dirs:
    - /workspace/skills/groups/sales/collaborative
    - /workspace/skills/organization/reference
```

The Hermes host gateway continues to use its profile multiplexing model; the
configuration is written explicitly to the agent's named profile with
`hermes -p <agentKey> config set ...`.

Hermes' current precedence remains part of the Fabric contract:

```text
project skills > local Hermes skills > external_dirs
```

Therefore an agent can import/copy a shared skill into its local skill directory,
customize it privately and naturally shadow the shared version without mutating
the common copy. This is intentional personalization, not a collision bug.

Collisions between multiple shared libraries still need to remain deterministic
and observable; the POC does not invent an additional precedence policy beyond
Hermes' existing behavior.

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
locally and tune it without changing the original.

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

`fabric-smoke` proves the first useful slice:

1. `skills-org-ref` exists on RWX storage and is mounted at
   `/workspace/skills/organization/reference` read-only to Hermes;
2. `skills-group-sales-rw` exists on RWX storage and is mounted at
   `/workspace/skills/groups/sales/collaborative` writable to `sales-probe`;
3. the outsider `probe` receives no sales skill mount;
4. Hermes' sales profile lists/discovers a skill created in the sales collaborative
   library through `skills.external_dirs`;
5. a management/curator write can seed the organization reference library and the
   agent can read/discover it but cannot modify it;
6. agent-local skills remain writable and private under `/opt/data`;
7. a same-named local skill shadows a shared external skill as Hermes currently
   defines;
8. shared skill data survives deletion/recreation of an individual agent.

Only after those physical checks pass is the shared-skill slice considered
accepted.
