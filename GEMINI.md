# Vixens GitOps Project - Gemini Instructions

## Project Overview
This project implements a GitOps approach for deploying and managing Kubernetes clusters based on Talos Linux. It encompasses the complete lifecycle, from infrastructure provisioning with Terraform to application management with ArgoCD. The architecture involves two control loops: a manual infrastructure loop via Terraform and an automated application loop via ArgoCD.

## Development Conventions
-   **GitOps:** The project adheres to GitOps principles, managing infrastructure and application configurations through Git.
-   **Infrastructure as Code:** Terraform is used for provisioning and managing the underlying Kubernetes infrastructure.
-   **Application Deployment:** ArgoCD is utilized for automated deployment and management of applications within the Kubernetes clusters.
-   **Operating System:** Talos Linux is the chosen operating system for the Kubernetes nodes.

## IMPORTANT NOTICE
- you will only work on the git dev branch
- you will work in gitops mode (non gitops approach/commands might be used for troubleshoot and testing only)
- you will plan before acting, never act without user explicit consent

## ALLOWED COMMAND
- you may use kubectl (using terraforms/environments/dev/kubeconfig-dev)
- you may use talosctl (using terraforms/environments/dev/talosconfig-dev)
- you may use any local command needed to parse or analyze

## TOOLS
- you will use serena for file access (use serena onboarding command to get full instructions)
- you may use playwright to test web access (ou curl if playwright is down)
- you will try using archon to get documentation, if failing you may use web fetch
