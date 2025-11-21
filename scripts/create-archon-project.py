#!/usr/bin/env python3
"""
Script pour créer le projet et les tâches Terraform Refactoring dans Archon
Via l'API REST Archon
"""

import json
import requests
import sys
from pathlib import Path

ARCHON_API_URL = "http://192.168.199.78:3737"
TASKS_FILE = "/root/vixens/docs/terraform-refactoring-archon-tasks.json"

def load_tasks_config():
    """Charge le fichier JSON de configuration des tâches"""
    with open(TASKS_FILE, 'r') as f:
        return json.load(f)

def create_project(project_data):
    """Crée le projet dans Archon"""
    url = f"{ARCHON_API_URL}/api/projects"

    payload = {
        "name": project_data["name"],
        "description": project_data["description"],
        "metadata": project_data["metadata"]
    }

    print(f"🚀 Création du projet: {project_data['name']}")

    try:
        response = requests.post(url, json=payload, timeout=10)
        response.raise_for_status()
        project = response.json()
        print(f"✅ Projet créé: {project.get('id', 'unknown')}")
        return project
    except requests.exceptions.RequestException as e:
        print(f"❌ Erreur création projet: {e}")
        return None

def create_task(project_id, task_data):
    """Crée une tâche dans Archon"""
    url = f"{ARCHON_API_URL}/api/projects/{project_id}/tasks"

    payload = {
        "title": task_data["title"],
        "description": task_data["description"],
        "status": task_data["status"],
        "priority": task_data["priority"],
        "estimated_duration": task_data["estimated_duration"],
        "labels": task_data.get("labels", []),
        "dependencies": task_data.get("dependencies", []),
        "metadata": {
            "acceptance_criteria": task_data.get("acceptance_criteria", []),
            "implementation_steps": task_data.get("implementation_steps", []),
            "validation_commands": task_data.get("validation_commands", []),
            "blockers": task_data.get("blockers", []),
            "documentation_refs": task_data.get("documentation_refs", [])
        }
    }

    print(f"  📝 Création tâche: {task_data['id']} - {task_data['title']}")

    try:
        response = requests.post(url, json=payload, timeout=10)
        response.raise_for_status()
        task = response.json()
        print(f"  ✅ Tâche créée: {task.get('id', 'unknown')}")
        return task
    except requests.exceptions.RequestException as e:
        print(f"  ❌ Erreur création tâche: {e}")
        return None

def main():
    print("="*80)
    print("CRÉATION PROJET TERRAFORM REFACTORING DANS ARCHON")
    print("="*80)
    print()

    # Charger la configuration
    print("📋 Chargement configuration depuis:", TASKS_FILE)
    config = load_tasks_config()

    # Créer le projet
    project = create_project(config["project"])
    if not project:
        print("\n❌ Échec création projet - arrêt")
        sys.exit(1)

    project_id = project.get("id")
    print(f"\n✅ Projet créé avec ID: {project_id}")
    print()

    # Créer toutes les tâches
    print(f"📝 Création de {len(config['tasks'])} tâches...")
    print()

    created_tasks = []
    for task_data in config["tasks"]:
        task = create_task(project_id, task_data)
        if task:
            created_tasks.append(task)

    print()
    print("="*80)
    print(f"✅ TERMINÉ: {len(created_tasks)}/{len(config['tasks'])} tâches créées")
    print("="*80)
    print()
    print(f"🔗 Projet ID: {project_id}")
    print(f"📊 Timeline estimée: {config['timeline']['estimated_total']}")
    print()
    print("💡 Voir le projet dans Archon UI:")
    print(f"   {ARCHON_API_URL}/projects/{project_id}")
    print()

if __name__ == "__main__":
    main()
