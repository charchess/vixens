# UMI TrueNAS CSI Applications

These Application manifests replace the burned TrueNAS CSI plane for the UMI TrueNAS bootstrap.
Dev value sources target `main`; production manifests target `prod-stable` after explicit promotion.

Apply only after creating the `truenas-csi` namespace and both driver config Secrets, or ArgoCD will render pods that cannot start because the `existingConfigSecret` is missing.
