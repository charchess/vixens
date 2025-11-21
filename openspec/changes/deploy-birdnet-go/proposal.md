# Deploy BirdNET-Go Bird Detection

## Why
BirdNET-Go provides real-time bird sound detection and species identification. Integrates with Home Assistant for automation triggers (e.g., notifications when rare bird detected).

## What Changes
Deploy tphakala/birdnet-go in `birdnet` namespace:
- Deployment (1 replica, requires audio input device or RTSP stream)
- PVC `birdnet-config` (1Gi) + `birdnet-data` (10Gi for recordings)
- Service ClusterIP port 8080
- Ingress with TLS for web UI
- ConfigMap for config.yaml (RTSP URL, detection sensitivity)
- Home Assistant integration via REST API

Non-Goals: Audio hardware passthrough (use RTSP stream instead), GPU acceleration (CPU sufficient)

## Testing
1. Deploy to dev with RTSP stream URL
2. Access web UI https://birdnet.dev.truxonline.com
3. Verify detections logged
4. Test Home Assistant sensor integration

## Success Criteria
- ✅ BirdNET-Go pod Running
- ✅ Web UI accessible via Ingress
- ✅ Detections saved to persistent storage
- ✅ Home Assistant can query recent detections
