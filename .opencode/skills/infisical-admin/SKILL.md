---
name: infisical-admin
description: "Infisical secrets management on the homelab. Manage projects, environments, secrets, service tokens, and integrations via CLI. Use when: any secrets/env vars management task on the homelab cluster."
---

# Infisical Admin — Homelab Skill

> Full Infisical secrets management via `infisical` CLI for the Vixens homelab cluster.

## Connection

```bash
# Login to Infisical instance
infisical login --domain https://infisical.truxonline.com

# Set default organization (if multiple orgs)
infisical organization set <org-id>

# Verify authentication
infisical user profile
```

- **Instance URL**: `https://infisical.truxonline.com`
- **Self-hosted**: Yes (running in K8s cluster)
- **Authentication**: Universal Auth, Service Tokens, or Machine Identities
- **CLI version**: Check with `infisical --version`

## Server Info

```bash
# Get current user info
infisical user profile

# List organizations
infisical organization list

# List projects
infisical projects list
```

---

## Projects

### List / Create / Delete

```bash
# List all projects
infisical projects list

# Create new project
infisical projects create --name "my-project" --organization <org-id>

# Get project details
infisical projects get --project <project-id>

# Delete project
infisical projects delete --project <project-id>
```

### Common Project Patterns

| Project | Purpose | Environments |
|---------|---------|--------------|
| `vixens-infra` | Infrastructure secrets | dev, staging, prod |
| `vixens-apps` | Application secrets | dev, staging, prod |
| `vixens-ci` | CI/CD secrets | ci, dev, prod |

---

## Environments

Projects have multiple environments (dev, staging, prod, etc.).

```bash
# List environments for a project
infisical environments list --project <project-id>

# Create environment
infisical environments create --project <project-id> --name staging --slug staging

# Update environment
infisical environments update --project <project-id> --id <env-id> --name "Staging New"

# Delete environment
infisical environments delete --project <project-id> --id <env-id>
```

**Standard environments**: `dev`, `staging`, `prod`

---

## Secrets

### Read Secrets

```bash
# List all secrets in an environment
infisical secrets list --project <project-id> --env <environment>

# Get a specific secret
infisical secrets get <SECRET_NAME> --project <project-id> --env <environment>

# Export secrets as environment variables (for local development)
infisical secrets export --project <project-id> --env dev

# Run command with injected secrets
infisical run --project <project-id> --env dev -- npm start
```

### Create / Update / Delete

```bash
# Create a secret
infisical secrets set <SECRET_NAME> <SECRET_VALUE> \
  --project <project-id> \
  --env <environment>

# Create secret with path (folder structure)
infisical secrets set <SECRET_NAME> <SECRET_VALUE> \
  --project <project-id> \
  --env <environment> \
  --path /app/database

# Update a secret (same as create - upsert behavior)
infisical secrets set <SECRET_NAME> <NEW_VALUE> \
  --project <project-id> \
  --env <environment>

# Delete a secret
infisical secrets delete <SECRET_NAME> \
  --project <project-id> \
  --env <environment>

# Delete with path
infisical secrets delete <SECRET_NAME> \
  --project <project-id> \
  --env <environment> \
  --path /app/database
```

### Bulk Operations

```bash
# Import secrets from .env file
infisical secrets import --project <project-id> --env dev --file .env

# Export secrets to .env file
infisical secrets export --project <project-id> --env dev --format dotenv > .env.dev

# Export as JSON
infisical secrets export --project <project-id> --env dev --format json > secrets.json
```

### Secret Paths (Folder Structure)

Infisical supports organizing secrets in paths (folders):

```bash
# Create secret in folder
infisical secrets set DB_HOST localhost \
  --project <project-id> \
  --env dev \
  --path /database

# List secrets in folder
infisical secrets list \
  --project <project-id> \
  --env dev \
  --path /database

# Nested paths
infisical secrets set API_KEY xyz123 \
  --project <project-id> \
  --env prod \
  --path /app/external/payment-gateway
```

**Common folder patterns**:
- `/database` - DB credentials
- `/api` - API keys and tokens
- `/external` - Third-party service credentials
- `/app` - Application-specific config

---

## Service Tokens

Service tokens allow applications/CI to access secrets without user authentication.

### Create Service Token

```bash
# Create service token with specific access
infisical service-token create \
  --project <project-id> \
  --name "CI Pipeline Token" \
  --environments dev,staging \
  --paths / \
  --ttl 0  # 0 = no expiration

# Output: Token will be displayed ONCE - save it immediately
```

### List / Revoke

```bash
# List service tokens
infisical service-token list --project <project-id>

# Revoke (delete) service token
infisical service-token revoke --id <token-id>
```

### Using Service Tokens

```bash
# Set token in environment (for CI/CD)
export INFISICAL_TOKEN="st.xxx.yyy.zzz"

# CLI will automatically use the token
infisical secrets list --project <project-id> --env dev

# In Kubernetes - create secret
kubectl create secret generic infisical-token \
  --from-literal=token="st.xxx.yyy.zzz" \
  -n <namespace>
```

---

## Machine Identities (Recommended for Production)

Machine Identities are more secure than service tokens (support rotation, fine-grained access).

```bash
# Create machine identity
infisical identities create --name "k8s-prod-app" --org-id <org-id>

# Create universal auth for the identity
infisical identities universal-auth create \
  --identity-id <identity-id> \
  --client-secret-ttl 7776000  # 90 days

# Grant access to project
infisical identities add --project <project-id> --identity-id <identity-id> --role viewer

# Authenticate using client ID and secret
export INFISICAL_UNIVERSAL_AUTH_CLIENT_ID="..."
export INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET="..."

infisical login --method universal-auth
```

---

## Users & Roles

### User Management

```bash
# List users in project
infisical users list --project <project-id>

# Invite user to project
infisical users invite --project <project-id> --email user@example.com --role developer

# Update user role
infisical users update --project <project-id> --user-id <user-id> --role admin

# Remove user from project
infisical users remove --project <project-id> --user-id <user-id>
```

### Roles

| Role | Permissions |
|------|-------------|
| **Owner** | Full control (project settings, delete project) |
| **Admin** | Manage secrets, users, integrations |
| **Developer** | Read/write secrets, no user management |
| **Viewer** | Read-only access to secrets |
| **No Access** | Cannot view or modify secrets |

---

## Integrations

Infisical can sync secrets to external platforms.

### Available Integrations

- **Kubernetes**: Sync secrets to K8s Secret objects
- **GitHub Actions**: Inject secrets into workflows
- **Vercel**: Sync env vars to Vercel projects
- **Docker**: Inject secrets into containers
- **AWS Secrets Manager**: Sync to AWS
- **HashiCorp Vault**: Bridge to Vault

### Kubernetes Integration (via Infisical Operator)

Deploy the Infisical Kubernetes Operator to auto-sync secrets:

```yaml
# InfisicalSecret CRD
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: app-secrets
  namespace: default
spec:
  hostAPI: https://infisical.truxonline.com/api
  resyncInterval: 60
  authentication:
    universalAuth:
      secretsScope:
        projectSlug: vixens-apps
        envSlug: prod
        secretsPath: /app
      credentialsRef:
        secretName: infisical-universal-auth
        secretNamespace: default
  managedSecretReference:
    secretName: app-secrets-synced
    secretNamespace: default
```

**Operator installation**:

```bash
# Add Helm repo
helm repo add infisical-helm-charts https://dl.cloudsmith.io/public/infisical/helm-charts/helm/charts/
helm repo update

# Install operator
helm install infisical-operator infisical-helm-charts/secrets-operator \
  --namespace infisical-operator-system \
  --create-namespace
```

---

## CLI Configuration

### Config File

Infisical CLI uses `~/.infisical/config.json` for configuration.

```bash
# View current config
cat ~/.infisical/config.json

# Set default project
infisical config set project-id <project-id>

# Set default environment
infisical config set env dev
```

### Environment Variables

```bash
# Authentication
export INFISICAL_TOKEN="st.xxx.yyy.zzz"  # Service token
export INFISICAL_UNIVERSAL_AUTH_CLIENT_ID="..."
export INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET="..."

# Instance URL (for self-hosted)
export INFISICAL_API_URL="https://infisical.truxonline.com/api"

# Default project/env
export INFISICAL_PROJECT_ID="<project-id>"
export INFISICAL_ENVIRONMENT="dev"
```

---

## Common Workflows

### Onboard New Application

```bash
# 1. Create project
infisical projects create --name "my-app" --organization <org-id>

# 2. Set secrets in dev
infisical secrets set DATABASE_URL "postgresql://..." --project <project-id> --env dev
infisical secrets set API_KEY "dev-key-123" --project <project-id> --env dev

# 3. Copy secrets to prod (manually set sensitive values)
infisical secrets set DATABASE_URL "postgresql://prod..." --project <project-id> --env prod
infisical secrets set API_KEY "prod-key-xyz" --project <project-id> --env prod

# 4. Create service token for CI
infisical service-token create --project <project-id> --name "CI Token" --environments dev,staging

# 5. Create machine identity for prod K8s
infisical identities create --name "k8s-my-app-prod" --org-id <org-id>
infisical identities universal-auth create --identity-id <identity-id>
infisical identities add --project <project-id> --identity-id <identity-id> --role viewer

# 6. Deploy InfisicalSecret CRD to K8s
kubectl apply -f infisical-secret.yaml
```

### Migrate from .env to Infisical

```bash
# 1. Export existing .env
cat .env

# 2. Import to Infisical dev
infisical secrets import --project <project-id> --env dev --file .env

# 3. Verify import
infisical secrets list --project <project-id> --env dev

# 4. Test local development
infisical run --project <project-id> --env dev -- npm start

# 5. Delete .env (now in Infisical)
rm .env
echo ".env" >> .gitignore
```

### Rotate Secrets

```bash
# 1. Generate new secret value
NEW_API_KEY=$(openssl rand -hex 32)

# 2. Update in Infisical
infisical secrets set API_KEY "$NEW_API_KEY" --project <project-id> --env prod

# 3. Restart pods (if using Infisical Operator, auto-reloads)
kubectl rollout restart deployment/my-app -n prod

# 4. Verify new secret is loaded
kubectl logs deployment/my-app -n prod | grep "API_KEY"
```

---

## Troubleshooting

### Authentication Issues

```bash
# Check if logged in
infisical user profile

# Re-authenticate
infisical logout
infisical login --domain https://infisical.truxonline.com

# Verify token (if using service token)
echo $INFISICAL_TOKEN
# Should start with "st."
```

### Secret Not Found

```bash
# Verify project ID
infisical projects list

# Verify environment
infisical environments list --project <project-id>

# Check path (secrets might be in subfolder)
infisical secrets list --project <project-id> --env dev --path /
infisical secrets list --project <project-id> --env dev --path /app
```

### CLI Version Mismatch

```bash
# Update CLI
# macOS/Linux
curl -1sLf 'https://dl.cloudsmith.io/public/infisical/infisical-cli/setup.sh' | sudo -E bash
sudo apt-get update && sudo apt-get install -y infisical

# Or via npm
npm install -g @infisical/cli

# Verify version
infisical --version
```

---

## Best Practices

### 1. Use Machine Identities for Production

Service tokens are simple but less secure. Use Machine Identities with Universal Auth for production workloads.

```bash
# ✅ Good: Machine Identity with rotation
infisical identities create --name "k8s-prod"

# ❌ Avoid: Long-lived service token
infisical service-token create --ttl 0
```

### 2. Organize Secrets with Paths

Use folder structure to group related secrets:

```bash
# ✅ Good: Organized paths
/database/host
/database/password
/api/stripe/key
/api/sendgrid/key

# ❌ Avoid: Flat structure
DATABASE_HOST
DATABASE_PASSWORD
STRIPE_API_KEY
SENDGRID_API_KEY
```

### 3. Use Environment-Specific Values

Never copy prod secrets to dev. Use placeholder values:

```bash
# Dev
infisical secrets set DATABASE_URL "postgresql://localhost:5432/dev" --env dev

# Prod
infisical secrets set DATABASE_URL "postgresql://prod.db:5432/prod" --env prod
```

### 4. Enable Secret Versioning

Infisical tracks secret history. To rollback:

```bash
# View secret history
infisical secrets history <SECRET_NAME> --project <project-id> --env prod

# Rollback to previous version (manual)
infisical secrets set <SECRET_NAME> <OLD_VALUE> --project <project-id> --env prod
```

### 5. Audit Logs

Enable and monitor audit logs for secret access:

```bash
# View audit logs (via Web UI)
# Navigate to Project > Settings > Audit Logs
```

---

## Naming Conventions (Homelab)

Existing patterns:

| Pattern | Example | Purpose |
|---------|---------|---------|
| `vixens-<category>-<app>` | `vixens-infra-postgres` | K8s app secrets |
| `vixens-ci-<app>` | `vixens-ci-argocd` | CI/CD secrets |
| `vixens-dev-<app>` | `vixens-dev-test-app` | Dev environment secrets |

Secret naming:
- Use `UPPER_SNAKE_CASE` for env vars
- Use descriptive names: `DATABASE_URL` not `DB`
- Group with prefixes: `STRIPE_API_KEY`, `STRIPE_WEBHOOK_SECRET`

---

## Integration with Vixens Cluster

### Infisical Instance

- **Namespace**: `infisical-system`
- **URL**: `https://infisical.truxonline.com`
- **Database**: PostgreSQL (via `vixens-prod-postgres` bucket)
- **Redis**: Local Redis cache

### Operator Deployment

```bash
# Check operator status
kubectl get pods -n infisical-operator-system

# View operator logs
kubectl logs -n infisical-operator-system deployment/infisical-operator

# List InfisicalSecret CRDs
kubectl get infisicalsecrets --all-namespaces
```

### Common InfisicalSecret Patterns

```yaml
# Pattern 1: Single app, all secrets
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: app-secrets
spec:
  hostAPI: https://infisical.truxonline.com/api
  authentication:
    universalAuth:
      secretsScope:
        projectSlug: vixens-apps
        envSlug: prod
        secretsPath: /app
      credentialsRef:
        secretName: infisical-auth
  managedSecretReference:
    secretName: app-secrets

# Pattern 2: Multiple paths
apiVersion: secrets.infisical.com/v1alpha1
kind: InfisicalSecret
metadata:
  name: app-db-secrets
spec:
  hostAPI: https://infisical.truxonline.com/api
  authentication:
    universalAuth:
      secretsScope:
        projectSlug: vixens-apps
        envSlug: prod
        secretsPath: /app/database
      credentialsRef:
        secretName: infisical-auth
  managedSecretReference:
    secretName: app-db-secrets
```

---

## Quick Reference

### Essential Commands

```bash
# Auth
infisical login --domain https://infisical.truxonline.com
infisical user profile

# Projects
infisical projects list
infisical projects create --name "my-project"

# Secrets
infisical secrets list --project <project-id> --env dev
infisical secrets set KEY value --project <project-id> --env dev
infisical secrets get KEY --project <project-id> --env dev
infisical secrets delete KEY --project <project-id> --env dev

# Service Tokens
infisical service-token create --project <project-id> --name "Token"
infisical service-token list --project <project-id>

# Run with secrets
infisical run --project <project-id> --env dev -- <command>

# Export
infisical secrets export --project <project-id> --env dev --format dotenv
```

### Web UI

- **URL**: https://infisical.truxonline.com
- **Projects**: Manage projects, environments, integrations
- **Audit Logs**: Track secret access and modifications
- **Settings**: User management, org settings, billing

---

**Last Updated**: 2026-03-10
