# Home Assistant Kubernetes Deployment

This directory contains the base Kubernetes manifests for deploying Home Assistant on the vixens cluster.

## Architecture

### Container Image
- **Image**: `ghcr.io/home-assistant/home-assistant:latest`
- **Source**: Official Home Assistant container registry
- **Update Strategy**: `:latest` tag for dev/test, pinned versions for staging/prod

### Storage Strategy
- **PVC**: 10Gi persistent volume using `synology-iscsi-retain` StorageClass
- **Mount**: `/config` directory (contains configuration.yaml, automations, scripts, etc.)
- **Backend**: Synology NAS (192.168.111.69) via iSCSI CSI driver
- **Retention**: `Retain` policy prevents accidental data loss on PVC deletion

### Network Configuration

#### Services
- **ClusterIP Service** (`homeassistant-svc`): Internal cluster access on port 80
- **NodePort Service** (`homeassistant-nodeport`): Direct node access on port 30812 for initial setup

#### Ingress
- **Host**: Environment-specific subdomain (e.g., homeassistant.dev.truxonline.com)
- **TLS**: cert-manager with Let's Encrypt (staging for dev, production for prod)
- **Traefik**: IngressRoute with websecure entrypoint

#### Special Network Features
- **hostNetwork: true**: Enables mDNS/UPnP/SSDP discovery for local IoT devices
  - Required for protocols like Chromecast, Sonos, HomeKit discovery
  - Uses `dnsPolicy: ClusterFirstWithHostNet` to preserve cluster DNS resolution
- **Trusted Proxies**: ConfigMap configures Traefik reverse proxy support
  - `use_x_forwarded_for: true` - Trust X-Forwarded-For headers from Traefik
  - `trusted_proxies: 10.244.0.0/16` - Pod CIDR for internal proxy traffic

### Resource Limits
Default base configuration (overridden per environment):
- **CPU**: 250m-1000m (request-limit)
- **Memory**: 512Mi-2Gi (request-limit)

Production environments use higher limits (1 CPU, 2Gi memory).

### Health Checks
- **Readiness Probe**: HTTP GET /api/ (120s initial delay, 5s interval)
- **Liveness Probe**: HTTP GET /api/ (180s initial delay, 10s interval)
- **Long Delays**: Home Assistant can take 2-3 minutes to fully initialize on first boot

## Files

- `namespace.yaml` - Creates `homeassistant` namespace
- `pvc.yaml` - 10Gi persistent volume for /config directory
- `deployment.yaml` - Home Assistant deployment with 1 replica
- `service.yaml` - ClusterIP service (port 80)
- `service-nodeport.yaml` - NodePort service (port 30812) for initial access
- `configmap.yaml` - HTTP configuration for Traefik reverse proxy support
- `kustomization.yaml` - Kustomize base manifest

## Environment Overlays

Each environment (dev, test, staging, prod) has an overlay in `overlays/<env>/`:

### Dev Environment
- **Ingress**: https://homeassistant.dev.truxonline.com
- **TLS**: Let's Encrypt staging (avoid rate limits)
- **Image**: `:latest` tag
- **Resources**: 250m CPU, 512Mi memory

### Test Environment
- **Ingress**: https://homeassistant.test.truxonline.com
- **TLS**: Let's Encrypt staging
- **Image**: `:latest` tag
- **Resources**: 500m CPU, 1Gi memory

### Staging Environment
- **Ingress**: https://homeassistant.staging.truxonline.com
- **TLS**: Let's Encrypt production
- **Image**: Pinned version tag (e.g., `:2024.11.3`)
- **Resources**: 750m CPU, 1.5Gi memory

### Production Environment
- **Ingress**: https://homeassistant.truxonline.com
- **TLS**: Let's Encrypt production
- **Image**: Pinned version tag (tested in staging)
- **Resources**: 1000m CPU, 2Gi memory
- **PVC**: Consider increasing to 50Gi for long-term database growth

## Deployment

### Via ArgoCD (GitOps - Recommended)
```bash
# ArgoCD automatically syncs from Git
# Check sync status
kubectl get application homeassistant -n argocd
```

### Manual Deployment (Development Only)
```bash
# Dev environment
kubectl apply -k overlays/dev/

# Check deployment
kubectl get pods -n homeassistant
kubectl logs -n homeassistant -l app=homeassistant
```

## Initial Setup

### First Access
1. **Via Ingress** (after DNS propagation):
   ```
   https://homeassistant.dev.truxonline.com
   ```

2. **Via NodePort** (immediate access):
   ```
   http://<any-node-ip>:30812
   # Example: http://192.168.208.162:30812
   ```

### Setup Wizard
1. Create owner account (first user becomes admin)
2. Name your home location
3. Configure unit system (metric/imperial)
4. Allow anonymous usage statistics (optional)
5. Complete onboarding wizard

### Configuration
Home Assistant configuration persists in `/config` on the PVC:
- `configuration.yaml` - Main configuration file (auto-generated)
- `automations.yaml` - Automation definitions
- `scripts.yaml` - Script definitions
- `secrets.yaml` - Sensitive values (API keys, passwords)
- `home-assistant.log` - Application logs
- `home-assistant_v2.db` - SQLite database (history, states)

## Integration with Other Services

### Mosquitto MQTT Broker
- **Service**: `mosquitto-svc.mosquitto.svc.cluster.local:1883`
- **Config**: Add MQTT integration in Home Assistant UI
- **Authentication**: Anonymous initially, Authentik in Phase 3

### Authentik SSO (Phase 3)
- **Provider**: OpenID Connect
- **URL**: https://authentik.dev.truxonline.com
- **Config**: Add Authentik integration for single sign-on

## Troubleshooting

### Pod Not Starting
```bash
# Check pod status
kubectl get pods -n homeassistant

# View logs
kubectl logs -n homeassistant -l app=homeassistant

# Check PVC binding
kubectl get pvc -n homeassistant

# Verify iSCSI LUN on Synology NAS
# Login to Synology DSM → Storage Manager → iSCSI → LUN
```

### Common Issues

#### PVC Pending
- **Cause**: Synology CSI driver not running or iSCSI target unreachable
- **Fix**: Check `kubectl get pods -n synology-csi` and verify NAS connectivity

#### WebSocket Connection Failed
- **Cause**: Reverse proxy misconfiguration
- **Fix**: Verify `trusted_proxies` in ConfigMap includes Traefik pod CIDR (10.244.0.0/16)

#### Local Discovery Not Working
- **Cause**: `hostNetwork: true` not enabled
- **Fix**: Verify deployment has `hostNetwork: true` and `dnsPolicy: ClusterFirstWithHostNet`

#### Slow Startup (>3 minutes)
- **Cause**: Normal behavior on first boot or after updates
- **Action**: Wait for readiness probe to succeed, check logs for initialization progress

### Backup and Restore

#### Backup Configuration
```bash
# Create temporary pod with PVC mounted
kubectl run -it --rm backup --image=ubuntu --restart=Never \
  --overrides='{"spec":{"volumes":[{"name":"config","persistentVolumeClaim":{"claimName":"homeassistant-config"}}],"containers":[{"name":"backup","image":"ubuntu","volumeMounts":[{"name":"config","mountPath":"/config"}]}]}}' \
  -n homeassistant -- bash

# Inside pod: tar configuration
tar czf /config/backup-$(date +%Y%m%d-%H%M%S).tar.gz -C /config \
  --exclude='*.db-shm' --exclude='*.db-wal' --exclude='*.log' \
  configuration.yaml automations.yaml scripts.yaml secrets.yaml

# Copy backup to local machine
kubectl cp homeassistant/<pod-name>:/config/backup-*.tar.gz ./backup.tar.gz
```

#### Restore Configuration
```bash
# Upload backup to temporary pod
kubectl cp ./backup.tar.gz homeassistant/<temp-pod>:/config/

# Extract backup
tar xzf /config/backup.tar.gz -C /config/

# Restart Home Assistant
kubectl rollout restart deployment/homeassistant -n homeassistant
```

## Security Considerations

### Network Policies
- Home Assistant requires access to:
  - Internet (for integrations, updates, weather APIs)
  - Local network (for IoT device discovery via hostNetwork)
  - Mosquitto MQTT (1883/tcp)
  - Synology NAS (iSCSI 3260/tcp)

### Secrets Management
- Initial deployment uses Kubernetes Secrets for basic auth
- Phase 3: Migrate to Infisical for centralized secret management
- Sensitive values in `secrets.yaml` should be gitignored

### TLS Certificates
- Dev/Test: Let's Encrypt staging (avoid rate limits)
- Staging/Prod: Let's Encrypt production (90-day validity, auto-renewal)
- Traefik handles automatic certificate renewal via cert-manager

## Performance Tuning

### Database Optimization
- **Recorder**: Configure purge_keep_days in configuration.yaml (default: 10 days)
- **History**: Exclude noisy sensors to reduce database size
- **Example**:
  ```yaml
  recorder:
    purge_keep_days: 7
    exclude:
      domains:
        - automation
        - script
  ```

### Resource Scaling
- Monitor pod metrics: `kubectl top pod -n homeassistant`
- Increase CPU/memory if OOMKilled or CPU throttling occurs
- Consider horizontal scaling for multi-instance HA (requires external database)

## References

- [Home Assistant Container Documentation](https://www.home-assistant.io/installation/linux#install-home-assistant-container)
- [Home Assistant Configuration](https://www.home-assistant.io/docs/configuration/)
- [Synology CSI Driver](https://github.com/zebernst/synology-csi)
- [Traefik IngressRoute](https://doc.traefik.io/traefik/routing/providers/kubernetes-crd/)
