# Product Guide: Vixens GitOps Platform

## Initial Concept
A robust, automated GitOps platform for deploying and managing Kubernetes clusters based on Talos Linux. The system creates a bridge between infrastructure provisioning (Terraform) and application lifecycle management (ArgoCD), enforcing strict operational workflows while adhering to high engineering standards.

## Target Audience
- **AI Agents (Coding Agent):** Primary operators performing tasks, implementing features, and managing configurations via defined protocols.
- **Human Operators (User):** Strategic oversight, task assignment, validation, and final approval of changes.
- **System Administrators:** Responsible for the underlying physical or virtual infrastructure.

## Core Value Proposition
- **State-of-the-Art Architecture:** Implementation of modern cloud-native patterns, ensuring a scalable and maintainable infrastructure.
- **Strict GitOps Adherence:** All changes are tracked, versioned, and applied through Git, ensuring auditability and reproducibility.
- **Agent-Centric Workflow:** Designed from the ground up to support AI-driven development and operations (Archon integration).
- **Environment Isolation:** Clear separation between Dev, Test, Staging, and Prod environments with defined promotion paths.

## Key Features & Principles
- **DRY (Don't Repeat Yourself):** Utilization of centralized Terraform modules and ArgoCD app-templates to minimize configuration duplication and reduce maintenance overhead.
- **Infrastructure as Code (IaC):** Modular, best-practice-aligned Terraform configuration for reproducible infrastructure.
- **Continuous Delivery:** ArgoCD for automated synchronization and proactive drift detection.
- **Best Practices & Maintainability:** Strict adherence to industry standards for security, networking (Cilium), and secrets management (Infisical).
- **Automated Validation:** Integrated testing (Playwright/Curl) to verify deployment success beyond simple status checks.
