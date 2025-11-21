# Deploy Music Assistant

## Why
Music Assistant (https://music-assistant.io) provides unified music streaming from multiple sources (Spotify, local files, radio) with Home Assistant integration for whole-home audio.

## What Changes
Deploy Music Assistant Server 2.x in `music-assistant` namespace:
- Deployment (1 replica)
- PVC `music-assistant-config` (1Gi) + mount NFS `media-shared` for music library
- Service ClusterIP port 8095
- Ingress with TLS for web UI
- Home Assistant integration via built-in add-on protocol

Non-Goals: Audio output (handled by Home Assistant), Spotify Premium (use free tier initially)

## Testing
1. Deploy to dev
2. Access web UI https://music-assistant.dev.truxonline.com
3. Configure music providers (local library from NFS)
4. Test Home Assistant media player integration
5. Play music to test audio routing

## Success Criteria
- ✅ Music Assistant pod Running
- ✅ Web UI accessible and configured
- ✅ NFS media library accessible
- ✅ Home Assistant detects Music Assistant as media source
- ✅ Can play music to Home Assistant media players
