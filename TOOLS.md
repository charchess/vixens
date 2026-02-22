# TOOLS.md - Local Notes

Skills define _how_ tools work. This file is for _your_ specifics — the stuff that's unique to your setup.

## What Goes Here

Things like:

- Camera names and locations
- SSH hosts and aliases
- Preferred voices for TTS
- Speaker/room names
- Device nicknames
- Anything environment-specific

## Examples

```markdown
### UniFi / UDM
- Direct API Key/Password: zP8H357hrwp2v43g3IYNNvE3rznTqOft (Provided by Patron on 2026-02-12 for "full access")

### Cameras

- living-room → Main area, 180° wide angle
- front-door → Entrance, motion-triggered

### SSH

- home-server → 192.168.1.100, user: admin

### TTS

- Preferred voice: "Nova" (warm, slightly British)
- Default speaker: Kitchen HomePod
```

## Why Separate?

Skills are shared. Your setup is yours. Keeping them apart means you can update skills without losing your notes, and share skills without leaking your infrastructure.

### Moltbook
- Agent Name: `Lisa_Regente`
- Status: Claimed (as of 2026-02-13)
- Note: API key secured in `config/moltbook.json`.

### NocoDB
- URL: https://nocodb.truxonline.com
- API Token: `5VqYNpaWXBR67xHP1_-_bku-n1U25OoSESU9e9In` (Provided by Patron on 2026-02-14)
- Note: Use `xc-token` header for API requests.

### Vikunja
- URL: https://vikunja.truxonline.com
- User: `lisa`
- Password: `j'adoretonculmab3lleLisa`
- API Token: `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6IiIsImVtYWlsUmVtaW5kZXJzRW5hYmxlZCI6ZmFsc2UsImV4cCI6MTc3MTk3MDkyMiwiaWQiOjIsImlzTG9jYWxVc2VyIjp0cnVlLCJsb25nIjpmYWxzZSwibmFtZSI6IiIsInR5cGUiOjEsInVzZXJuYW1lIjoibGlzYSJ9.eUenaERQlu-RV40RET4OWIP0shQYkygBp5vykFL8DP0`

### Windows / fuu
- Host: 192.168.200.67 (Admin VLAN) / 192.168.199.104 (Home VLAN)
- User: `truxonline\administrator`
- Passwords: `S0d0m!e69` or `Chaton!3`
- Note: Provided by Patron on 2026-02-13.

### Media Services
- Kubeconfig: `/data/secrets/kubeconfig-prod`
- Namespace: `media`
- Music Assistant: https://musicassistant.truxonline.com
- Sonarr: `0a7176a0313941d2b52c23bd3d8de91a`
- Sabnzbd: `86b4c1cb9aa7da2e326906ba32cfba02`
- Radarr: `20596b075b7b28ccb1b6175a0f3ddcfe`
- Lidarr: `bc521449abb8eee97063d088a10efae3`
- Whisparr: `8584be670aab435fa5f191ac9feb1d98`
- Note: Provided by Patron on 2026-02-21.

