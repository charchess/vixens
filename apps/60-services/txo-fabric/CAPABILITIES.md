# TXO Fabric Hermes capability governance

This document is the living compatibility contract for executable Hermes capabilities used by TXO Fabric.

It complements the durable storage/skills and immutable-runtime decisions in ADR-033 and ADR-034. Capability policy is intentionally separate from skills: agents may create and improve local skills, while executable toolsets remain platform-owned.

## Runtime version covered

The first policy implementation targets:

- Hermes image baseline: `nousresearch/hermes-agent:v2026.9.24`
- upstream commit behind that tag: `f97608f178d1ffeca59860195ab7da295f7c8e5f`

An immutable TXO Hermes image may add reviewed runtime providers, but the Hermes toolset catalog belongs to the pinned Hermes source version. A runtime-image upgrade must review this compatibility inventory before promotion.

## Pinned Hermes toolset inventory

The pinned runtime contains these static toolset identities:

### Primary executable capability families

- `web`, `search`, `x_search`
- `vision`, `video`
- `image_gen`, `video_gen`
- `computer_use`
- `terminal`
- `skills`
- `browser`
- `cronjob`
- `file`
- `tts`
- `todo`
- `memory`
- `context_engine`
- `session_search`
- `connections`
- `clarify`
- `code_execution`
- `delegation`
- `homeassistant`
- `kanban`
- `discord`, `discord_admin`
- `yuanbao`
- `feishu_doc`, `feishu_drive`
- `spotify`

### Session/client/reserved surfaces

- `project`
- `bot_room`
- `desktop_ui`
- `setup`
- `coding`

These are not automatically equivalent to ordinary tenant capabilities. Some are client/session/role-coupled upstream surfaces.

### Composite aliases

Hermes also ships composite aliases such as `safe`, `debugging`, `hermes-cli`, `hermes-api-server`, `hermes-cron`, the messaging `hermes-*` aliases and `hermes-gateway`.

Fabric policy should prefer primary capability families instead of trying to express a deny policy with overlapping composites. Upstream applies disabled toolsets at individual-tool granularity; denying an overlapping composite could therefore remove tools intentionally enabled through another family.

### Dynamic plugin and MCP toolsets

Hermes can discover additional plugin/MCP toolsets at runtime. Fabric does not implicitly admit them.

The managed Fabric policy writes an explicit `platform_toolsets` list containing the admitted toolsets plus Hermes' `no_mcp` sentinel and pins `plugins.enabled` to an empty allow-list. A future platform-reviewed plugin/MCP capability requires an explicit extension of this contract.

## Policy states

`AgentRuntimeProfile.spec.capabilities.toolsets` owns the maximum executable surface for every AgentIdentity using the profile.

Each declared toolset has one state:

- **On** — always present in the platform selection; an AgentIdentity does not need to request it.
- **Off** — forbidden by the profile and included in the managed deny set.
- **AllowedOff** — permitted by the profile but denied until the AgentIdentity explicitly names it under `spec.runtime.capabilities.enableToolsets`.

An AgentIdentity may request only an `AllowedOff` entry. Requests for `On`, `Off` or undeclared toolsets fail closed and the runtime is not reconciled to a widened surface.

A Hermes runtime profile without an explicit toolset policy also fails closed.

## Enforcement boundary

Fabric renders the effective policy into a per-AgentIdentity ConfigMap and mounts it read-only at:

`/etc/hermes/config.yaml`

Hermes v2026.9.24 already implements this as its administrator-managed scope. Managed values overlay the user configuration after `/opt/data/config.yaml`, so editing the private agent configuration cannot widen the effective platform toolset list.

The generated managed policy pins:

- an explicit toolset list for every built-in Hermes platform surface in the pinned runtime;
- `no_mcp` to prevent implicit MCP server admission;
- `plugins.enabled: []` so user-installed plugins are not silently activated;
- `security.allow_lazy_installs: false`.

The pod also sets:

`HERMES_DISABLE_LAZY_INSTALLS=1`

This process-level guard is outside `/opt/data` and provides a second independent block against first-use dependency installation.

The effective policy revision is copied to the pod template and AgentIdentity status. A profile policy change or an `AllowedOff` activation therefore causes an observable Deployment rollout.

## Default POC profile

The initial `hermes-default` profile deliberately keeps the existing broad POC core available, including terminal, files, web/browser families, skills, memory, code execution and cron.

- `cronjob` is explicitly **On**.
- `delegation` is **AllowedOff** and requires an authorized AgentIdentity change.
- `computer_use` and other upstream opt-in/platform-specific capability families are **Off** unless deliberately promoted later.

This is capability admission, not a claim that every admitted tool is operational. Provider configuration, credentials and egress policy may independently make an admitted tool unavailable.

## Toolsets are not a sandbox

A toolset policy controls which Hermes tool schemas/capabilities are exposed to the model. It is not an operating-system syscall sandbox.

In particular, an agent with `terminal` or code-execution authority may be able to perform operations that overlap narrower toolsets. Network/credential authorization and policy-derived egress are a separate boundary tracked by #3687.

This distinction is intentional: #3686 governs executable capability registration; #3687 governs integration authority and network reachability.

## Skills remain independently extensible

ADR-033 remains unchanged.

- private/local skills live under `/opt/data` and may be created or improved by the agent;
- shared skill libraries remain under `/workspace/skills` with their existing authorization/mutability semantics;
- changing a skill does not grant a forbidden toolset;
- the `skills` toolset controls Hermes' skill-management executable API, not whether shared skill files become an executable capability.

## Cron persistence

In the pinned Hermes runtime, cron storage is anchored at the active `HERMES_HOME`; jobs are stored under `<HERMES_HOME>/cron/jobs.json`.

TXO Fabric sets `HERMES_HOME=/opt/data`. Therefore:

- pod replacement does not remove cron jobs;
- AgentIdentity recreation with private runtime retention `Retain` preserves the cron store with the retained PVC;
- AgentIdentity deletion with retention `Delete` deliberately discards that private cron state with the PVC.

The POC does not add cron quotas in #3686.

## Webhook direction

Webhook-triggered automation is a separate trigger surface from cron and remains follow-up work.

The desired direction is:

- ingress/authentication is platform-controlled;
- a webhook maps to an explicit TenantBundle/AgentIdentity and authorized workflow/capability set;
- untrusted webhook payloads must not implicitly select a broader toolset than the target AgentIdentity policy;
- webhook networking, credentials and external integration authorization must compose with #3687 rather than bypass it.

No webhook listener is introduced by the first toolset-policy slice.

## Acceptance focus

Physical acceptance for #3718 should prove:

1. an On toolset is present and callable;
2. an Off toolset stays absent after writing a conflicting local `/opt/data/config.yaml`;
3. `delegation` starts absent as AllowedOff;
4. enabling `delegation` through the AgentIdentity control path changes the policy revision, rolls the pod and exposes the toolset;
5. removing that opt-in returns it to denied;
6. local/shared skills remain intact;
7. lazy install remains disabled by both managed config and process environment.
