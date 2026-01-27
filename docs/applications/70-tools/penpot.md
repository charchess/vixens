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
- **Storage:** S3 (AWS/MinIO) via Infisical secrets
- **Mail:** SMTP via Mail Gateway

### Secrets (Infisical)
Path: `/tools/penpot`
- `PENPOT_SECRET_KEY`
- `PENPOT_DATABASE_URI`
- `PENPOT_REDIS_URI`
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `PENPOT_SMTP_USERNAME`
- `PENPOT_SMTP_PASSWORD`

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
