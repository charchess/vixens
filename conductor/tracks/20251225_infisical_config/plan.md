# Infisical Advanced Configuration Plan

## Goal
Extend Infisical usage to manage complex application configurations (specifically `sabnzbd.ini` and similar stateful apps) and centralize API keys. The goal is to move away from scattered secrets while preserving the ability for applications to write to their configuration files (maintaining UI settings).

## Phase 1: Analysis & Design
- [x] **Sabnzbd Hybrid Strategy**: Design the specific "InitContainer Patcher" pattern for Sabnzbd.
    -   *Mechanism:* Use an `initContainer` to read Infisical secrets (Env Vars) and patch the persistent `sabnzbd.ini` file using `sed` or a script before the main app starts.
    -   *Target Secrets:* API Key, Usenet Credentials (Host, User, Pass).
- [x] **Candidate App Identification**: Exhaustively analyze all deployed applications to identify which ones fit this "Stateful Config + Immutable Secrets" pattern.
    -   *Priority 1 (The *Arr Stack):* Sonarr, Radarr, Lidarr, Whisparr, Prowlarr.
    -   *Priority 2 (Infrastructure):* AdGuard Home (YAML config).
    -   *Priority 3 (Home Automation):* Home Assistant (Inject `secrets.yaml`).
- [x] **API Key Mapping**: Map out the dependencies (e.g., Sonarr needs Sabnzbd's API Key).

## Phase 2: Implementation (Dev)
- [ ] **Prototype Sabnzbd**: Implement the InitContainer pattern for `sabnzbd` in the `dev` environment.
    -   Create a generic "Config Patcher" script (using `sed` or Python).
    -   Configure `InfisicalSecret` to inject the raw values as Env Vars.
    -   Add `initContainer` to Deployment.
- [ ] **Generalize to *Arr Stack**: Apply the same pattern to the identified candidate apps (Sonarr, Radarr, etc.) to enforce their API Keys from Infisical.
- [ ] **Verification**: Validation of the application functionality in `dev` after the changes (ensure UI settings still save, but Secrets are forced on boot).

## Phase 3: Standardization
- [ ] **Documentation**: Document the "Hybrid Config Pattern" for future applications.
- [ ] **Promotion**: Prepare the changes for promotion to `prod`.