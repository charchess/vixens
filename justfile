#!/usr/bin/env just --justfile
# workflow.just - State Machine Workflow pour Gemini (Coding Agent)
# Phases séquentielles avec garde-fous et processus GitOps complet

set shell := ["bash", "-uc"]

JUST := "just -f WORKFLOW.just"

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
    import subprocess, json, sys, re

    # Récupérer la tâche en cours
    result = subprocess.run(
        ["bd", "list", "--status", "in_progress", "--assignee", "coding-agent", "--json"],
        capture_output=True, text=True
    )

    if result.returncode != 0:
        print("❌ Erreur bd:", result.stderr)
        sys.exit(1)

    tasks = json.loads(result.stdout)

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
                f"Vérifier branch actuelle: git branch --show-current (doit être 'dev')",
                "Commit les changements: git add + git commit -m '...'",
                "Push vers dev: git push origin dev",
                f"Attendre ArgoCD sync: just wait-argocd {app_name}",
                "Vérifier status: Health=Healthy, Sync=Synced"
            ],
            "forbidden": [
                "❌ INTERDIT: Push vers main (uniquement via PR)",
                "❌ INTERDIT: Créer des tags manuellement",
                "❌ INTERDIT: Avancer avant ArgoCD Synced+Healthy",
                "❌ INTERDIT: kubectl apply/edit direct"
            ],
            "rules": [
                "📜 Branch: Toujours dev pour développement",
                "📜 GitOps: Git push → ArgoCD auto-sync",
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
                "🎯 PROMOTION VERS PRODUCTION:",
                "  1. Validé sur dev ✅",
                "  2. Pour déployer en prod:",
                "     → Créer PR: dev → main",
                "     → Attendre review + merge",
                "     → Tag auto-créé: prod-vX.Y.Z",
                "     → ArgoCD sync automatique sur prod cluster",
                "  3. Ne JAMAIS push direct sur main",
                "  4. Ne JAMAIS créer de tag manuellement"
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
    import subprocess, json, re, sys

    # Vérifier qu'on est sur dev branch
    branch_result = subprocess.run(
        ["git", "branch", "--show-current"],
        capture_output=True, text=True
    )
    current_branch = branch_result.stdout.strip()

    if current_branch != "dev":
        print(f"⚠️  WARNING: Sur branch '{current_branch}', pas 'dev'")
        print("   Le workflow GitOps nécessite d'être sur dev")
        response = input("   Continuer quand même? (y/N): ")
        if response.lower() != 'y':
            sys.exit(1)

    # Mettre à jour le statut et initialiser la phase
    subprocess.run([
        "bd", "update", "{{task_id}}",
        "--status", "in_progress",
        "--assignee", "coding-agent",
        "--notes", f"PHASE:0 - Tâche démarrée (branch: {current_branch})"
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
            print("⚠️  Aucun changement détecté. Êtes-vous sûr d'avoir terminé l'implémentation?")
            response = input("Continuer quand même? (y/N): ")
            if response.lower() != 'y':
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
        if current_branch != "dev":
            print(f"⚠️  WARNING: Sur branch '{current_branch}', attendu 'dev'")

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
    #            response = input("   Continuer la vérification ArgoCD? (y/N): ")
    #            if response.lower() != 'y':
    #                sys.exit(1)

        # Vérifier ArgoCD sync status
        print(f"🔍 Vérification ArgoCD pour: {app_name}")

        # Détecter si l'app est hibernée (commentée dans kustomization.yaml)
        was_hibernated = False
        kustomization_path = f"argocd/overlays/{current_branch}/kustomization.yaml"
        try:
            with open(kustomization_path, 'r') as f:
                content = f.read()
                # Chercher si l'app est commentée
                if f"# - apps/{app_name}.yaml" in content:
                    print(f"   ⚠️  Application '{app_name}' est HIBERNÉE dans {current_branch}")
                    print(f"   (Commentée dans {kustomization_path})")
                    print()
                    print("   💡 Pour tester, l'app doit être RÉACTIVÉE puis RE-HIBERNÉE après validation")
                    response = input("   → Décommenter automatiquement pour test? (y/N): ")
                    
                    if response.lower() == 'y':
                        # Décommenter l'app
                        new_content = content.replace(
                            f"# - apps/{app_name}.yaml",
                            f"- apps/{app_name}.yaml"
                        )
                        with open(kustomization_path, 'w') as f:
                            f.write(new_content)
                        
                        print(f"   ✅ App décommentée dans {kustomization_path}")
                        print("   📝 Commit des changements...")
                        
                        # Commit automatique
                        subprocess.run(["git", "add", kustomization_path])
                        subprocess.run([
                            "git", "commit", "-m",
                            f"test({app_name}): réactiver temporairement pour test (était hibernée)"
                        ])
                        subprocess.run(["git", "push", "origin", current_branch])
                        
                        print("   ⏳ Attendre ~30s pour ArgoCD auto-sync...")
                        import time
                        time.sleep(30)
                        
                        # Marquer qu'elle était hibernée (pour la re-hiberner en phase 6)
                        was_hibernated = True
                        subprocess.run([
                            "bd", "update", "{{task_id}}",
                            "--notes", f"{notes}\nWAS_HIBERNATED: {app_name} (à re-hiberner en Phase 6)"
                        ])
                    else:
                        print("   ⏸️  Décommenter annulé - impossible de tester une app hibernée")
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
    #            response = input("   Ignorer cette vérification? (y/N): ")
    #            if response.lower() != 'y':
    #                sys.exit(1)
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
                    print(f"   ⚠️  Application pas encore Synced (status: {sync_status})")
                    print(f"   💡 Attendre avec: just wait-argocd {app_name}")
                    response = input("   Ignorer et continuer? (y/N): ")
                    if response.lower() != 'y':
                        sys.exit(1)

                if health_status not in ['Healthy', 'Progressing']:
                    print(f"   ⚠️  Application pas Healthy (status: {health_status})")
                    response = input("   Continuer quand même? (y/N): ")
                    if response.lower() != 'y':
                        sys.exit(1)

                print("   ✅ ArgoCD status OK")
            except Exception as e:
                print(f"   ⚠️  Erreur parsing status ArgoCD: {e}")

        # Marquer le déploiement
        subprocess.run([
            "bd", "update", "{{task_id}}",
            "--notes", f"{notes}\nDEPLOYED: {datetime.now().isoformat()} (branch: {current_branch})"
        ])
        print("✅ Phase DEPLOYMENT complétée")

    elif current_phase == 5:
        # Phase VALIDATION: BLOQUER si validation non passée
        if not app_name:
            print("❌ BLOQUÉ: Impossible de valider sans nom d'application")
            sys.exit(1)

        print(f"🎭 VALIDATION OBLIGATOIRE (post-deployment): {app_name}")
        val_result = subprocess.run(
            ["python3", "scripts/validate.py", app_name, "dev"],
            capture_output=True, text=True
        )

        if val_result.returncode != 0:
            print(f"❌ VALIDATION ÉCHOUÉE:\n{val_result.stderr}")
            subprocess.run([
                "bd", "update", "{{task_id}}",
                "--notes", f"{notes}\nVALIDATION FAIL: {val_result.stderr[:200]}"
            ])
            print("\n💡 Pour corriger: just reset-phase {{task_id}} 3")
            sys.exit(1)

        print("✅ VALIDATION RÉUSSIE")
        # Marquer la validation dans les notes
        subprocess.run([
            "bd", "update", "{{task_id}}",
            "--notes", f"{notes}\nVALIDATION OK: {datetime.now().isoformat()}"
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

    # Vérifier si l'app était hibernée et proposer de la re-hiberner
    if "WAS_HIBERNATED:" in notes:
        # Extraire le nom de l'app des notes
        hibernated_match = re.search(r'WAS_HIBERNATED: (\w+)', notes)
        if hibernated_match:
            app_name = hibernated_match.group(1)
            print()
            print(f"💤 HIBERNATION DÉTECTÉE: '{app_name}' était hibernée avant test")
            print()
            response = input("   → Re-hiberner l'application maintenant? (y/N): ")
            
            if response.lower() == 'y':
                # Déterminer la branch
                current_branch_result = subprocess.run(
                    ["git", "branch", "--show-current"],
                    capture_output=True, text=True
                )
                current_branch = current_branch_result.stdout.strip()
                
                # Re-commenter dans kustomization.yaml
                kustomization_path = f"argocd/overlays/{current_branch}/kustomization.yaml"
                try:
                    with open(kustomization_path, 'r') as f:
                        content = f.read()
                    
                    # Re-commenter l'app
                    new_content = content.replace(
                        f"  - apps/{app_name}.yaml",
                        f"  # - apps/{app_name}.yaml"
                    )
                    
                    with open(kustomization_path, 'w') as f:
                        f.write(new_content)
                    
                    print(f"   ✅ App re-commentée dans {kustomization_path}")
                    print("   📝 Commit des changements...")
                    
                    # Commit automatique
                    subprocess.run(["git", "add", kustomization_path])
                    subprocess.run([
                        "git", "commit", "-m",
                        f"chore({app_name}): re-hiberner après test (économie ressources)"
                    ])
                    subprocess.run(["git", "push", "origin", current_branch])
                    
                    print("   💤 Application re-hibernée avec succès")
                    
                    # Marquer la re-hibernation dans les notes
                    subprocess.run([
                        "bd", "update", "{{task_id}}",
                        "--notes", f"{notes}\nRE_HIBERNATED: {app_name}"
                    ])
                except Exception as e:
                    print(f"   ⚠️  Erreur lors de la re-hibernation: {e}")
                    print("   💡 Vérifier manuellement le kustomization.yaml")
            else:
                print("   ⚠️  App laissée active - penser à la re-hiberner manuellement")
            print()

    # Afficher checklist finale
    print("📋 CHECKLIST FINALE:")
    print("   [✓] Code déployé sur dev (ArgoCD synced)")
    print("   [✓] Validation réussie")
    print("   [ ] Documentation à jour (docs/applications/<category>/<app>.md)")
    print("   [ ] STATUS.md à jour si nécessaire")
    print("   [ ] Changements de doc committés + pushés")
    print()
    print("🎯 PROMOTION PRODUCTION:")
    print("   Pour déployer en production:")
    print("   1. Créer PR: dev → main")
    print("   2. Review + merge")
    print("   3. Tag auto: prod-vX.Y.Z")
    print("   4. ArgoCD sync auto sur prod")
    print()

    response = input("✅ Tout est prêt pour fermer? (y/N): ")
    if response.lower() != 'y':
        print("⏸️  Fermeture annulée")
        sys.exit(0)

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
    @echo "🎯 PROCESSUS DE PROMOTION VERS PRODUCTION"
    @echo ""
    @echo "📋 Prérequis:"
    @echo "   ✅ Changements validés sur dev"
    @echo "   ✅ Tâche Beads fermée"
    @echo "   ✅ Branch dev à jour"
    @echo ""
    @echo "🔄 Étapes de promotion:"
    @echo "   1. Vérifier l'état:"
    @echo "      git status"
    @echo "      git log dev..main  # Voir ce qui sera promu"
    @echo ""
    @echo "   2. Créer Pull Request:"
    @echo "      gh pr create --base main --head dev --title 'Release vX.Y.Z' --body '...'"
    @echo ""
    @echo "   3. Review + Merge:"
    @echo "      - Review dans GitHub UI"
    @echo "      - Merge PR (crée tag auto prod-vX.Y.Z)"
    @echo ""
    @echo "   4. Vérifier déploiement prod:"
    @echo "      kubectl -n argocd get applications  # cluster prod"
    @echo "      just wait-argocd <app_name>  # avec KUBECONFIG prod"
    @echo ""
    @echo "⚠️  RÈGLES:"
    @echo "   • JAMAIS push direct sur main"
    @echo "   • JAMAIS créer de tag manuellement"
    @echo "   • TOUJOURS passer par PR dev → main"
    @echo "   • Tags auto: prod-vX.Y.Z créés par GitHub Actions"

# ============================================
# UTILITAIRES
# ============================================

burst title:
    bd create "{{title}}" --status open --assignee coding-agent --label burst
    @echo "✅ Idée enregistrée dans Beads"

create-task:
    #!/usr/bin/env python3
    import subprocess, re, sys, os, glob

    print("🎯 CRÉATION DE TÂCHE (Template Vixens)")
    print("=" * 50)
    print()
    print("📋 Format requis: 'Action Description (app_name)'")
    print("   Exemples:")
    print("   • Migrer vers version 3.2 (traefik)")
    print("   • Corriger sync loop (argocd)")
    print("   • Ajouter widget monitoring (homepage)")
    print()

    # Action
    print("1️⃣  ACTION (verbe)")
    print("   Suggestions: Migrer, Corriger, Ajouter, Configurer, Mettre à jour")
    action = input("   → Action: ").strip()

    if not action:
        print("❌ Action requise")
        sys.exit(1)

    # Description courte
    print("\n2️⃣  DESCRIPTION COURTE")
    print("   Ex: 'vers version 3.2', 'le bug de sync', 'support HTTPS'")
    desc = input("   → Description: ").strip()

    if not desc:
        print("❌ Description requise")
        sys.exit(1)

    # Application
    print("\n3️⃣  APPLICATION CIBLÉE")
    print("   Chercher dans apps/...")
    app_input = input("   → Application: ").strip()

    if not app_input:
        print("❌ Application requise")
        sys.exit(1)

    # Vérifier que l'app existe dans apps/
    app_found = False
    app_path = None

    # Chercher dans apps/**/
    for root, dirs, files in os.walk("apps"):
        dir_name = os.path.basename(root)
        if dir_name == app_input:
            app_found = True
            app_path = root
            break

    if not app_found:
        print(f"   ⚠️  Application '{app_input}' non trouvée dans apps/")
        print("   Applications disponibles:")

        # Lister les apps
        app_dirs = []
        for root, dirs, files in os.walk("apps"):
            # Ignorer _shared et les overlays
            if os.path.basename(root) in ['_shared', 'overlays', 'base']:
                continue
            # Si contient base/ ou kustomization.yaml, c'est une app
            if 'base' in dirs or any(f == 'kustomization.yaml' for f in files):
                app_dirs.append(os.path.basename(root))

        # Afficher triées
        for app in sorted(set(app_dirs))[:20]:
            print(f"      • {app}")

        response = input(f"\n   Continuer avec '{app_input}' quand même? (y/N): ")
        if response.lower() != 'y':
            sys.exit(0)
    else:
        print(f"   ✅ Application trouvée: {app_path}")

    app = app_input

    # Construire le titre selon template
    title = f"{action} {desc} ({app})"

    # Description détaillée (optionnelle)
    print("\n4️⃣  DESCRIPTION DÉTAILLÉE (optionnel)")
    print("   Contexte supplémentaire, liens, notes...")
    description = input("   → Description: ").strip()

    # Priority
    print("\n5️⃣  PRIORITÉ")
    print("   0 = Critical (P0) - Bloquant, urgent")
    print("   1 = High (P1) - Important, à faire rapidement")
    print("   2 = Medium (P2) - Normal (défaut)")
    print("   3 = Low (P3) - Peut attendre")
    print("   4 = Backlog (P4) - Future")
    priority_input = input("   → Priority [0-4] (défaut: 2): ").strip()
    priority = priority_input if priority_input in ['0','1','2','3','4'] else '2'

    # Récapitulatif
    print("\n" + "=" * 50)
    print("📋 RÉCAPITULATIF:")
    print(f"   Titre: {title}")
    if description:
        print(f"   Description: {description}")
    print(f"   Priority: {priority} (P{priority})")
    print(f"   Assigné à: coding-agent")
    print(f"   Status: open")
    print("=" * 50)

    # Confirmation
    confirm = input("\n✅ Créer cette tâche? (y/N): ")
    if confirm.lower() != 'y':
        print("❌ Création annulée")
        sys.exit(0)

    # Créer avec bd
    cmd = [
        "bd", "create",
        "--title", title,
        "--status", "open",
        "--assignee", "coding-agent",
        "--priority", priority
    ]

    if description:
        cmd.extend(["--description", description])

    result = subprocess.run(cmd, capture_output=True, text=True)

    if result.returncode == 0:
        print("\n✅ Tâche créée avec succès!")

        # Extraire task_id de la sortie bd
        match = re.search(r'(beads-[a-z0-9]+)', result.stdout + result.stderr)
        if match:
            task_id = match.group(1)
            print(f"   ID: {task_id}")
            print(f"\n💡 Commandes suivantes:")
            print(f"   just start {task_id}    # Démarrer la tâche")
            print(f"   just resume             # Voir toutes les tâches")
        else:
            print("💡 Lancer: just resume")
    else:
        print(f"\n❌ Erreur lors de la création:")
        print(result.stderr)
        sys.exit(1)

lint:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "🔍 Validation YAML avec yamllint..."
    if yamllint -c yamllint-config.yml apps/**/*.yaml argocd/**/*.yaml; then
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
    @echo ""
    @echo "Utilitaires:"
    @echo "  just create-task           - Créer une tâche (template guidé) ⭐"
    @echo "  just reset-phase <id> <N>  - Réinitialiser à la phase N (debug)"
    @echo "  just burst <title>         - Créer une idée rapide"
    @echo "  just lint                  - Valider YAML"
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
    @echo "  • Production: PR dev→main uniquement"
