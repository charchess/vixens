# UMI TrueNAS CSI Applications

These Application manifests replace the burned TrueNAS CSI plane for the UMI TrueNAS bootstrap.
Dev targets `feat/hermes-staging`; prod manifests target `prod-stable` after promotion.

Apply only after creating the `truenas-csi` namespace and both driver config Secrets, or ArgoCD will render pods that cannot start because the `existingConfigSecret` is missing.
