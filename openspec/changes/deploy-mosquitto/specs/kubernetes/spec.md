# Kubernetes Specification - Mosquitto

## ADDED Requirements

### Requirement: Mosquitto MQTT Broker SHALL Be Deployed

Eclipse Mosquitto SHALL be deployed as StatefulSet for MQTT pub/sub messaging.

#### Scenario: Mosquitto pod running and accepting connections
**GIVEN** Mosquitto is deployed in namespace `mosquitto`
**WHEN** checking pod status
**THEN** pod SHALL be Running with 1/1 containers ready
**AND** Service SHALL expose port 1883 (ClusterIP, internal only)
**AND** clients SHALL be able to connect to `mosquitto.mosquitto.svc:1883`

### Requirement: Mosquitto SHALL Have Persistent Storage

Mosquitto configuration and data SHALL persist across pod restarts using PVCs.

#### Scenario: Configuration persists after pod restart
**GIVEN** Mosquitto has PVC `mosquitto-config` and `mosquitto-data`
**WHEN** pod is deleted and recreated
**THEN** configuration SHALL remain unchanged
**AND** retained messages SHALL still exist
