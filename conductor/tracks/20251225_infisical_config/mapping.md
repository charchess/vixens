# API Key Dependency Mapping

This document maps the inter-dependencies of API keys among the media applications.

## API Key Providers & Consumers

| Application | Provides Key | Needs Key From | Note |
|-------------|--------------|----------------|------|
| **Sabnzbd** | `SABNZBD__API_KEY` | - | Primary download client. |
| **Prowlarr** | `PROWLARR__API_KEY` | Sonarr, Radarr, Lidarr, Whisparr | Manages indexers for all *Arrs. |
| **Sonarr** | `SONARR__API_KEY` | Sabnzbd, Prowlarr | Series management. |
| **Radarr** | `RADARR__API_KEY` | Sabnzbd, Prowlarr | Movie management. |
| **Lidarr** | `LIDARR__API_KEY` | Sabnzbd, Prowlarr | Music management. |
| **Whisparr** | `WHISPARR__API_KEY` | Sabnzbd, Prowlarr | Adult content management. |
| **Mylar** | `MYLAR__API_KEY` | Sabnzbd | Comic management. |
| **Jellyseerr** | - | Sonarr, Radarr | Request management. |
| **Bazarr** | - | Sonarr, Radarr | Subtitle management. |

## Strategy for Cross-App Injection

Each application will have an `initContainer` that patches its own configuration file using environment variables provided by Infisical.

For cross-app dependencies (e.g., Sonarr needing Sabnzbd's API key), the secret must be duplicated or shared in Infisical at the consumer's path.

Example for Sonarr:
- Infisical Path: `/apps/20-media/sonarr`
- Secrets:
    - `SONARR__API_KEY` (Provided by Sonarr)
    - `SABNZBD__API_KEY` (Copied from Sabnzbd)
    - `PROWLARR__API_KEY` (Copied from Prowlarr)
