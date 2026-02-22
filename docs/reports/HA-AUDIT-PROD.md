# 🔍 Audit - Configuration Réelle HomeAssistant (Prod)

**Date:** 2026-02-22
**Environnement:** Production
**URL:** https://homeassistant.truxonline.com

---

## 📊 Informations Générales

| Paramètre | Valeur |
|-----------|--------|
| **Version HA** | 2026.2.3 |
| **Namespace** | homeassistant |
| **PVC** | homeassistant-config (150Gi) |
| **Base de données** | 1.8 GB (avec Litestream réplication) |
| **URL externe** | homeassistant.truxonline.com |

---

## 🔌 Intégrations Configurées (~80+)

### Équipements réseaux

| Intégration | Description | IP/Host |
|-------------|-------------|---------|
| **Shelly** | ~30 appareils (lumières, prises, thermostats, volets) | 192.168.207.x |
| **UniFi** | Contrôleur réseau | 192.168.201.1 |
| **MQTT** | Mosquitto | 192.168.201.70 |
| **Withings** | Santé (charchess) | cloud API |
| **Netatmo** | Thermostat et météo | cloud API |
| **Google Maps** | Localisation | cloud API |

### Domotique

| Intégration | Description |
|-------------|-------------|
| **Alarmo** | Système d'alarme complet |
| **Zigbee2MQTT** | Passerelle Zigbee |
| **Sure Petcare** | Chattière connectée |
| **Frigate** | Détection vidéo |
| **LLMVision** | Analyse AI des caméras |

### Multimédia

| Intégration | Description | IP/Host |
|-------------|-------------|---------|
| **Google Cast** | Chromecast | - |
| **DLNA** | Synelia | 192.168.204.69:50001 |
| **Hyperion** | Ambilight | 192.168.207.132:19444 |
| **WLED** | LEDs | 192.168.207.55 |

### Cloud & Services

| Intégration | Description |
|-------------|-------------|
| **Home Assistant Cloud** | Nabu Casa |
| **Google Assistant** | Commande vocale |
| **Discord** | Notifications bot |
| **HACS** | Custom components |

---

## 🤖 Automations (50+)

### Alarme & Sécurité

| ID | Alias | Description |
|----|-------|-------------|
| 1666435228025 | Alarmo Mode away si maison vide | Arme alarme en mode away |
| 1698802670098 | Alarme Detection en mode night | Détection intrusion nuit |
| 1698803244888 | Alarme Detection en mode away | Détection intrusion absent |
| 1699098374896 | Alarmo Mode night | Activation soir |
| 1699098388147 | Alarmo Désactivation matin | Désactivation automatique |
| 1715349215009 | Alarmo Mode home | Désarme si présence |
| 1723208677426 | Verouillage chattière soir | Verrouillage nocturne |
| 1723208766878 | Déverouillage chattière matin | Déverrouillage matinal |

### Aspirateur Robot (Roomby)

| ID | Alias | Description |
|----|-------|-------------|
| 1666435756916 | Roomby rentre à la base | Retour si présence |
| 1666801146898 | Roomby est de retour | Notification fin |
| 1688634168955 | Roomby est coincé | Alerte TTS |
| 1699882551913 | Roomby démarre à midi | Nettoyage auto si inactivity |
| 1701642360082 | Si Maison vide alors on lance roomby | Nettoyage si absent |

### Volets Roulants

| ID | Alias | Description |
|----|-------|-------------|
| 1699098709948 | Fermeture des volets le soir | Coucher du soleil +15min |
| 1699187167175 | Ouverture des volets le matin | Lever du soleil +15min |

### Chauffage & Thermostats

| ID | Alias | Description |
|----|-------|-------------|
| 1741852747542 | Control - thermostat enchauffe | Pacemaker every 5min |
| 1746608206001 | Pompe à chaleur - reglage consigne | Ajustement cible PAC |
| 1762431438918 | Chauffage salon - 7h - boost | Consigne 25°C à 7h |
| 1762432030809 | Chauffage salon - temperature normale | Consigne 21°C à 8h |
| 1733666616199 | Helper cost_tempo | Calcul tarif EDF Tempo |

### Caméras & AI

| ID | Alias | Description |
|----|-------|-------------|
| 1742044247204 | test - ai on exterieur | Analyse événements extérieurs |
| 1747466134888 | test - ai - hass response | Analyse motion cameras |
| 1698803244888 | Alarme Detection | LLMVision sur détection |

### Éclairage

| ID | Alias | Description |
|----|-------|-------------|
| 1732337973887 | Lumière escalier off | Timer 2min |
| 1732359556144 | Lumière couloir off | Timer 3min |
| 1737179149920 | WC off -> couloir on | Motion detection |
| 1754561698441 | Turn Off Dining Room Lights | No motion detection |
| 1761147442873 | Eteindre lumière chambre | Si pas de présence |

### 3D Printing

| ID | Alias | Description |
|----|-------|-------------|
| 1738448354659 | 3D printing - printing done | Notification fin |
| 1738453980112 | 3D printing - problème | Notification problème |
| 1739426744705 | 3D print - filament runout | Détection clogging |

### Monitoring & Notifications

| ID | Alias | Description |
|----|-------|-------------|
| 1670646568537 | Low battery level | Alerte batterie faible |
| 1687289857479 | Coupure de courant debut | Alerte voltage < 200V |
| 1700150815395 | Coupure de courant fin | Retour normale |
| 1726422541645 | Camera down | Alerte caméra HS |
| 1726424276251 | Camera up | Alerte caméra revenue |
| 1752503727803 | Server down | Webhook monitoring |
| 1748884001688 | Watchdog thermostat | Check every 5min |

### Animaux

| ID | Alias | Description |
|----|-------|-------------|
| 1714915616577 | Mouvement chat | Notification Praline/Vanille |
| 1770626236418 | Chattière - Entrée uniquement | Lock in |
| 1770626253673 | Chattière - Sortie autorisée | Lock out |
| 1770626279786 | Chattière - Ouverte | Unlock |
| 1770627923217 | Chattière - Fermée | Lock all |

### Divers

| ID | Alias | Description |
|----|-------|-------------|
| 1747466134888 | Health Report of the Night | Résumé sommeil Withings |
| 1734727760629 | Notifications temperatures | Alerte temperature |

---

## 🔧 Composants Personnalisés (HACS - 34)

### Sécurité & Alarme
- **alarmo** - Système d'alarme
- **blitzortung** - Détection foudre

### Thermostat & Chauffage
- **versatile_thermostat** - Thermostat avancé
- **thermal_comfort** - Confort thermique

### Lumière & Volets
- **adaptive_cover** - Volets adaptatifs
- **shadow_control** - Contrôle ombrage

### Presence & Mouvement
- **area_occupancy** - Occupation des pièces
- **bermuda** - Presence detection
- **magic_areas** - Zones logiques
- **network_scanner** - Scan réseau

### Caméra & Vidéo
- **frigate** - Détection vidéo
- **llmvision** - Analyse AI

### Maison Connectée
- **localtuya** - Appareils Tuya
- **homewhiz** - Appareils Electrolux
- **electrolux_status** - Status Electrolux

### Services Cloud
- **hacs** - HACS
- **weathersense** - Météo

### Objets Connectés
- **nest_protect** - Détecteurs Nest
- **surepetcare** - Chattière
- **candy** - Lave-linge Candy
- **unifi_voucher** - WiFi guests

### Energie
- **rtetempo** - EDF Tempo

### Automatisation
- **scheduler** - Planificateur
- **presence_simulation** - Simulation présence
- **watchman** - Rapport automatisations
- **ai_automation_suggester** - Suggestions AI
- **webhook_service** - Webhooks

### Services
- **cafe** - Commandes HA
- **battery_consumption** - Batterie
- **ha_sunforecast_plus** - Prévisions soleil

### Maison
- **advanced_snapshot** - Snapshots
- **hass_agent** - Agent Windows
- **moonraker** - Contrôle imprimantes 3D
- **bodypetscale** - Balance pets
- **material_symbols** - Icônes
- **kubernetes** -监控 Kubernetes

### Conversation
- **llama_conversation** - AI conversation

---

## ⚠️ Constatations

### Positives ✅
1. **Configuration restore fonctionnelle** - Init containers restore-config et restore-db actifs
2. **Sauvegarde Litestream** - DB répliquée en temps réel
3. **Diversité des intégrations** - Maison bien connectée
4. **Automatisations complètes** - Alarme, chauffage, Roomby, cameras

### Points d'Attention ⚠️

1. **Base de données volumineuse** - 1.8 GB
   - Peut nécessiter un cleanup périodique
   - Recorder purge recommended

2. **Appareils désactivés**
   - Tuya (ignored)
   - Nest Protect (ignored)
   - ESPHome test-1 à test-5 (disabled)
   - Plusieurs Shelly désactivés

3. **Tokens dans configuration**
   - Google Maps (cookies)
   - Discord (token)
   - Netatmo (token)
   - SurePetcare (password)
   - à migrer vers secrets

4. **Custom components nombreux** - 34 installations
   - Risque de conflit après mise à jour
   - Maintenir compatibilité

---

## 📋 Appareils Shelly Configurés

### Lumières & Prises
| Nom | IP | Modèle |
|-----|-----|--------|
| Sapin | 192.168.207.139 | SHPLG-S |
| Prise Home Cinema | 192.168.207.19 | SHPLG-S |
| Prise temp | 192.168.207.237 | SNPL-00112EU |
| Prise temp 2 | 192.168.207.167 | SHPLG-S (disabled) |
| Lumière cuisine | 192.168.207.102 | SNSW-001X16EU |
| Lumière cour | 192.168.207.175 | SNSW-001X16EU |
| Lumière chauffagerie | 192.168.207.85 | SHSW-L |
| Lumières couloir | 192.168.207.173 | SPSW-201PE16EU |
| Eclairages_1 | 192.168.207.206 | SPSW-003XE16EU |
| Ventilateur chambre | 192.168.207.75 | SHSW-1 |
| Spots salon | 192.168.207.223 | SHDM-2 |
| Volet baie vitrée | 192.168.207.230 | SNSW-002P16EU |
| PresenceChaufferie | 192.168.207.127 | SHMOS-01 |
| Thermostat Baie réseau | 192.168.207.175 | SHHT-1 |

### Volets Roulants (VR)
| Nom | IP | Modèle |
|-----|-----|--------|
| VR Baie Vitrée | 192.168.132.137 | SHSW-25 (disabled) |
| VR Chambre Camille | 192.168.132.145 | SHSW-25 (disabled) |
| VR Chambre haut | 192.168.132.164 | SHSW-25 (disabled) |
| VR Chambre Bas | 192.168.132.132 | SHSW-25 (disabled) |
| VR Couloir haut | 192.168.132.157 | SHSW-25 (disabled) |
| VR Cuisine | 192.168.132.191 | SHSW-25 (disabled) |
| VR salon | 192.168.132.128 | SHSW-25 (disabled) |
| VR Salle à Manger | 192.168.132.127 | SHSW-25 (disabled) |

### Thermostats (Shelly TRV)
| Nom | IP | Modèle |
|-----|-----|--------|
| Radiateur chambre | 192.168.207.62 | SHTRV-01 |
| Radiateur cuisine | 192.168.207.129 | SHTRV-01 |
| Radiateur chambre fifilles | 192.168.207.51 | SHTRV-01 (disabled) |
| Radiateur salon M | 192.168.207.220 | SHTRV-01 |
| Radiateur salon N | 192.168.207.164 | SHTRV-01 (disabled) |
| Radiateur salon S | 192.168.207.49 | SHTRV-01 |
| Radiateur bureau | 192.168.207.29 | SHTRV-01 |
| HT Chambre maman | 192.168.207.168 | SNSN-0013A |

### ESPHome (Désactivés)
| Nom | IP |
|-----|-----|
| test-1 | 192.168.207.51 |
| test-2 | 192.168.207.164 |
| test-3 | 192.168.207.53 |
| test-4 | 192.168.207.166 |
| test-5 | 192.168.207.55 |

---

## 🔄 Restore & Backup

### Init Containers Configurés
1. **fix-perms** - Fix permissions sur PVC
2. **install-python-deps** - Installe `hass-web-proxy-lib==0.0.7`
3. **restore-config** - Restore depuis S3/MinIO via rclone
4. **restore-db** - Restore DB via Litestream
5. **config-init** - Copie config par défaut si absente

### Backup
- **Litestream** : Réplication continue DB vers S3
- **rclone** : Sync config toutes les 60s vers MinIO
- **Exclusions** : *.log, tts/**, backups/**, tmp_backups/**

---

## 📝 Recommandations

1. **Cleanup DB** - Lancer purge recorder
2. **Migrer secrets** - Tokens vers Infisical/secrets
3. **Nettoyer devices** - Supprimer ESPHome tests
4. **Vérifier VRs** - 8 volets Rolux disabled (réseau différent 192.168.132.x)
5. **Maintenir HACS** - Mettre à jour régulièrement

---

*Rapport généré le 2026-02-22 via audit kubectl exec*
