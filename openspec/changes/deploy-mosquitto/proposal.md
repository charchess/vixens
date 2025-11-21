# Deploy Mosquitto MQTT Broker

## Why
Home Assistant and IoT devices need MQTT for communication. Mosquitto is lightweight, stable MQTT broker.

## What Changes
Deploy Eclipse Mosquitto 2.x in `mosquitto` namespace with:
- StatefulSet (1 replica)
- PVC `mosquitto-config` (1Gi) + `mosquitto-data` (5Gi)
- Service ClusterIP port 1883 (internal only, no ingress)
- ConfigMap for mosquitto.conf (anonymous auth initially, Authentik later)
- ArgoCD Application

Non-Goals: External access (use NodePort if needed later), TLS (Phase 3), Auth (Authentik Phase 3)

## Testing
1. Deploy to dev
2. Test from Home Assistant: `mqtt.Client().connect("mosquitto.mosquitto.svc", 1883)`
3. Rollout to test/staging/prod

## Success Criteria
- ✅ Mosquitto pod Running
- ✅ Home Assistant can publish/subscribe to topics
- ✅ Persistent config/data across pod restarts
