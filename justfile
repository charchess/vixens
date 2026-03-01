#!/usr/bin/env just --justfile
# workflow.just - State Machine Workflow pour Gemini (Coding Agent)
# Phases séquentielles avec garde-fous et processus GitOps complet

set shell := ["bash", "-uc"]
# Variables de chemin
scripts_path := "scripts"
report_path := "docs/reports"

JUST := "just"

# ============================================
# PHASES DU WORKFLOW (State Machine)
# ============================================
# 0. SELECTION      - Sélectionner la tâche
# 1. PREREQS        - Vérifier prérequis (PVC/RWO, toleration)
# 2. DOCUMENTATION  - Charger documentation de l'application
# 3. IMPLEMENTATION - Coder (Serena/Archon) - SCOPE LIMITÉ
# 4. DEPLOYMENT     - Commit + Push + Wait ArgoCD sync ⭐ CRITIQUE
# 5. VALIDATION     - Valider APRÈS déploiement
# 6. FINALIZATION   - Documentation + Close + Instructions promotion

# ============================================
# COMMANDE PRINCIPALE : Reprendre où on en est
# ============================================
default:
    @{{JUST}} resume

resume:
    #!/usr/bin/env python3
    import subprocess, json, sys, re, os

    def get_current_agent():
        """Détecter l'agent actuel de manière intelligente"""
        # 1. Env var explicite (priorité)
        agent = os.getenv("AGENT_NAME")
        if agent:
            return agent
        
        # 2. Détection via Claude Code context
        if os.path.exists("/.claude") or os.path.exists(".claude"):
            return "claude"
        
        # 3. Default fallback
        return "coding-agent"

    def filter_tasks_for_agent(all_tasks, current_agent):
        """
        Un agent peut prendre:
        - Ses tâches spécifiques (assignee = agent_name)
        - Les tâches génériques (assignee = 'coding-agent')
        - Les tâches non assignées (assignee = null/empty)
        """
        return [t for t in all_tasks 
                if t.get('assignee') in [current_agent, 'coding-agent', None, '']]

    # Détecter l'agent actuel
    current_agent = get_current_agent()
    
    # Récupérer toutes les tâches en cours
    result = subprocess.run(
        ["bd", "list", "--status", "in_progress", "--json"],
        capture_output=True, text=True
    )
    
    # Filtrer pour l'agent actuel
    if result.returncode == 0:
        all_tasks = json.loads(result.stdout)
        tasks = filter_tasks_for_agent(all_tasks, current_agent)
        result = type('obj', (object,), {'returncode': 0, 'stdout': json.dumps(tasks), 'stderr': ''})()
    else:
        tasks = []

    if result.returncode != 0:
        print("❌ Erreur bd:", result.stderr)
        sys.exit(1)
    
    # tasks déjà filtrées ci-dessus

    if not tasks:
        print("📋 AUCUNE TÂCHE EN COURS.")
        print("\n🔍 Tâches ouvertes disponibles:")
        subprocess.run(["bd", "list", "--status", "open", "--limit", "10"])
        print("\n💡 Commande: just start <task_id>")
        sys.exit(0)

    task = tasks[0]
    task_id = task['id']
    title = task['title']
    notes = task.get('notes', '')

    # Détecter la phase actuelle
    phase_match = re.search(r'PHASE:(\d+)', notes)
    current_phase = int(phase_match.group(1)) if phase_match else 0

    print(f"🔥 TÂCHE EN COURS: {task_id}")
    print(f"📌 Titre: {title}")
    print(f"📍 Phase actuelle: {current_phase}")
    print()

    # Extraire app name pour contexte
    app_match = re.search(r'\(([^)]+)\)', title)
    app_name = app_match.group(1) if app_match else "N/A"

    # Afficher les instructions de la phase
    phases = {
        0: {
            "name": "SELECTION",
            "todo": [
                "Lire le titre et la description de la tâche",
                "Identifier l'application ciblée (entre parenthèses)",
                "Comprendre l'objectif de la tâche"
            ],
            "forbidden": [
                "❌ NE PAS commencer à coder",
                "❌ NE PAS toucher aux fichiers"
            ],
            "next_cmd": f"just next {task_id}"
        },
        1: {
            "name": "PREREQS",
            "todo": [
                "Vérifier si PVC RWO → noter 'strategy: Recreate' requis",
                "Vérifier si controlplane → noter 'tolerations' requis",
                "Identifier les dépendances techniques"
            ],
            "forbidden": [
                "❌ NE PAS modifier de fichiers",
                "❌ NE PAS coder"
            ],
            "next_cmd": f"just next {task_id}"
        },
        2: {
            "name": "DOCUMENTATION",
            "todo": [
                "Lire docs/applications/<category>/<app>.md",
                "Comprendre l'architecture actuelle",
                "Utiliser Archon RAG pour rechercher patterns similaires"
            ],
            "forbidden": [
                "❌ NE PAS modifier de code",
                "❌ NE PAS créer de fichiers"
            ],
            "next_cmd": f"just next {task_id}"
        },
        3: {
            "name": "IMPLEMENTATION",
            "todo": [
                "Coder UNIQUEMENT l'application ciblée",
                "Utiliser Serena pour édition de code",
                "Suivre les patterns existants (DRY)",
                "Respecter GitOps (ZERO kubectl apply direct)"
            ],
            "forbidden": [
                "❌ INTERDIT: Toucher à d'autres applications",
                "❌ INTERDIT: kubectl apply/edit/delete (GitOps only)",
                "❌ INTERDIT: Créer des duplications (DRY)",
                "❌ INTERDIT: Fermer la tâche",
                "❌ INTERDIT: Bypasser la validation",
                "❌ INTERDIT: Commit/push (phase suivante)"
            ],
            "rules": [
                "📜 GitOps: Tout passe par Git → ArgoCD sync",
                "📜 DRY: Réutiliser apps/_shared/ si applicable",
                "📜 Scope: UNIQUEMENT l'app dans le titre de la tâche",
                "📜 NO COMMIT: Attendre phase DEPLOYMENT"
            ],
            "next_cmd": f"just next {task_id}"
        },
        4: {
            "name": "DEPLOYMENT",
            "todo": [
                f"Vérifier branch actuelle: git branch --show-current (doit être 'main')",
                "Commit les changements: git add + git commit -m '...'",
                "Push vers main: git push origin main (ou feature branch + PR)",
                f"Attendre ArgoCD sync: just wait-argocd {app_name}",
                "Vérifier status: Health=Healthy, Sync=Synced"
            ],
            "forbidden": [
                "❌ INTERDIT: Push direct vers main pour features majeures (utiliser PR)",
                "❌ INTERDIT: Créer des tags manuellement (sauf prod promotion)",
                "❌ INTERDIT: Avancer avant ArgoCD Synced+Healthy",
                "❌ INTERDIT: kubectl apply/edit direct"
            ],
            "rules": [
                "📜 Branch: Toujours main pour développement (trunk-based)",
                "📜 GitOps: git push → ArgoCD auto-sync dev",
                "📜 Attente: ArgoCD peut prendre 1-3 minutes",
                "📜 Vérification: Synced + Healthy obligatoires"
            ],
            "next_cmd": f"just next {task_id}"
        },
        5: {
            "name": "VALIDATION",
            "todo": [
                f"Validation APRÈS déploiement: python3 scripts/validate.py {app_name} dev",
                "Vérifier que la validation passe (exit code 0)",
                "Corriger les erreurs si échec (retour phase 3)"
            ],
            "forbidden": [
                "❌ INTERDIT: Valider AVANT ArgoCD sync",
                "❌ INTERDIT: Avancer sans validation réussie",
                "❌ INTERDIT: Fermer la tâche manuellement"
            ],
            "rules": [
                "📜 Validation: Teste l'app DÉPLOYÉE sur cluster dev",
                "📜 Échec: Retour phase 3 (just reset-phase)",
                "📜 Succès: Marqué dans notes Beads"
            ],
            "next_cmd": f"just next {task_id}"
        },
        6: {
            "name": "FINALIZATION",
            "todo": [
                "Mettre à jour docs/applications/<category>/<app>.md",
                "Mettre à jour docs/STATUS.md si nécessaire",
                "Committer les changements de documentation",
                "Vérifier git push réussi"
            ],
            "forbidden": [],
            "promotion": [
                "🎯 PROMOTION VERS PRODUCTION (ADR-017):",
                "  1. Validé sur dev ✅",
                "  2. Lancer workflow de promotion:",
                "     → gh workflow run promote-prod.yaml -f version=vX.Y.Z",
                "     → Déplace le tag prod-stable",
                "     → ArgoCD sync automatique sur prod cluster",
                "  3. Ne JAMAIS créer de tag manuellement"
            ],
            "next_cmd": f"just close {task_id}"
        }
    }

    phase_info = phases.get(current_phase, phases[0])

    print(f"🎯 PHASE {current_phase}: {phase_info['name']}")
    print()
    print("✅ À FAIRE:")
    for item in phase_info['todo']:
        print(f"   • {item}")

    if phase_info.get('forbidden'):
        print()
        print("🚫 INTERDICTIONS:")
        for item in phase_info['forbidden']:
            print(f"   {item}")

    if phase_info.get('rules'):
        print()
        print("📜 RÈGLES CRITIQUES:")
        for rule in phase_info['rules']:
            print(f"   {rule}")

    if phase_info.get('promotion'):
        print()
        for line in phase_info['promotion']:
            print(line)

    print()
    print(f"➡️  Commande suivante: {phase_info['next_cmd']}")

# ============================================
# DÉMARRER UNE TÂCHE (Phase 0)
# ============================================
start task_id:
    #!/usr/bin/env python3
    import subprocess, json, re, sys, os

    def get_current_agent():
        """Détecter l'agent actuel de manière intelligente"""
        # 1. Env var explicite (priorité)
        agent = os.getenv("AGENT_NAME")
        if agent:
            return agent
        
        # 2. Détection via Claude Code context
        if os.path.exists("/.claude") or os.path.exists(".claude"):
            return "claude"
        
        # 3. Default fallback
        return "coding-agent"

    # Vérifier qu'on est sur main branch
    branch_result = subprocess.run(
        ["git", "branch", "--show-current"],
        capture_output=True, text=True
    )
    current_branch = branch_result.stdout.strip()

    if current_branch != "main":
        print(f"❌ BLOQUÉ: Branch actuelle '{current_branch}', attendu 'main'")
        print("   Le workflow requiert d'être sur main pour démarrer")
        print("   💡 Solution: git checkout main")
        sys.exit(1)

    # Récupérer les infos de la tâche pour vérifier l'assignee actuel
    task_result = subprocess.run(
        ["bd", "show", "{{task_id}}", "--json"],
        capture_output=True, text=True
    )
    
    # Déterminer l'assignee à utiliser
    if task_result.returncode == 0:
        task_data = json.loads(task_result.stdout)
        # task_data est un array, prendre le premier élément
        task_info = task_data[0] if isinstance(task_data, list) else task_data
        current_assignee = task_info.get('assignee')
        
        # Ne définir l'assignee que s'il est vide/null
        if not current_assignee or current_assignee in ['', 'null']:
            assignee = get_current_agent()
            print(f"📝 Attribution à: {assignee}")
        else:
            assignee = current_assignee
            print(f"📝 Assignee préservé: {assignee}")
    else:
        # Fallback si on ne peut pas lire la tâche
        assignee = get_current_agent()
        print(f"📝 Attribution par défaut à: {assignee}")

    # Mettre à jour le statut et initialiser la phase (préserve assignee)
    subprocess.run([
        "bd", "update", "{{task_id}}",
        "--status", "in_progress",
        "--assignee", assignee,
        "--notes", f"PHASE:0 - Tâche démarrée (branch: {current_branch}, agent: {assignee})"
    ])

    print("✅ Tâche démarrée en Phase 0: SELECTION")
    print("💡 Lancer: just resume")

# ============================================
# AVANCER À LA PHASE SUIVANTE (avec validation)
# ============================================
next task_id:
    #!/usr/bin/env python3
    import subprocess, json, re, sys
    from datetime import datetime

    # Récupérer la tâche
    result = subprocess.run(
        ["bd", "show", "{{task_id}}", "--json"],
        capture_output=True, text=True, check=True
    )

    tasks = json.loads(result.stdout)
    if not tasks:
        print("❌ Tâche non trouvée")
        sys.exit(1)

    task = tasks[0]
    title = task['title']
    notes = task.get('notes', '')

    # Détecter phase actuelle
    phase_match = re.search(r'PHASE:(\d+)', notes)
    current_phase = int(phase_match.group(1)) if phase_match else 0

    # Extraire nom de l'app
    app_match = re.search(r'\(([^)]+)\)', title)
    app_name = app_match.group(1) if app_match else None

    print(f"📍 Phase actuelle: {current_phase}")

    # VALIDATION SELON LA PHASE
    if current_phase == 1:
        # Phase PREREQS: vérifier notes prérequis
        print("✅ Phase PREREQS complétée")
        if "PVC" in title and "RWO" in title:
            if "strategy: Recreate" not in notes:
                subprocess.run([
                    "bd", "update", "{{task_id}}",
                    "--notes", f"{notes}\nREQUIS: strategy: Recreate (PVC RWO)"
                ])

    elif current_phase == 2:
        # Phase DOCUMENTATION: vérifier que doc existe (non bloquant)
        print("✅ Phase DOCUMENTATION complétée")

    elif current_phase == 3:
        # Phase IMPLEMENTATION: vérifier qu'il y a des changements
        git_result = subprocess.run(
            ["git", "status", "--porcelain"],
            capture_output=True, text=True
        )
        if not git_result.stdout.strip():
            print("❌ BLOQUÉ: Aucun changement détecté")
            print("   L'implémentation (Phase 3) nécessite des modifications de code")
            print("   💡 Solution:")
            print("      - Vérifier que les changements sont bien effectués")
            print("      - Si l'implémentation est complète: git add .")
            print("      - Sinon: continuer le développement")
            sys.exit(1)
        print("✅ Phase IMPLEMENTATION complétée")

    elif current_phase == 4:
        # Phase DEPLOYMENT: vérifier commit, push, ArgoCD sync
        if not app_name:
            print("❌ BLOQUÉ: Impossible de déployer sans nom d'application")
            sys.exit(1)

        # Vérifier branch
        branch_result = subprocess.run(
            ["git", "branch", "--show-current"],
            capture_output=True, text=True
        )
        current_branch = branch_result.stdout.strip()
        if current_branch != "main":
            print(f"⚠️  WARNING: Sur branch '{current_branch}', attendu 'main'")

        # Vérifier qu'il n'y a plus de changements non committés
        git_status = subprocess.run(
            ["git", "status", "--porcelain"],
            capture_output=True, text=True
        )
        if git_status.stdout.strip():
            print("⚠️  Changements non committés détectés:")
            print(git_status.stdout)
            print("   Assurez-vous d'avoir commit+push tous les changements")
            sys.exit(1)

        # Vérifier ArgoCD sync status
        print(f"🔍 Vérification ArgoCD pour: {app_name}")

        # Détecter si l'app est hibernée (commentée dans kustomization.yaml)
        was_hibernated = False
        kustomization_path = f"argocd/overlays/dev/kustomization.yaml"
        try:
            with open(kustomization_path, 'r') as f:
                content = f.read()
                # Chercher si l'app est commentée
                if f"# - apps/{app_name}.yaml" in content:
                    print(f"❌ BLOQUÉ: Application '{app_name}' est HIBERNÉE dans dev")
                    print(f"   (Commentée dans {kustomization_path})")
                    print()
                    print("   💡 Solution - Décommenter MANUELLEMENT pour tester:")
                    print(f"      1. Éditer {kustomization_path}")
                    print(f"      2. Décommenter: # - apps/{app_name}.yaml → - apps/{app_name}.yaml")
                    print("      3. Commit et push")
                    print("      4. Attendre ArgoCD sync (~30s)")
                    print("      5. Reprendre workflow: just next {{task_id}}")
                    print()
                    print("   ⚠️  IMPORTANT: Re-hiberner après test!")
                    sys.exit(1)
        except FileNotFoundError:
            pass  # Fichier pas trouvé, continuer la vérification normale

        # Vérification ArgoCD (toujours effectuée maintenant)
        argocd_result = subprocess.run(
            ["kubectl", "-n", "argocd", "get", "application", app_name, "-o", "json"],
            capture_output=True, text=True
        )

        if argocd_result.returncode != 0:
            print(f"⚠️  Application ArgoCD '{app_name}' non trouvée")
            print("   Vérifiez le nom de l'application dans ArgoCD")
            print("   💡 Si l'app est prod-only, c'est normal en dev")
        else:
            # App exists, check its status
            try:
                import json as json_module
                app_status = json_module.loads(argocd_result.stdout)
                sync_status = app_status.get('status', {}).get('sync', {}).get('status', 'Unknown')
                health_status = app_status.get('status', {}).get('health', {}).get('status', 'Unknown')

                print(f"   Sync Status: {sync_status}")
                print(f"   Health Status: {health_status}")

                if sync_status != 'Synced':
                    print(f"   ❌ BLOQUÉ: Application pas Synced (status: {sync_status})")
                    print(f"   💡 Solution: Attendre la synchronisation")
                    print(f"      just wait-argocd {app_name}")
                    print("   Ou vérifier manuellement:")
                    print(f"      kubectl -n argocd get application {app_name}")
                    sys.exit(1)

                if health_status not in ['Healthy', 'Progressing']:
                    print(f"   ❌ BLOQUÉ: Application pas Healthy (status: {health_status})")
                    print("   💡 Solution: Diagnostiquer le problème")
                    print(f"      kubectl -n argocd describe application {app_name}")
                    print(f"      kubectl -n <namespace> get pods")
                    print("   Corriger les erreurs avant de continuer")
                    sys.exit(1)

                print("   ✅ ArgoCD status OK")
            except Exception as e:
                print(f"   ⚠️  Erreur parsing status ArgoCD: {e}")

        # Marquer le déploiement
        notes = f"{notes}\nDEPLOYED: {datetime.now().isoformat()} (branch: main)"
        subprocess.run([
            "bd", "update", "{{task_id}}",
            "--notes", notes
        ])
        print("✅ Phase DEPLOYMENT complétée")

    elif current_phase == 5:
        # Phase VALIDATION: BLOQUER si validation non passée
        if not app_name:
            print("❌ BLOQUÉ: Impossible de valider sans nom d'application")
            sys.exit(1)

        print(f"🎭 VALIDATION OBLIGATOIRE (post-deployment): {app_name}")
        val_result = subprocess.run(
            ["python3", "scripts/validation/validate.py", app_name, "dev"],
            capture_output=True, text=True
        )

        if val_result.returncode != 0:
            print(f"❌ VALIDATION ÉCHOUÉE:\n{val_result.stdout}\n{val_result.stderr}")
            subprocess.run([
                "bd", "update", "{{task_id}}",
                "--notes", f"{notes}\nVALIDATION FAIL: {val_result.stdout[:100]} {val_result.stderr[:100]}"
            ])
            print("\n💡 Pour corriger: just reset-phase {{task_id}} 3")
            sys.exit(1)

        print("✅ VALIDATION RÉUSSIE")
        # Marquer la validation dans les notes
        notes = f"{notes}\nVALIDATION OK: {datetime.now().isoformat()}"
        subprocess.run([
            "bd", "update", "{{task_id}}",
            "--notes", notes
        ])

    # AVANCER À LA PHASE SUIVANTE
    next_phase = current_phase + 1
    if next_phase > 6:
        print("✅ Toutes les phases complétées!")
        print("💡 Lancer: just close {{task_id}}")
        sys.exit(0)

    # Mettre à jour la phase
    new_notes = re.sub(r'PHASE:\d+', f'PHASE:{next_phase}', notes)
    if 'PHASE:' not in new_notes:
        new_notes = f"PHASE:{next_phase}\n{notes}"

    subprocess.run([
        "bd", "update", "{{task_id}}",
        "--notes", new_notes
    ])

    phase_names = ["SELECTION", "PREREQS", "DOCUMENTATION", "IMPLEMENTATION", "DEPLOYMENT", "VALIDATION", "FINALIZATION"]
    print(f"➡️  Avancé à Phase {next_phase}: {phase_names[next_phase]}")
    print("💡 Lancer: just resume")

# ============================================
# FERMER LA TÂCHE (avec vérification finale)
# ============================================
close task_id:
    #!/usr/bin/env python3
    import subprocess, json, re, sys

    # Récupérer la tâche
    result = subprocess.run(
        ["bd", "show", "{{task_id}}", "--json"],
        capture_output=True, text=True, check=True
    )

    tasks = json.loads(result.stdout)
    if not tasks:
        print("❌ Tâche non trouvée")
        sys.exit(1)

    task = tasks[0]
    notes = task.get('notes', '')

    # Vérifier phase 6 atteinte
    phase_match = re.search(r'PHASE:(\d+)', notes)
    current_phase = int(phase_match.group(1)) if phase_match else 0

    if current_phase < 6:
        print(f"❌ BLOQUÉ: Phase actuelle {current_phase}, phase 6 requise")
        print("💡 Lancer: just next {{task_id}} pour avancer")
        sys.exit(1)

    # Vérifier validation présente
    if "VALIDATION OK" not in notes:
        print("❌ BLOQUÉ: Validation obligatoire avant fermeture")
        print("💡 Retourner en phase 5: just reset-phase {{task_id}} 5")
        sys.exit(1)

    # Vérifier déploiement présent
    if "DEPLOYED" not in notes:
        print("❌ BLOQUÉ: Déploiement obligatoire avant fermeture")
        print("💡 Retourner en phase 4: just reset-phase {{task_id}} 4")
        sys.exit(1)

    # Vérifier si l'app était hibernée → BLOQUER pour action manuelle
    if "WAS_HIBERNATED:" in notes:
        # Extraire le nom de l'app des notes
        hibernated_match = re.search(r'WAS_HIBERNATED: (\w+)', notes)
        if hibernated_match:
            app_name = hibernated_match.group(1)
            print()
            print(f"❌ BLOQUÉ: Application '{app_name}' était HIBERNÉE avant test")
            print()
            print("   💡 Solution: Re-hiberner MANUELLEMENT avant de fermer")
            print("      1. Éditer argocd/overlays/dev/kustomization.yaml")
            print(f"      2. Re-commenter: - apps/{app_name}.yaml → # - apps/{app_name}.yaml")
            print("      3. Commit: git add + git commit -m 'chore: re-hibernate...'")
            print("      4. Push: git push")
            print("      5. Reprendre: just close {{task_id}}")
            print()
            sys.exit(1)

    # Afficher checklist finale
    print("📋 CHECKLIST FINALE:")
    print("   [✓] Code déployé sur dev (ArgoCD synced from main HEAD)")
    print("   [✓] Validation réussie")
    print("   [ ] Documentation à jour (docs/applications/<category>/<app>.md)")
    print("   [ ] STATUS.md à jour si nécessaire")
    print("   [ ] Changements de doc committés + pushés")
    print()
    print("🎯 PROMOTION PRODUCTION (ADR-017):")
    print("   Pour déployer en production:")
    print("   gh workflow run promote-prod.yaml -f version=vX.Y.Z")
    print()

    # Vérification finale sans interaction
    print("✅ Vérifications automatiques complètes:")
    print("   [✓] Phase 6 atteinte")
    print("   [✓] Validation OK présente")
    print("   [✓] Déploiement présent")
    print()
    print("⚠️  RAPPEL: Vérifier que la documentation est à jour")
    print("   - docs/applications/<category>/<app>.md")
    print("   - docs/STATUS.md (si nécessaire)")
    print()

    # Fermer la tâche
    subprocess.run([
        "bd", "close", "{{task_id}}"
    ])

    print("✅ Tâche fermée avec succès!")
    print("💡 Prochaine: just resume")

# ============================================
# ATTENDRE ARGOCD SYNC (Helper)
# ============================================
wait-argocd app_name:
    #!/usr/bin/env bash
    set -euo pipefail

    echo "⏳ Attente ArgoCD sync pour: {{app_name}}"
    echo "   (timeout: 5 minutes)"

    # Vérifier que l'app existe
    if ! kubectl -n argocd get application {{app_name}} &>/dev/null; then
        echo "❌ Application '{{app_name}}' non trouvée dans ArgoCD"
        echo "   Applications disponibles:"
        kubectl -n argocd get applications -o name | sed 's|application.argoproj.io/||'
        exit 1
    fi

    # Attendre Synced
    echo "   Attente Sync Status = Synced..."
    timeout 300 bash -c '
        while true; do
            STATUS=$(kubectl -n argocd get application {{app_name}} -o jsonpath='\''{.status.sync.status}'\'' 2>/dev/null || echo "Unknown")
            echo "   → Current status: $STATUS"
            if [ "$STATUS" = "Synced" ]; then
                break
            fi
            sleep 5
        done
    ' || {
        echo "❌ Timeout: ArgoCD n'a pas sync en 5 minutes"
        echo "   Vérifier: kubectl -n argocd get application {{app_name}}"
        exit 1
    }

    # Attendre Healthy
    echo "   Attente Health Status = Healthy..."
    timeout 120 bash -c '
        while true; do
            HEALTH=$(kubectl -n argocd get application {{app_name}} -o jsonpath='\''{.status.health.status}'\'' 2>/dev/null || echo "Unknown")
            echo "   → Current health: $HEALTH"
            if [ "$HEALTH" = "Healthy" ]; then
                break
            fi
            if [ "$HEALTH" = "Degraded" ]; then
                echo "   ⚠️  Application Degraded, arrêt de l'\''attente"
                exit 1
            fi
            sleep 5
        done
    ' || {
        echo "⚠️  Warning: Health status non Healthy"
        echo "   Continuer manuellement si c'\''est attendu"
        exit 1
    }

    echo "✅ ArgoCD sync complété: {{app_name}} est Synced + Healthy"

# ============================================
# RESET PHASE (Debug / Correction)
# ============================================
reset-phase task_id phase:
    #!/usr/bin/env python3
    import subprocess, json, re, sys

    phase_num = int("{{phase}}")
    if phase_num < 0 or phase_num > 6:
        print("❌ Phase invalide (0-6)")
        sys.exit(1)

    # Récupérer la tâche
    result = subprocess.run(
        ["bd", "show", "{{task_id}}", "--json"],
        capture_output=True, text=True, check=True
    )

    tasks = json.loads(result.stdout)
    if not tasks:
        print("❌ Tâche non trouvée")
        sys.exit(1)

    task = tasks[0]
    notes = task.get('notes', '')

    # Mettre à jour la phase
    new_notes = re.sub(r'PHASE:\d+', f'PHASE:{phase_num}', notes)
    if 'PHASE:' not in new_notes:
        new_notes = f"PHASE:{phase_num}\n{notes}"

    subprocess.run([
        "bd", "update", "{{task_id}}",
        "--notes", new_notes
    ])

    phase_names = ["SELECTION", "PREREQS", "DOCUMENTATION", "IMPLEMENTATION", "DEPLOYMENT", "VALIDATION", "FINALIZATION"]
    print(f"🔄 Phase réinitialisée à {phase_num}: {phase_names[phase_num]}")
    print("💡 Lancer: just resume")

# ============================================
# PROMOTION PRODUCTION (Instructions)
# ============================================
promote-prod:
    @echo "🎯 PROCESSUS DE PROMOTION VERS PRODUCTION (ADR-017)"
    @echo ""
    @echo "📋 Prérequis:"
    @echo "   ✅ Changements validés sur dev cluster"
    @echo "   ✅ Tâche Beads fermée"
    @echo ""
    @echo "🔄 Étapes de promotion:"
    @echo "   1. Déclencher le workflow GitHub:"
    @echo "      gh workflow run promote-prod.yaml -f version=vX.Y.Z"
    @echo ""
    @echo "   2. Le workflow va:"
    @echo "      - Créer un tag prod-vX.Y.Z"
    @echo "      - Déplacer le tag prod-stable vers ce commit"
    @echo ""
    @echo "   3. Vérifier déploiement prod:"
    @echo "      kubectl -n argocd get applications  # cluster prod"
    @echo "      just wait-argocd <app_name>  # avec KUBECONFIG prod"
    @echo ""
    @echo "⚠️  RÈGLES:"
    @echo "   • JAMAIS créer de tag manuellement"
    @echo "   • Promotion via GitHub Actions uniquement"
    @echo ""
    @echo "💡 OU utilisez: just SendToProd (automatisé)"

# ============================================
# PROMOTION PRODUCTION AUTOMATISÉE
# ============================================
SendToProd version:
    #!/usr/bin/env bash
    set -euo pipefail

    VERSION="{{version}}"
    # Remove 'v' prefix if present
    VERSION=${VERSION#v}

    echo "🚀 PROMOTION VERS PRODUCTION - v${VERSION}"
    echo ""

    # 1. Vérifier branch = main
    echo "📍 Étape 1/8: Vérification branch..."
    CURRENT_BRANCH=$(git branch --show-current)
    if [ "$CURRENT_BRANCH" != "main" ]; then
        echo "❌ Erreur: Branch actuelle '$CURRENT_BRANCH', attendu 'main'"
        echo "💡 Solution: git checkout main"
        exit 1
    fi
    echo "   ✅ Branch: main"

    # 2. Vérifier git status propre
    echo "📍 Étape 2/8: Vérification working tree..."
    if [ -n "$(git status --porcelain)" ]; then
        echo "❌ Erreur: Working tree non propre"
        echo "💡 Solution: git add . && git commit -m '...' && git push"
        git status
        exit 1
    fi
    echo "   ✅ Working tree propre"

    # 3. Pull latest
    echo "📍 Étape 3/8: Pull des derniers changements..."
    git pull origin main --ff-only || {
        echo "❌ Erreur: Impossible de pull (fast-forward)"
        echo "💡 Solution: Résoudre les conflits manuellement"
        exit 1
    }
    echo "   ✅ Up to date avec remote"

    # 4. Créer tag dev-vX.Y.Z
    echo "📍 Étape 4/8: Création tag dev-v${VERSION}..."
    DEV_TAG="dev-v${VERSION}"

    # Vérifier si le tag existe déjà
    if git rev-parse "$DEV_TAG" >/dev/null 2>&1; then
        echo "⚠️  Tag $DEV_TAG existe déjà"
        read -p "   Supprimer et recréer? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "❌ Annulé"
            exit 1
        fi
        git tag -d "$DEV_TAG"
        git push origin ":refs/tags/$DEV_TAG" 2>/dev/null || true
    fi

    git tag -a "$DEV_TAG" -m "Dev release v${VERSION} - ready for prod promotion"
    echo "   ✅ Tag créé: $DEV_TAG"

    # 5. Push tag
    echo "📍 Étape 5/8: Push du tag..."
    git push origin "$DEV_TAG" || {
        echo "❌ Erreur: Impossible de push le tag"
        echo "💡 Rollback: git tag -d $DEV_TAG"
        git tag -d "$DEV_TAG"
        exit 1
    }
    echo "   ✅ Tag pushé: $DEV_TAG"

    # 6. Déclencher workflow GitHub
    echo "📍 Étape 6/8: Déclenchement workflow GitHub..."
    if ! command -v gh &> /dev/null; then
        echo "❌ Erreur: gh CLI non installé"
        echo "💡 Solution: brew install gh (ou équivalent)"
        exit 1
    fi

    gh workflow run promote-prod.yaml -f version="v${VERSION}" || {
        echo "❌ Erreur: Impossible de déclencher le workflow"
        echo "💡 Vérifier: gh auth status"
        exit 1
    }
    echo "   ✅ Workflow déclenché"

    # 7. Attendre le workflow (timeout 10 min)
    echo "📍 Étape 7/8: Attente du workflow (timeout: 10 min)..."
    TIMEOUT=600  # 10 minutes
    ELAPSED=0
    INTERVAL=10

    while [ $ELAPSED -lt $TIMEOUT ]; do
        sleep $INTERVAL
        ELAPSED=$((ELAPSED + INTERVAL))

        # Vérifier si le workflow est terminé
        STATUS=$(gh run list --workflow=promote-prod.yaml --limit=1 --json status --jq '.[0].status')

        if [ "$STATUS" = "completed" ]; then
            CONCLUSION=$(gh run list --workflow=promote-prod.yaml --limit=1 --json conclusion --jq '.[0].conclusion')
            if [ "$CONCLUSION" = "success" ]; then
                echo "   ✅ Workflow terminé avec succès"
                break
            else
                echo "   ❌ Workflow échoué: $CONCLUSION"
                echo "   💡 Voir les logs: gh run view"
                exit 1
            fi
        fi

        echo "   ⏳ Workflow en cours... ($ELAPSED/$TIMEOUT s)"
    done

    if [ $ELAPSED -ge $TIMEOUT ]; then
        echo "   ❌ Timeout: Le workflow a pris plus de 10 minutes"
        echo "   💡 Vérifier manuellement: gh run view"
        exit 1
    fi

    # 8. Vérification finale
    echo "📍 Étape 8/8: Vérification tags créés..."
    git fetch --tags

    PROD_TAG="prod-v${VERSION}"
    if git rev-parse "$PROD_TAG" >/dev/null 2>&1; then
        echo "   ✅ Tag prod créé: $PROD_TAG"
    else
        echo "   ⚠️  Tag prod non trouvé: $PROD_TAG"
    fi

    if git rev-parse "prod-stable" >/dev/null 2>&1; then
        echo "   ✅ Tag prod-stable mis à jour"
    else
        echo "   ⚠️  Tag prod-stable non trouvé"
    fi

    echo ""
    echo "✅ PROMOTION RÉUSSIE!"
    echo ""
    echo "📊 Prochaines étapes:"
    echo "   1. Vérifier ArgoCD prod:"
    echo "      export KUBECONFIG=/root/vixens/.secrets/prod/kubeconfig-prod"
    echo "      kubectl -n argocd get applications"
    echo ""
    echo "   2. Sauvegarder config fonctionnelle:"
    echo "      git tag prod-working prod-stable"
    echo "      git push origin prod-working"
    echo ""
    echo "🎯 Version déployée en production: v${VERSION}"

# ============================================
# AUTOMATION DES RAPPORTS (Consolidé)
# ============================================

# Générer TOUS les rapports (remplace reports + lint-report + vpa.sh)
reports:
    #!/usr/bin/env bash
    set -euo pipefail

    echo "📊 GÉNÉRATION COMPLÈTE DES RAPPORTS VIXENS"
    echo "=========================================="
    echo ""

    # Créer trash/ pour fichiers obsolètes
    mkdir -p docs/reports/trash

    # === PHASE 1: CLUSTER STATE (VPA + Resources) ===
    echo "🔍 Phase 1/7: État cluster (VPA + Resources)"
    echo "--------------------------------------------"

    # DEV cluster
    if [ -f "/root/vixens/.secrets/dev/kubeconfig-dev" ]; then
        echo "  → Dev cluster..."
        export KUBECONFIG="/root/vixens/.secrets/dev/kubeconfig-dev"

        python3 scripts/reports/generate_actual_state_vpa.py \
            --env dev \
            --output docs/reports/STATE-ACTUAL-dev.md \
            --json-output docs/reports/STATE-dev.json

        echo "  ✅ STATE-ACTUAL-dev.md + STATE-dev.json"
    else
        echo "  ⚠️  Skip dev (kubeconfig non trouvé)"
    fi

    # PROD cluster
    if [ -f "/root/vixens/.secrets/prod/kubeconfig-prod" ]; then
        echo "  → Prod cluster..."
        export KUBECONFIG="/root/vixens/.secrets/prod/kubeconfig-prod"

        python3 scripts/reports/generate_actual_state_vpa.py \
            --env prod \
            --output docs/reports/STATE-ACTUAL-prod.md \
            --json-output docs/reports/STATE-prod.json

        # Legacy compatibility: copie prod → STATE-ACTUAL.md
        cp docs/reports/STATE-ACTUAL-prod.md docs/reports/STATE-ACTUAL.md
        echo "  ✅ STATE-ACTUAL-prod.md + STATE-prod.json + STATE-ACTUAL.md (legacy)"
    else
        echo "  ⚠️  Skip prod (kubeconfig non trouvé)"
    fi

    echo ""

    # === PHASE 2: APPLICATION VERSIONS ===
    echo "📦 Phase 2/7: Inventaire versions"
    echo "--------------------------------------------"

    if [ -f "/root/vixens/.secrets/prod/kubeconfig-prod" ]; then
        export KUBECONFIG="/root/vixens/.secrets/prod/kubeconfig-prod"
        python3 scripts/reports/generate_app_versions.py \
            --output docs/reports/APP-VERSIONS.md
        echo "  ✅ APP-VERSIONS.md"
    else
        echo "  ⚠️  Skip (prod kubeconfig requis)"
    fi

    echo ""

    # === PHASE 3: LINT & QUALITY ===
    echo "🧹 Phase 3/7: Qualité code YAML"
    echo "--------------------------------------------"

    python3 scripts/reports/generate_lint_report.py \
        --paths apps argocd \
        --output docs/reports/LINT-REPORT.md \
        --fail-threshold 0 || true  # Non-bloquant

    echo "  ✅ LINT-REPORT.md"
    echo ""

    # === PHASE 4: CONFORMITY ===
    echo "📏 Phase 4/7: Conformité (Actual vs Desired)"
    echo "--------------------------------------------"

    if [ -f "docs/reports/STATE-ACTUAL-dev.md" ]; then
        python3 scripts/reports/conformity_checker.py \
            --actual docs/reports/STATE-ACTUAL-dev.md \
            --desired docs/reports/STATE-DESIRED.md \
            --output docs/reports/CONFORMITY-dev.md
        echo "  ✅ CONFORMITY-dev.md"
    fi

    if [ -f "docs/reports/STATE-ACTUAL-prod.md" ]; then
        python3 scripts/reports/conformity_checker.py \
            --actual docs/reports/STATE-ACTUAL-prod.md \
            --desired docs/reports/STATE-DESIRED.md \
            --output docs/reports/CONFORMITY-prod.md
        echo "  ✅ CONFORMITY-prod.md"
    fi

    echo ""

    # === PHASE 5: DASHBOARD CONSOLIDÉ ===
    echo "📊 Phase 5/7: Dashboard STATUS.md"
    echo "--------------------------------------------"

    if [ -f "docs/reports/STATE-dev.json" ] && [ -f "docs/reports/STATE-prod.json" ]; then
        python3 scripts/reports/generate_status_report.py \
            --dev-state docs/reports/STATE-dev.json \
            --prod-state docs/reports/STATE-prod.json \
            --dev-conformity docs/reports/CONFORMITY-dev.md \
            --prod-conformity docs/reports/CONFORMITY-prod.md \
            --output docs/reports/STATUS.md
        echo "  ✅ STATUS.md"
    else
        echo "  ⚠️  Skip (fichiers JSON manquants)"
    fi

    echo ""

    # === PHASE 6: RAPPORT CHEFFERIE ===
    echo "👔 Phase 6/7: Rapport Chefferie"
    echo "--------------------------------------------"

    if [ -f "/root/vixens/.secrets/prod/kubeconfig-prod" ]; then
        export KUBECONFIG="/root/vixens/.secrets/prod/kubeconfig-prod"
        python3 scripts/reports/generate_management_report.py \
            --output docs/reports/MANAGEMENT-REPORT.md
        echo "  ✅ MANAGEMENT-REPORT.md"
    else
        echo "  ⚠️  Skip (prod kubeconfig requis)"
    fi

    echo ""

    # === PHASE 7: CLEANUP (Fichiers obsolètes) ===
    echo "🗑️  Phase 7/7: Nettoyage fichiers obsolètes"
    echo "--------------------------------------------"

    # Déplacer fichiers obsolètes vers trash/
    TRASH_DIR="docs/reports/trash/$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$TRASH_DIR"

    # AUDIT-CONFORMITY.md → Remplacé par CONFORMITY-*.md + STATUS.md
    [ -f "docs/reports/AUDIT-CONFORMITY.md" ] && {
        mv docs/reports/AUDIT-CONFORMITY.md "$TRASH_DIR/"
        echo "  🗑️  AUDIT-CONFORMITY.md → trash/ (remplacé par CONFORMITY-*.md)"
    }

    # Rapports historiques datés (2024-*, 2025-*)
    find docs/reports/ -maxdepth 1 -name "20[0-9][0-9]-*.md" -type f | while read -r file; do
        mv "$file" "$TRASH_DIR/"
        echo "  🗑️  $(basename "$file") → trash/ (historique)"
    done

    # Fichiers JSON temporaires
    [ -f "docs/reports/STATE-dev.json" ] && rm -f docs/reports/STATE-dev.json
    [ -f "docs/reports/STATE-prod.json" ] && rm -f docs/reports/STATE-prod.json

    echo "  ✅ Cleanup terminé"
    echo ""

    # === RÉSUMÉ FINAL ===
    echo "=========================================="
    echo "✅ RAPPORTS GÉNÉRÉS AVEC SUCCÈS"
    echo "=========================================="
    echo ""
    echo "📋 Rapports vivants (Living Documents):"
    echo "   • STATE-ACTUAL-dev.md      (état dev avec VPA)"
    echo "   • STATE-ACTUAL-prod.md     (état prod avec VPA)"
    echo "   • STATE-ACTUAL.md          (prod - legacy)"
    echo "   • CONFORMITY-dev.md        (conformité dev)"
    echo "   • CONFORMITY-prod.md       (conformité prod)"
    echo "   • STATUS.md                (dashboard consolidé)"
    echo "   • LINT-REPORT.md           (qualité code)"
    echo "   • APP-VERSIONS.md          (inventaire versions)"
    echo "   • MANAGEMENT-REPORT.md     (rapport chefferie)"
    echo ""
    echo "📚 Rapports de référence (manuels):"
    echo "   • STATE-DESIRED.md         (standards cibles)"
    echo "   • STORAGE-STRATEGY.md      (stratégie storage)"
    echo ""
    echo "🗑️  Fichiers déplacés: $TRASH_DIR"
    echo ""
    echo "💡 Consulter: docs/reports/README.md"

# Voir la maturité réelle des applications du cluster
maturity:
    @echo "📊 MATURITÉ RÉELLE DES APPLICATIONS (Cluster Prod)"
    @echo "-----------------------------------------------"
    @export KUBECONFIG=.secrets/prod/kubeconfig-prod; \
    (printf "NAMESPACE APPLICATION MATURITY\n-------- ----------- --------\n" && \
    kubectl get deployment,statefulset,daemonset -A -o json | \
    jq -r '.items[] | "\(.metadata.namespace) \(.metadata.name) \(.metadata.labels["vixens.io/maturity"] // "none")"' | \
    sort -k3,3 -k1,1) | column -t

# LEGACY: Ancienne commande reports (gardée pour compatibilité)
reports-legacy env="all":
    #!/usr/bin/env bash
    if [ "{{env}}" == "all" ]; then
        echo "📊 Génération des rapports consolidés (DEV + PROD)..."
        # DEV
        python3 scripts/reports/generate_actual_state.py --env dev --output docs/reports/STATE-ACTUAL-dev.md --json-output docs/reports/STATE-dev.json
        python3 scripts/reports/conformity_checker.py --actual docs/reports/STATE-ACTUAL-dev.md --output docs/reports/CONFORMITY-dev.md
        # PROD
        python3 scripts/reports/generate_actual_state.py --env prod --output docs/reports/STATE-ACTUAL-prod.md --json-output docs/reports/STATE-prod.json
        python3 scripts/reports/conformity_checker.py --actual docs/reports/STATE-ACTUAL-prod.md --output docs/reports/CONFORMITY-prod.md
        # CONSOLIDATED
        python3 scripts/reports/generate_status_report.py \
            --dev-state docs/reports/STATE-dev.json \
            --prod-state docs/reports/STATE-prod.json \
            --dev-conformity docs/reports/CONFORMITY-dev.md \
            --prod-conformity docs/reports/CONFORMITY-prod.md
        # Final cleanup for main files
        cp docs/reports/STATE-ACTUAL-prod.md docs/reports/STATE-ACTUAL.md
        echo "✅ Rapports consolidés générés dans docs/reports/"
    else
        echo "📊 Génération des rapports d'état pour l'environnement {{env}}..."
        python3 scripts/reports/generate_actual_state.py --env {{env}} --output docs/reports/STATE-ACTUAL.md
        python3 scripts/reports/conformity_checker.py --actual docs/reports/STATE-ACTUAL.md --output docs/reports/CONFORMITY-REPORT.md
        if [ "{{env}}" == "dev" ]; then
            python3 scripts/reports/generate_status_report.py --dev-conformity docs/reports/CONFORMITY-REPORT.md
        else
            python3 scripts/reports/generate_status_report.py --prod-conformity docs/reports/CONFORMITY-REPORT.md
        fi
        echo "✅ Rapports générés dans docs/reports/"
    fi

# ============================================
# UTILITAIRES
# ============================================

burst title:
    bd create "{{title}}" --status open --assignee coding-agent --label burst
    @echo "✅ Idée enregistrée dans Beads"

lint:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "🔍 Validation YAML avec yamllint..."
    if find apps/ argocd/ -name "*.yaml" | xargs yamllint -c yamllint-config.yml; then
        echo "✅ Validation YAML réussie"
        exit 0
    else
        echo "❌ Validation YAML échouée"
        echo ""
        echo "⚠️  Ces erreurs bloqueront le push dans GitHub Actions"
        echo "💡 Corrigez les erreurs avant de faire git push"
        exit 1
    fi

# ============================================
# AIDE
# ============================================
help:
    @echo "🎯 WORKFLOW VIXENS - State Machine GitOps"
    @echo ""
    @echo "Commandes principales:"
    @echo "  just resume              - Afficher phase actuelle et instructions"
    @echo "  just start <task_id>     - Démarrer une nouvelle tâche (phase 0)"
    @echo "  just next <task_id>      - Avancer à la phase suivante (avec validation)"
    @echo "  just close <task_id>     - Fermer la tâche (phase 6 uniquement)"
    @echo ""
    @echo "Helpers GitOps:"
    @echo "  just wait-argocd <app>   - Attendre ArgoCD sync (Synced+Healthy)"
    @echo "  just promote-prod        - Instructions promotion production"
    @echo "  just SendToProd <ver>    - ⭐ Promotion automatisée vers prod (vX.Y.Z)"
    @echo ""
    @echo "Rapports & Qualité:"
    @echo "  just reports             - ⭐ TOUS les rapports (VPA + lint + versions + dashboards)"
    @echo "  just lint                - Valider YAML uniquement"
    @echo "  just report              - ⚠️  DEPRECATED: Utiliser 'just reports'"
    @echo ""
    @echo "Utilitaires:"
    @echo "  just reset-phase <id> <N>  - Réinitialiser à la phase N (debug)"
    @echo "  just burst <title>         - Créer une idée rapide"
    @echo ""
    @echo "Phases du workflow:"
    @echo "  0. SELECTION      - Comprendre la tâche"
    @echo "  1. PREREQS        - Vérifier prérequis techniques"
    @echo "  2. DOCUMENTATION  - Charger documentation"
    @echo "  3. IMPLEMENTATION - Coder (Serena/Archon) - SCOPE LIMITÉ"
    @echo "  4. DEPLOYMENT     - Commit + Push + ArgoCD sync ⭐"
    @echo "  5. VALIDATION     - Valider APRÈS déploiement"
    @echo "  6. FINALIZATION   - Documentation + Close"
    @echo ""
    @echo "🚫 RÈGLES CRITIQUES:"
    @echo "  • GitOps ONLY (ZERO kubectl apply direct)"
    @echo "  • DRY (réutiliser apps/_shared/)"
    @echo "  • Scope limité à l'app dans le titre"
    @echo "  • Deployment + Validation OBLIGATOIRES"
    @echo "  • Production: Promotion via tag uniquement"

# ============================================
# HELPERS D'ORCHESTRATION MULTI-AGENT
# ============================================

# Réassigner une tâche à un agent spécifique
assign task_id agent:
    #!/usr/bin/env python3
    import subprocess, sys
    
    valid_agents = ['claude', 'gemini', 'coding-agent']
    agent = "{{agent}}"
    
    if agent not in valid_agents:
        print(f"❌ Agent invalide: {agent}")
        print(f"   Agents valides: {', '.join(valid_agents)}")
        sys.exit(1)
    
    result = subprocess.run([
        "bd", "update", "{{task_id}}",
        "--assignee", agent
    ])
    
    if result.returncode == 0:
        print(f"✅ Tâche {{task_id}} assignée à: {agent}")
    else:
        print(f"❌ Erreur lors de l'assignation")
        sys.exit(1)

# Prendre une tâche pour l'agent actuel
claim task_id:
    #!/usr/bin/env python3
    import subprocess, sys, os
    
    def get_current_agent():
        """Détecter l'agent actuel"""
        agent = os.getenv("AGENT_NAME")
        if agent:
            return agent
        if os.path.exists("/.claude") or os.path.exists(".claude"):
            return "claude"
        return "coding-agent"
    
    current_agent = get_current_agent()
    
    result = subprocess.run([
        "bd", "update", "{{task_id}}",
        "--assignee", current_agent
    ])
    
    if result.returncode == 0:
        print(f"✅ Tâche {{task_id}} réclamée par: {current_agent}")
    else:
        print(f"❌ Erreur lors de la réclamation")
        sys.exit(1)

# Lister les agents disponibles et leurs capacités
agents:
    #!/usr/bin/env python3
    import os
    
    def get_current_agent():
        """Détecter l'agent actuel"""
        agent = os.getenv("AGENT_NAME")
        if agent:
            return agent
        if os.path.exists("/.claude") or os.path.exists(".claude"):
            return "claude"
        return "coding-agent"
    
    current_agent = get_current_agent()
    
    print("🤖 Agents Disponibles:\n")
    
    agents_info = {
        'claude': {
            'name': 'Claude Code',
            'capabilities': ['Code analysis', 'File editing', 'Architecture design', 'Documentation'],
            'types': ['feature', 'refactor', 'docs']
        },
        'gemini': {
            'name': 'Gemini Agent',
            'capabilities': ['Automation', 'Workflow execution', 'Batch processing'],
            'types': ['task', 'chore', 'fix']
        },
        'coding-agent': {
            'name': 'Generic Coding Agent',
            'capabilities': ['General purpose'],
            'types': ['all']
        }
    }
    
    for agent_id, info in agents_info.items():
        marker = "👉" if agent_id == current_agent else "  "
        print(f"{marker} {agent_id:15s} - {info['name']}")
        print(f"   Capacités: {', '.join(info['capabilities'])}")
        print(f"   Types préférés: {', '.join(info['types'])}")
        print()
    
    print(f"Agent actuel détecté: {current_agent}")
    print("\n💡 Pour changer d'agent:")
    print("   export AGENT_NAME=claude")
    print("   export AGENT_NAME=gemini")

# Voir la charge de travail par agent
workload:
    #!/usr/bin/env python3
    import subprocess, json, sys
    from collections import defaultdict
    
    # Récupérer toutes les tâches
    result_in_progress = subprocess.run(
        ["bd", "list", "--status", "in_progress", "--json"],
        capture_output=True, text=True
    )
    
    result_open = subprocess.run(
        ["bd", "list", "--status", "open", "--json"],
        capture_output=True, text=True
    )
    
    if result_in_progress.returncode != 0 or result_open.returncode != 0:
        print("❌ Erreur lors de la récupération des tâches")
        sys.exit(1)
    
    in_progress = json.loads(result_in_progress.stdout) if result_in_progress.stdout.strip() else []
    open_tasks = json.loads(result_open.stdout) if result_open.stdout.strip() else []
    
    # Compter par agent
    workload = defaultdict(lambda: {'in_progress': 0, 'open': 0})
    
    for task in in_progress:
        assignee = task.get('assignee') or 'unassigned'
        workload[assignee]['in_progress'] += 1
    
    for task in open_tasks:
        assignee = task.get('assignee') or 'unassigned'
        workload[assignee]['open'] += 1
    
    print("📊 Charge de Travail par Agent:\n")
    
    # Trier par nombre de tâches in_progress décroissant
    sorted_agents = sorted(workload.items(), 
                          key=lambda x: (x[1]['in_progress'], x[1]['open']), 
                          reverse=True)
    
    for agent, counts in sorted_agents:
        in_prog = counts['in_progress']
        open_count = counts['open']
        total = in_prog + open_count
        
        # Indicateur visuel de charge
        if in_prog == 0:
            indicator = "🟢"
        elif in_prog == 1:
            indicator = "🟡"
        else:
            indicator = "🔴"
        
        print(f"{indicator} {agent:15s}  {in_prog} in_progress, {open_count} open (total: {total})")
    
    print("\n💡 Utilisation:")
    print("   just assign <task_id> <agent>  # Réassigner une tâche")
    print("   just claim <task_id>            # Prendre une tâche")

# ============================================
# GESTION DE L'HIBERNATION
# ============================================

# Mettre une application en hibernation (replicas=0)
hibernate app_name:
    @python3 scripts/infra/hibernate.py hibernate {{app_name}}
    @git push origin main

# Réactiver une application (replicas=1)
unhibernate app_name:
    @python3 scripts/infra/hibernate.py unhibernate {{app_name}}
    @git push origin main

# Lister les applications hibernées
hibernated:
    @python3 scripts/infra/hibernate.py list



# ============================================
# DEV TESTING: WAKE/SLEEP WORKFLOW
# ============================================

# Réveiller une application pour test (sans modifier Git)
wake app_name:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "🌅 Réveil de {{app_name}} pour test..."
    
    # 1. Forcer sync ArgoCD
    echo "  1️⃣  Sync ArgoCD..."
    argocd app sync {{app_name}} --prune || {
        echo "⚠️  ArgoCD sync échouée, app peut-être pas dans ArgoCD"
        echo "  Essai de scale direct..."
    }
    
    # 2. Désactiver self-heal
    echo "  2️⃣  Désactivation self-heal..."
    argocd app set {{app_name}} --self-heal=false || {
        echo "⚠️  Impossible de désactiver self-heal, l'app n'existe peut-être pas dans ArgoCD"
    }
    
    # 3. Scaler à 1 replica
    echo "  3️⃣  Scale à 1 replica..."
    kubectl scale deployment {{app_name}} -n {{app_name}} --replicas=1 || {
        echo "❌ Erreur: Deployment non trouvé"
        echo "   Vérifier: kubectl get deployments -A | grep {{app_name}}"
        exit 1
    }
    
    echo ""
    echo "✅ {{app_name}} réveillé (self-heal désactivé)"
    echo "💡 Tester l'application, puis: just sleep {{app_name}}"

# Remettre en veille après test (ArgoCD resync à replicas: 0)
sleep app_name:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "💤 Mise en veille de {{app_name}}..."
    
    # Réactiver self-heal (ArgoCD va resyncer automatiquement)
    echo "  🔄 Réactivation self-heal..."
    argocd app set {{app_name}} --self-heal=true || {
        echo "⚠️  Impossible de réactiver self-heal"
        echo "  Scale manuel à 0..."
        kubectl scale deployment {{app_name}} -n {{app_name}} --replicas=0
        exit 0
    }
    
    echo ""
    echo "✅ Self-heal réactivé"
    echo "⏳ ArgoCD va resyncer et remettre replicas: 0 (~30s)"
    echo "💡 Vérifier: kubectl get deployment {{app_name}} -n {{app_name}}"

report:
    @echo "🔍 Selene lance l'analyse profonde..."
    @bash {{scripts_path}}/reports/generate_actual_state.sh
    @echo "📉 Mise à jour de STATUS.md..."
    @# Extraction rapide du score moyen pour le dashboard
    @SCORE=$$(grep "|" {{report_path}}/STATE-ACTUAL.md | tail -n +3 | awk -F'|' '{sum+=$13; ++n} END { print sum/n }'); \
    sed -i "s/Score Moyen : .*/Score Moyen : $$SCORE/g" docs/STATUS.md
    @echo "✅ Rapport actualisé. Score Moyen du Cluster : $$SCORE"

# Vérifier la conformité ADR-008
audit:
    @echo "⚖️ Vérification de la loi de Serena..."
    @kubectl get pods -A -o json | jq -r '.items[] | select(.status.qosClass == "BestEffort") | "⚠️ ATTENTION : \(.metadata.namespace)/\(.metadata.name) est en BestEffort !"'

# --- 💾 GESTION DES TÂCHES (BEADS) ---

# Lister les tâches Beads en cours
tasks:
    @jq -r '. | select(.status == "open" or .status == "in_progress") | "[\(.id)] \(.title) (Priority: \(.priority))"' .beads/issues.jsonl

# --- 🧹 HOUSEKEEPING (ADR-020) ---

# Nettoyer les ReplicaSets orphelins (> 3)
cleanup:
    @echo "🧹 Ménage de printemps pour la panthère..."
    @kubectl get deploy -A -o json | jq -r '.items[] | select(.spec.revisionHistoryLimit > 3) | "kubectl patch deploy -n \(.metadata.namespace) \(.metadata.name) -p \"{\"spec\":{\"revisionHistoryLimit\":3}}\""' | bash
    @echo "✨ Cluster assaini."

# --- 🏥 RECOVERY ---

# Vérifier l'intégrité iSCSI après ton crash DSM
check-iscsi:
    @echo "🩹 Diagnostic des plaies de Charchess..."
    @kubectl get pv | grep -v "Bound" && echo "❌ PVs orphelins détectés !" || echo "✅ Stockage stable."

# --- 🦊 PERSONA ---

# Demander une gratouille (Usage réservé au Snep)
scratch:
    @echo "Selene : *Oreilles qui s'abaissent* ... Seulement parce que tu as fini ton report, Charchess. Mais ne t'habitue pas."
