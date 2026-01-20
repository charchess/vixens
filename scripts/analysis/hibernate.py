#!/usr/bin/env python3
import os
import sys
import subprocess
import argparse
import re
import yaml # Pour manipuler les fichiers YAML plus proprement

ARGOCD_KUST_PATH = "argocd/overlays/dev/kustomization.yaml"

def find_app_dir(app_name):
    """Recherche le répertoire de l'application dans apps/"""
    for root, dirs, files in os.walk("apps"):
        if os.path.basename(root) == app_name:
            if 'base' in dirs or 'kustomization.yaml' in files:
                return root
    return None

def manage_argocd_status(app_name, activate=True):
    """Décommente ou commente l'application dans le kustomization ArgoCD"""
    if not os.path.exists(ARGOCD_KUST_PATH):
        return False, f"ArgoCD kustomization non trouvé: {ARGOCD_KUST_PATH}"

    with open(ARGOCD_KUST_PATH, 'r') as f:
        content = f.read()

    commented_pattern = rf'#\s*-\s+apps/{app_name}\.yaml'
    active_pattern = rf'-\s+apps/{app_name}\.yaml'

    if activate:
        if re.search(commented_pattern, content):
            new_content = re.sub(commented_pattern, f'  - apps/{app_name}.yaml', content) # Ajout de l'indentation
            with open(ARGOCD_KUST_PATH, 'w') as f:
                f.write(new_content)
            return True, "Activée dans ArgoCD"
        elif re.search(active_pattern, content):
            return True, "Déjà active"
        else: # L'app n'est ni commentée ni active, on l'ajoute
            new_content = content.strip() + f"\n  - apps/{app_name}.yaml\n"
            with open(ARGOCD_KUST_PATH, 'w') as f:
                f.write(new_content)
            return True, "Ajoutée à ArgoCD"
    else:
        # Si on désactive (on ne le fait plus vraiment avec replicas:0)
        if re.search(active_pattern, content):
            new_content = re.sub(active_pattern, f'# {active_pattern}', content)
            with open(ARGOCD_KUST_PATH, 'w') as f:
                f.write(new_content)
            return True, "Désactivée dans ArgoCD"
        
    return True, "Inchangée"


def patch_kustomization(app_dir, replicas):
    """Ajoute ou modifie le patch de réplicas dans kustomization.yaml"""
    kust_path = os.path.join(app_dir, "overlays/dev/kustomization.yaml")
    os.makedirs(os.path.dirname(kust_path), exist_ok=True) # S'assurer que le dossier existe

    # Lire le contenu YAML, ou initialiser si vide/nouveau
    if os.path.exists(kust_path) and os.path.getsize(kust_path) > 0:
        with open(kust_path, 'r') as f:
            kust_data = yaml.safe_load(f)
            if kust_data is None: # Fichier vide
                kust_data = {'apiVersion': 'kustomize.config.k8s.io/v1beta1', 'kind': 'Kustomization'}
    else:
        kust_data = {'apiVersion': 'kustomize.config.k8s.io/v1beta1', 'kind': 'Kustomization'}
    
    app_name = os.path.basename(app_dir)

    # Assurer la présence de `resources` et y ajouter `../..base`
    if 'resources' not in kust_data or not isinstance(kust_data['resources'], list):
        kust_data['resources'] = []
    if '../..base' not in kust_data['resources']:
        kust_data['resources'].insert(0, '../..base') # S'assurer que base est toujours inclus
    
    # Préparer le patch
    new_patch = {
        'patch': f"""
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {app_name}
spec:
  replicas: {replicas}
""",
        'target': {
            'kind': 'Deployment',
            'name': app_name
        }
    }

    # Chercher un patch existant pour cette application
    found_existing = False
    if 'patches' not in kust_data:
        kust_data['patches'] = []
    
    for i, patch_item in enumerate(kust_data['patches']):
        if isinstance(patch_item, dict) and 'target' in patch_item and patch_item['target'].get('name') == app_name:
            kust_data['patches'][i] = new_patch
            found_existing = True
            break
            
    if not found_existing:
        kust_data['patches'].append(new_patch)

    with open(kust_path, 'w') as f:
        yaml.dump(kust_data, f, default_flow_style=False, sort_keys=False)
    
    return True, kust_path

def remove_patch(app_dir):
    """Supprime le patch de réplicas pour revenir à l'état base"""
    kust_path = os.path.join(app_dir, "overlays/dev/kustomization.yaml")
    if not os.path.exists(kust_path):
        return False, f"Fichier non trouvé: {kust_path}"

    with open(kust_path, 'r') as f:
        kust_data = yaml.safe_load(f)
        if kust_data is None: # Fichier vide
            return True, "Fichier vide, rien à supprimer"

    app_name = os.path.basename(app_dir)
    
    if 'patches' in kust_data and isinstance(kust_data['patches'], list):
        initial_patches_count = len(kust_data['patches'])
        kust_data['patches'] = [
            p for p in kust_data['patches']
            if not (isinstance(p, dict) and 'target' in p and p['target'].get('name') == app_name)
        ]
        
        if len(kust_data['patches']) < initial_patches_count:
            # Si on a supprimé des patches, vider le bloc patches s'il est vide
            if not kust_data['patches']:
                del kust_data['patches']
            
            with open(kust_path, 'w') as f:
                yaml.dump(kust_data, f, default_flow_style=False, sort_keys=False)
            return True, kust_path
        else:
            return True, "Aucun patch à supprimer"
    
    return True, "Aucun patch à supprimer"


def main():
    parser = argparse.ArgumentParser(description="Gère l'hibernation des applications")
    parser.add_argument("action", choices=["hibernate", "unhibernate", "list"])
    parser.add_argument("app", nargs="?")
    args = parser.parse_args()

    if args.action == "list":
        print("🔍 Recherche des applications hibernées (replicas: 0)...")
        hibernated = []
        for root, dirs, files in os.walk("apps"):
            if "kustomization.yaml" in files and "overlays/dev" in root:
                with open(os.path.join(root, "kustomization.yaml"), 'r') as f:
                    if "replicas: 0" in f.read():
                        hibernated.append(os.path.basename(os.path.dirname(os.path.dirname(root))))
        
        if not hibernated:
            print("✅ Aucune application hibernée via Git.")
        else:
            print("💤 Applications hibernées (replicas: 0) :")
            for app in sorted(hibernated):
                print(f"  - {app}")
        return

    if not args.app:
        print("❌ Erreur: Nom de l'application requis")
        sys.exit(1)

    app_dir = find_app_dir(args.app)
    if not app_dir:
        print(f"❌ Erreur: Application '{args.app}' non trouvée dans apps/")
        sys.exit(1)

    # Récupérer l'état actuel pour le message de commit
    try:
        current_kust_file = os.path.join(app_dir, "overlays/dev/kustomization.yaml")
        with open(current_kust_file, 'r') as f:
            current_kust_content = f.read()
        current_replicas_match = re.search(r'replicas:\s+(\d+)', current_kust_content)
        current_replicas = int(current_replicas_match.group(1)) if current_replicas_match else "N/A"
    except Exception:
        current_replicas = "N/A"


    if args.action == "hibernate":
        if current_replicas == 0:
            print(f"ℹ️  L'application {args.app} est déjà hibernée (replicas: 0).")
            return
        
        print(f"⚠️  Mise en veille de {args.app} (replicas: 0)...")
        # 1. Mettre replicas: 0
        success_patch, patch_path = patch_kustomization(app_dir, 0)
        # 2. S'assurer qu'elle est décommentée dans ArgoCD
        success_argocd, argocd_msg = manage_argocd_status(args.app, activate=True)
        
        if success_patch and success_argocd:
            print(f"✅ Patch appliqué: {patch_path}")
            print(f"✅ Statut ArgoCD: {argocd_msg}")
            
            commit_msg = f"chore({args.app}): hibernate app (replicas=0)"
            subprocess.run(["git", "add", patch_path, ARGOCD_KUST_PATH])
            subprocess.run(["git", "commit", "-m", commit_msg])
            print(f"🔄 Commit effectué: {commit_msg}")
            print("💡 N'oubliez pas de faire 'git push origin main' pour appliquer.")
        else:
            print(f"❌ Erreur patch: {patch_path}, Erreur ArgoCD: {argocd_msg}")

    elif args.action == "unhibernate":
        if current_replicas == 1:
            print(f"ℹ️  L'application {args.app} est déjà active (replicas: 1).")
            return

        print(f"🚀 Réactivation de {args.app} (replicas: 1)...")
        # 1. S'assurer qu'elle est décommentée dans ArgoCD
        success_argocd, argocd_msg = manage_argocd_status(args.app, activate=True)
        # 2. Mettre replicas: 1
        success_patch, patch_path = patch_kustomization(app_dir, 1)
        
        if success_patch and success_argocd:
            print(f"✅ Patch appliqué: {patch_path}")
            print(f"✅ Statut ArgoCD: {argocd_msg}")
            
            commit_msg = f"chore({args.app}): unhibernate app (replicas=1)"
            subprocess.run(["git", "add", patch_path, ARGOCD_KUST_PATH])
            subprocess.run(["git", "commit", "-m", commit_msg])
            print(f"🔄 Commit effectué: {commit_msg}")
            print("💡 N'oubliez pas de faire 'git push origin main' pour appliquer.")
        else:
            print(f"❌ Erreur patch: {patch_path}, Erreur ArgoCD: {argocd_msg}")

if __name__ == "__main__":
    main()
