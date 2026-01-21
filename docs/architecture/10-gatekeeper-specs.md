# 10. Gatekeeper & Plugin Loader Specifications

Le Gatekeeper est le module du H-Core responsable du cycle de vie des agents (Hotplug). Il garantit la stabilité et la sécurité du système.

## 10.1 Protocole de Validation (4 Étapes)

### Étape 1 : Validation du Manifeste (Structure)
- **Outil :** Pydantic `AgentManifest` model.
- **Vérification :** Champs obligatoires (`id`, `uuid`, `role`, `name`, et `cognition.primary_scope`).
- **Action en cas d'erreur :** Arrêt immédiat du chargement, notification Redis `system:error`.

### Étape 2 : Intégrité des Assets
- **Extensions autorisées :** `.webp`, `.png`, `.svg`.
- **Mapping :** Vérifier que chaque fichier listé dans `pose_mapping` existe dans `/assets/`.
- **Obligation :** La pose `idle` est critique et doit être présente.

### Étape 3 : Analyse Statique de la Logique
- **Méthode :** `ast.parse(logic.py)` sans exécution.
- **Interdictions :** `import os`, `subprocess`, `sys`, `shutil`, `socket`.
- **Signature :** Toutes les fonctions dans `capabilities` doivent être `async def (ctx, args)`.

### Étape 4 : Heartbeat & Hot-reload
- **Hash :** Calcul d'un hash SHA-256 du dossier agent.
- **Monitoring :** Utilisation de `watchdog` (Inotify) sur le dossier `/agents/`.
- **Reload :** Si le hash change, l'agent est déchargé, re-validé et re-chargé.
