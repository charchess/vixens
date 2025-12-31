# Suggested Commands for Vixens Project Development

This document lists essential commands for developing and managing the Vixens project.

## 1. Quality Assurance & Validation

*   **YAML Linting:**
    ```bash
    find apps argocd -name "*.yaml" -o -name "*.yml" | xargs yamllint -c .yamllint
    ```
    *Description:* Lints all YAML files within the `apps` and `argocd` directories to ensure adherence to coding standards.

*   **Terraform Plan:**
    ```bash
    terraform plan
    ```
    *Description:* Generates an execution plan, showing what actions Terraform will take. Essential for review before applying changes.

*   **Terraform Apply:**
    ```bash
    terraform apply
    ```
    *Description:* Applies the changes required to reach the desired state of the configuration.

*   **Terraform Destroy:**
    ```bash
    terraform destroy
    ```
    *Description:* Destroys the Terraform-managed infrastructure. Use with extreme caution.

## 2. Git Operations

*   **Check Status:**
    ```bash
    git status
    ```
    *Description:* Shows the working tree status.

*   **Add Changes:**
    ```bash
    git add <file>
    git add .
    ```
    *Description:* Stages changes for the next commit.

*   **Commit Changes:**
    ```bash
    git commit -m "<type>(<scope>): <description>"
    # Examples:
    # git commit -m "feat(homeassistant): add MQTT integration"
    # git commit -m "fix(traefik): correct ingress TLS configuration"
    # git commit -m "chore(docs): update application documentation"
    ```
    *Description:* Records changes using Conventional Commits format (type: feat, fix, chore, docs, refactor, etc.).

*   **Push Changes:**
    ```bash
    git push
    ```
    *Description:* Uploads local repository content to a remote repository.

*   **Pull Changes:**
    ```bash
    git pull
    ```
    *Description:* Fetches from and integrates with another repository or a local branch.

*   **Create/Switch Branch:**
    ```bash
    git checkout -b <branch-name>
    git checkout <branch-name>
    ```
    *Description:* Creates a new branch and switches to it, or switches to an existing branch.

*   **Merge Branches:**
    ```bash
    git merge <branch-to-merge>
    ```
    *Description:* Joins two or more development histories together.

*   **Rebase Branch:**
    ```bash
    git rebase <base-branch>
    ```
    *Description:* Reapplies commits on top of another base tip.

## 3. Kubernetes & Cluster Management

*   **Kubectl (General Usage):**
    ```bash
    kubectl get pods -n <namespace>
    kubectl describe pod <pod-name> -n <namespace>
    kubectl logs <pod-name> -n <namespace>
    ```
    *Description:* Command-line tool for running commands against Kubernetes clusters.

*   **Talosctl (General Usage):**
    ```bash
    talosctl get nodes
    talosctl logs -n <namespace> <pod-name>
    ```
    *Description:* Command-line tool for managing Talos Linux clusters.

*   **ArgoCD CLI (General Usage):**
    ```bash
    argocd app list
    argocd app sync <app-name>
    ```
    *Description:* Command-line tool for managing ArgoCD applications.

## 4. General Linux Utilities

*   **List Directory Contents:**
    ```bash
    ls -la
    ```
    *Description:* Lists files and directories.

*   **Change Directory:**
    ```bash
    cd <directory>
    ```
    *Description:* Changes the current working directory.

*   **Search for Patterns in Files:**
    ```bash
    grep -r "pattern" .
    ```
    *Description:* Searches for a pattern in files.

*   **Find Files:**
    ```bash
    find . -name "*.tf"
    ```
    *Description:* Searches for files in a directory hierarchy.
