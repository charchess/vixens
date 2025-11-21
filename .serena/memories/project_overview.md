# Project Purpose & Overview
*   Infrastructure GitOps for deploying and managing Kubernetes clusters based on Talos Linux.
*   Covers full lifecycle: Terraform for infrastructure provisioning, ArgoCD for application management.
*   Two control loops: Infrastructure (manual, Terraform) and Application (automated, ArgoCD).
*   Current phase: Phase 2 - GitOps services deployment (with ongoing issues in `test` environment).
*   Objective: `OBJECTIF-02` in `DEFINITIONS.md`.

# Tech Stack
*   Kubernetes
*   Talos Linux
*   Terraform
*   ArgoCD
*   Git
*   Mozilla SOPS (for secrets management)
*   KMS (for SOPS encryption keys)
*   Hyper-V (for VMs, implied by Terraform provisioning)
*   DynamoDB (for Terraform state locking)
*   S3 (for Terraform backend)
