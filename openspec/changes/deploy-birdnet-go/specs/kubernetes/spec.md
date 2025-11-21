# Kubernetes Specification - BirdNET-Go

## ADDED Requirements

### Requirement: BirdNET-Go SHALL Be Deployed for Bird Detection

BirdNET-Go SHALL be deployed as Deployment with audio stream processing and web UI.

#### Scenario: BirdNET-Go processes audio and detects birds
**GIVEN** BirdNET-Go is deployed with RTSP stream configured
**WHEN** bird sounds are present in audio stream
**THEN** pod SHALL log detected species with confidence score
**AND** detections SHALL be stored in persistent volume
**AND** Web UI SHALL display recent detections

### Requirement: BirdNET-Go SHALL Integrate with Home Assistant

Home Assistant SHALL be able to query BirdNET-Go API for recent bird detections.

#### Scenario: Home Assistant retrieves detections
**GIVEN** BirdNET-Go has detected birds in last hour
**WHEN** Home Assistant queries REST API `/api/detections`
**THEN** response SHALL include list of detected species
**AND** each detection SHALL have timestamp and confidence score
