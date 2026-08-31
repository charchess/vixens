# UMI TrueNAS CSI Applications

These two Application manifests are intentionally separate from the older Synology CSI prod/dev app pair.
They target the `feat/hermes-staging` branch used by the live UMI lab Applications.

Apply only after creating the `truenas-csi` namespace and both driver config Secrets, or ArgoCD will render pods that cannot start because the `existingConfigSecret` is missing.
