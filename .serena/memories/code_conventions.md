# Code Style and Conventions

## General
*   **Code & Commits:** English (variable names, resources, comments, commit messages).
*   **Documentation:** French (`.md` files).
*   **File Encoding:** UTF-8.
*   **Line Endings:** Unix (LF).

## Commit Messages
*   **Format:** Conventional Commits (`<type>(<scope>): <description>`)
*   **Types:** `feat`, `fix`, `chore`, `docs`, `refactor`, `test`, `ci`, `perf`, `style`
*   **Examples:**
    *   `feat(homeassistant): add MQTT integration`
    *   `fix(traefik): correct TLS ingress configuration`
    *   `chore(deps): update ArgoCD to v7.7.7`
    *   `docs(applications): add frigate documentation`
*   **Footer:** Add `🤖 Generated with [Claude Code]` and `Co-Authored-By: Claude Sonnet 4.5` for AI-generated commits

## Task Management
*   **Primary System:** Archon MCP Server (never use TodoWrite)
*   **Task Status Flow:** `todo` → `doing` → `review` → `done`
*   **Priority:** Higher `task_order` (0-100) = higher priority
*   **Assignment:** Tasks assigned to "Coding Agent" for AI work

## Terraform (`.tf`)
*   **Naming:** `snake_case` for all identifiers (resources, data sources, variables, outputs).
*   **Variables:** Must include `type` and `description`. `default` is highly recommended.
*   **Tagging:** Common tags for traceability (`project`, `environment`, `managed-by`).

## Kubernetes (`.yaml`)
*   **File Structure:** Start with comment indicating full path and `---` separator.
*   **Resource Naming:** `metadata.name` in `kebab-case`.
*   **Labels:** Use Kubernetes Recommended Labels.

## Multi-Environment Configuration
*   **Mirroring Principle:** New environments replicate `dev` structure.
*   **IP Addressing:** Last octet of IP address remains same across different service VLANs (e.g., ArgoCD (dev) `192.168.208.81` vs ArgoCD (test) `192.168.209.81`).
