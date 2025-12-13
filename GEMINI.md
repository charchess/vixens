# Vixens GitOps Project - Gemini Instructions

## Project Overview
This project implements a GitOps approach for deploying and managing Kubernetes clusters based on Talos Linux. It encompasses the complete lifecycle, from infrastructure provisioning with Terraform to application management with ArgoCD. The architecture involves two control loops: a manual infrastructure loop via Terraform and an automated application loop via ArgoCD.

## Building and Running

### Prerequisites
- Terraform
- kubectl
- talosctl

### Installation & Usage

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/charchess/vixens.git
    cd vixens
    ```

2.  **Configure the `dev` environment (if needed):**
    ```bash
    cd terraform/environments/dev
    # Create or review terraform.tfvars
    ```

3.  **Initialize and apply Terraform for the `dev` environment (from project root):**
    ```bash
    terraform -chdir=terraform/environments/dev init -upgrade
    terraform -chdir=terraform/environments/dev apply -auto-approve
    ```

## Documentation
The project's main documentation is organized as follows:
- **Documentation Hierarchy**: See `docs/DOCUMENTATION-HIERARCHY.md`
- **Functional Recipe (RECETTE-FONCTIONNELLE)**: See `docs/RECETTE-FONCTIONNELLE.md`
- **Technical Recipe (RECETTE-TECHNIQUE)**: See `docs/RECETTE-TECHNIQUE.md`
- **Architecture Decision Records (ADR)**: See `docs/adr/`
- **Procedures**: See `docs/procedures/`
- **Reports**: See `docs/reports/`

## Development Conventions
-   **GitOps:** The project adheres to GitOps principles, managing infrastructure and application configurations through Git.
-   **Infrastructure as Code:** Terraform is used for provisioning and managing the underlying Kubernetes infrastructure.
-   **Application Deployment:** ArgoCD is utilized for automated deployment and management of applications within the Kubernetes clusters.
-   **Operating System:** Talos Linux is the chosen operating system for the Kubernetes nodes.

## Current Project Status
The project is currently in **Phase 2 - GitOps services deployment**.