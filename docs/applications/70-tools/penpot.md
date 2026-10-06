# Penpot

**Penpot** is the first Open Source design and prototyping platform meant for cross-domain teams.

## 📋 Status

| Environment | Deployed | Configured | Verified | Version |
| :--- | :--- | :--- | :--- | :--- |
| **Dev** | [x] | [x] | [ ] | latest |
| **Prod** | [x] | [x] | [ ] | latest |

*Note: Verified [ ] because dev cluster is offline.*

## 🚀 Access

- **Public URL:** https://design.truxonline.com (Prod)
- **Dev URL:** https://design.dev.truxonline.com (Dev)

## 🛠️ Configuration

### Infrastructure
- **Namespace:** `tools`
- **Replicas:** 1 (each component)
- **Ingress:** Traefik + Cert-Manager

### Dependencies
- **Database:** PostgreSQL Shared (`penpot` database)
- **Cache:** Redis Shared (db 0)
- **Storage:** S3 (AWS/MinIO) with credentials projected from OpenBao via External Secrets Operator
- **Mail:** SMTP via Mail Gateway

### Secrets (OpenBao / External Secrets Operator)
`ExternalSecret/penpot-secrets-sync` reads the environment-specific OpenBao path
(`vixens/dev/apps/70-tools/penpot` in the base; prod patches it to the prod path)
through `ClusterSecretStore/openbao` and materializes `Secret/penpot-secrets`.

Secret properties include the Penpot application/database/cache/S3/SMTP values
consumed by the workload. Do not recreate the retired Infisical path.

## 📦 Deployment Details

Deployed via ArgoCD App-of-Apps.
- **Backend:** `penpotapp/backend`
- **Frontend:** `penpotapp/frontend`
- **Exporter:** `penpotapp/exporter`

## 🔧 Troubleshooting

### Common Issues
- **Database connection:** Ensure PostgreSQL user and database are created.
- **S3 connection:** Verify AWS credentials and bucket existence.
- **Assets not loading:** Check `PENPOT_PUBLIC_URI` matches the ingress host.
