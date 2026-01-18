# ADR-010: Use Static (Hydrated) Manifests for Infrastructure Applications

**Date:** 2026-01-01
**Status:** Deprecated
**Deciders:** User, Coding Agent
**Tags:** infra, manifests

---

## Context
The project currently uses the "inline" `helmCharts` generator feature of Kustomize (via ArgoCD) for several infrastructure applications, notably Reloader.

While convenient for initial deployment, this approach has significant drawbacks:
1.  **Opaqueness:** The actual YAML manifests running in the cluster are not visible in the Git repository.
2.  **Auditability:** It is impossible to review the exact changes introduced by a Helm chart upgrade in a Pull Request.
3.  **Debugging:** Troubleshooting requires inspecting the generated output in ArgoCD or running complex `kustomize build --enable-helm` commands locally, which depends on local Helm repo state.
4.  **Renovate Reliability:** Renovate has limited visibility into inline Helm charts compared to standard file-based dependencies.

## Decision
We will transition critical infrastructure applications to a **"Hydrated Manifests"** (Static) approach.

**Methodology:**
1.  Use `helm template` to generate the full YAML manifests from the upstream chart.
2.  Store the resulting YAML file in the application's `base/` directory (e.g., `apps/00-infra/reloader/base/manifests.yaml`).
3.  Reference this static file in `kustomize.yaml`.
4.  Apply environment-specific modifications via standard Kustomize patches (overlays).

**Scope:**
This decision applies initially to **Reloader** as a pilot. It will be evaluated for other infrastructure components (like VPA) based on the success of this migration.

## Consequences

### Positive
*   **GitOps Purity:** The Git repository becomes the absolute source of truth. What you see is what you get.
*   **Reviewability:** Every line of configuration change is visible in PRs.
*   **Stability:** Upgrades are intentional and explicit; no "magic" updates because a chart version changed in a registry.
*   **Performance:** Faster ArgoCD sync (no need to run Helm templating on the server).

### Negative
*   **Maintenance Overhead:** Upgrading an application requires a manual step (running `helm template` and commiting the result) rather than just bumping a version number.
*   **Verbosity:** The repository size increases slightly as it stores the full manifest content.

## Compliance
This change adheres to the project's principle of **Explicit over Implicit** configuration.
