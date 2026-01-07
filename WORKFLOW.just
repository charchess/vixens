#!/usr/bin/env just --justfile
# workflow.just - Backend bd (Beads) + Frontends Serena/Archon

set shell := ["bash", "-uc"]

# ============================================
# COMMANDE PRINCIPALE : Reprendre la bonne tâche
# ============================================
default:
    @just resume

resume:
    #!/usr/bin/env python3
    import subprocess, json, sys
    
    result = subprocess.run(
        ["bd", "list", "--status", "in_progress", "--assignee", "coding-agent", "--json"],
        capture_output=True, text=True
    )
    
    if result.returncode != 0:
        print("❌ Erreur bd:", result.stderr)
        sys.exit(1)
    
    tasks = json.loads(result.stdout)
    
    if tasks:
        t = tasks[0]
        print(f"🔥 RESUME: {t['id']} | {t['title']}")
        print(f"   Commande: just work {t['id']}")
    else:
        print("📋 AUCUNE TÂCHE EN COURS. TODO:")
        subprocess.run(["bd", "list", "--status", "open", "--json", "--limit", "10"])

# ============================================
# TRAVAILLER SUR UNE TÂCHE (Orchestration)
# ============================================
work task_id:
    bd update {{task_id}} --status in_progress --assignee coding-agent
    @echo "=== Phase 1: Prérequis ==="
    just _check-prereqs {{task_id}}
    
    @echo "=== Phase 2: Doc Application ==="
    just _load-doc {{task_id}}
    
    @echo "=== Phase 3: Exécution (Gemini) ==="
    @echo "   💡 Gemini: analyse et propose des changements"
    @echo "   💡 Utiliser: @serena pour le code, @archon pour le RAG"
    
    @echo "=== Phase 4: Validation FORCÉE ==="
    just _validate {{task_id}}

# ============================================
# SOUS-ÉTAPES (Privées)
# ============================================

_check-prereqs task_id:
    #!/usr/bin/env python3
    import subprocess, json, re, sys
    
    try:
        task_data = subprocess.run(
            ["bd", "show", "{{task_id}}", "--json"], 
            capture_output=True, text=True, check=True
        ).stdout
        
        task = json.loads(task_data)
        title = task.get("title", "")
        
        if "PVC" in title and "RWO" in title:
            print("   ⚠️  ALERTE: PVC RWO détecté → strategy: Recreate nécessaire")
            subprocess.run([
                "bd", "update", "{{task_id}}", 
                "--note", "AUTO: PVC RWO nécessite strategy: Recreate"
            ])
    except:
        print("   ⚠️  Erreur lecture tâche")
        sys.exit(1)

_load-doc task_id:
    #!/usr/bin/env python3
    import subprocess, json, re, sys
    
    try:
        task_data = subprocess.run(
            ["bd", "show", "{{task_id}}", "--json"], 
            capture_output=True, text=True, check=True
        ).stdout
        
        task = json.loads(task_data)
        title = task.get("title", "")
        match = re.search(r'\(([^)]+)\)', title)
        
        if match:
            app_name = match.group(1)
            print(f"   📖 Doc à consulter: docs/applications/{app_name}.md")
            print(f"   💡 Commande Gemini: @serena get_file_content docs/applications/{app_name}.md")
        else:
            print("   ℹ️  Pas d'app détectée dans le titre")
    except:
        print("   ⚠️  Erreur lecture tâche")

_validate task_id:
    #!/usr/bin/env python3
    import subprocess, json, re, sys
    
    try:
        task_data = subprocess.run(
            ["bd", "show", "{{task_id}}", "--json"], 
            capture_output=True, text=True, check=True
        ).stdout
        
        task = json.loads(task_data)
        title = task.get("title", "")
        match = re.search(r'\(([^)]+)\)', title)
        
        if not match:
            print("   ❌ ABORT: Pas d'application à valider")
            sys.exit(1)
        
        app_name = match.group(1)
        print(f"   🎭 VALIDATION DEV: {app_name}")
        
        result = subprocess.run([
            "python3", "scripts/validate.py", app_name, "dev"
        ], capture_output=True, text=True)
        
        if result.returncode != 0:
            print(f"   ❌ VALIDATION ÉCHOUÉE")
            subprocess.run([
                "bd", "update", "{{task_id}}", 
                "--note", f"VALIDATION FAIL: {result.stderr}"
            ])
            sys.exit(1)
        
        print("   ✅ VALIDATION DEV OK")
        subprocess.run([
            "bd", "update", "{{task_id}}", 
            "--status", "review", "--assignee", "user"
        ])
    except Exception as e:
        print(f"   ❌ Erreur validation: {e}")
        sys.exit(1)

# ============================================
# UTILITAIRES
# ============================================

burst title:
    bd create "{{title}}" --status open --assignee coding-agent --label burst
    @echo "✅ Idée enregistrée dans Beads"
