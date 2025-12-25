# Infisical Advanced Configuration Plan

## Goal
Extend Infisical usage to manage complex application configurations (specifically `sabnzbd.ini` and similar stateful apps) and centralize API keys. The goal is to move away from scattered secrets while preserving the ability for applications to write to their configuration files (maintaining UI settings).

## Phase 1: Analysis & Design
- [x] **Sabnzbd Hybrid Strategy**: Design the specific "InitContainer Patcher" pattern for Sabnzbd.
- [x] **Candidate App Identification**: Exhaustively analyze all deployed applications to identify which ones fit this "Stateful Config + Immutable Secrets" pattern.
- [x] **API Key Mapping**: Map out the dependencies (e.g., Sonarr needs Sabnzbd's API Key).

## Phase 2: Implementation (Dev)
- [x] **Prototype Sabnzbd**: Implement the InitContainer pattern for `sabnzbd` in the `dev` environment.
- [x] **Generalize to *Arr Stack**: Apply the same pattern to the identified candidate apps (Sonarr, Radarr, etc.) to enforce their API Keys from Infisical.
- [x] **Automate inter-app links**: Implement cross-application key injection (Sabnzbd API key in Sonarr/Radarr, etc.) via DB patching.
- [ ] **Verification**: Validation of the application functionality in `dev` after the changes.

## Phase 3: Standardization
- [ ] **Documentation**: Document the "Hybrid Config Pattern" for future applications.
- [ ] **Promotion**: Prepare the changes for promotion to `prod`.
