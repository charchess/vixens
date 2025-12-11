# Report: Development References in Production and Base Configurations

This report details the findings regarding configurations that may be incorrectly pointing to development (dev) environments within production overlays and base configurations.

## 1. Configurations in `prod` overlays pointing to `dev` references:

The following configurations were identified within production (`prod`) environment overlays that contain references or values intended for the development (`dev`) environment. These should be reviewed and updated to reflect `prod` specific values.

*   **Application: `homepage`**
    *   **File:** `apps/70-tools/homepage/overlays/prod/ingress.yaml`
        *   **Line/Issue:** `secretName: homepage-dev-tls`
        *   **Description:** The Ingress resource in the production overlay is configured to use a TLS secret named `homepage-dev-tls`. This secret name suggests it belongs to the `dev` environment, which is inappropriate for a production deployment. It should likely be `homepage-prod-tls`.
    *   **File:** `apps/70-tools/homepage/overlays/prod/infisical-secret.yaml`
        *   **Line/Issue:** `envSlug: dev`
        *   **Description:** The Infisical secret configuration explicitly sets the `envSlug` to `dev`. In a production overlay, this should be set to `prod` to ensure it fetches secrets from the correct Infisical environment.
    *   **File:** `apps/70-tools/homepage/overlays/prod/ingress-redirect.yaml`
        *   **Line/Issue:** `- host: "homepage.dev.truxonline.com"`
        *   **Description:** The Ingress redirect configuration includes a host entry `homepage.dev.truxonline.com`. This indicates that the production environment is attempting to redirect to a `dev` environment host, which is incorrect. This should be updated to `homepage.prod.truxonline.com`.

*   **Application: `linkwarden`**
    *   **File:** `apps/70-tools/linkwarden/overlays/prod/kustomization.yaml`
        *   **Line/Issue:** `environment: dev`
        *   **Description:** The `kustomization.yaml` file for the `linkwarden` production overlay explicitly sets the `environment` to `dev`. This needs to be changed to `prod` to ensure correct environment targeting for Kustomize builds.
    *   **File:** `apps/70-tools/linkwarden/overlays/prod/infisical-secret.yaml`
        *   **Line/Issue:** `envSlug: dev`
        *   **Description:** Similar to `homepage`, the Infisical secret configuration for `linkwarden` in the production overlay sets the `envSlug` to `dev`. This should be updated to `prod`.

## 2. Configurations in `base` pointing to hardcoded development references (non-overridable):

A thorough search was conducted in `terraform/base` and `argocd/base` directories for any hardcoded references to "dev" environments that would not be overrideable by environment-specific configurations.

**No problematic hardcoded "dev" references were found.**

The instances of "dev" discovered in these base directories were determined to be:
*   **In comments:** Providing examples or explanations (e.g., in `terraform/base/variables.tf`, `argocd/base/app-templates/README.md`).
*   **In documentation:** Within README files explaining usage patterns.
*   **Standard system paths:** Such as `/dev/null` in `terraform/base/wait_for_k8s.tf`, which is a standard Linux device file and not an environment configuration.

These instances do not represent hardcoded development environment configurations that would negatively impact production deployments or prevent proper overlay functionality.

---
**Note:** The `gethomepage.dev` strings found in various `ingress.yaml` files are part of annotations used by the Homepage application for service discovery and do not indicate a reference to a development environment.