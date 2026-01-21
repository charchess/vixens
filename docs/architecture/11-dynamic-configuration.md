# 11. Configuration Dynamique du Système

Le framework hAIrem est piloté par un fichier central `system.yaml` permettant de modifier le comportement du noyau sans redémarrage.

## 11.1 Structure du fichier `config/system.yaml`

```yaml
# --- Paramètres du H-Core ---
core:
  log_level: "INFO"
  default_llm: "gpt-4-turbo"
  lite_llm_endpoint: "http://ollama:11434"

# --- Gouverneur de Sécurité ---
safety_governor:
  p0_mode:
    inhibit_entropy: true
    force_visual_preset: "emergency_red"
    force_audio_preset: "drone_alert"
  p1_mode:
    inhibit_entropy: false
    visual_overlay: "orange_glow"

# --- Paramètres de l'Entropie (Dieu) ---
chaos_settings:
  enabled: true
  global_probability: 0.05
  banned_scopes: ["security", "emergency", "health"]
```
