#!/usr/bin/env python3
import os
import sys
import subprocess
import argparse
import re

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

    # Pattern pour l'application
    commented_pattern = rf'#\s*-\s+apps/{app_name}\.yaml'
    active_pattern = rf'-\s+apps/{app_name}\.yaml'

    if activate:
        if re.search(commented_pattern, content):
            new_content = re.sub(commented_pattern, f'- apps/{app_name}.yaml', content)
            with open(ARGOCD_KUST_PATH, 'w') as f:
                f.write(new_content)
            return True, "Activée dans ArgoCD"
    else:
        # Note: On ne commente plus, on utilise replicas: 0, 
        # mais je garde la fonction pour la flexibilité
        pass
    
    return True, "Déjà active ou inchangée"

def patch_kustomization(app_dir, replicas):
    """Ajoute ou modifie le patch de réplicas dans kustomization.yaml"""
    kust_path = os.path.join(app_dir, "overlays/dev/kustomization.yaml")
    if not os.path.exists(kust_path):
        # Créer l'overlay dev s'il n'existe pas (cas rare mais possible)
        os.makedirs(os.path.dirname(kust_path), exist_ok=True)
        with open(kust_path, 'w') as f:
            f.write(f"apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n  - ../../base\n")

    with open(kust_path, 'r') as f:
        content = f.read()

    app_name = os.path.basename(app_dir)
    
    # Template du patch propre
    patch_block = f"""
patches:
  - patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: {app_name}
      spec:
        replicas: {replicas}"""

    # Si le patch existe déjà, on le remplace
    if f"name: {app_name}" in content and "replicas:" in content:
        new_content = re.sub(
            r'(name:\s+' + app_name + r'.*?replicas:\s+)\d+',
            r'\1' + str(replicas),
            content,
            flags=re.DOTALL
        )
    elif "patches:" in content:
        # On ajoute à la fin (simple)
        new_content = content.strip() + f"\n  - patch: |-\n      apiVersion: apps/v1\n      kind: Deployment\n      metadata:\n        name: {app_name}\n      spec:\n        replicas: {replicas}\n"
    else:
        new_content = content.strip() + "\n\n" + patch_block + "\n"

    with open(kust_path, 'w') as f:
        f.write(new_content)
    
    return True, kust_path

def main():
    parser = argparse.ArgumentParser(description="Gère l'hibernation des applications")
    parser.add_argument("action", choices=["hibernate", "unhibernate", "list"])
    parser.add_argument("app", nargs="?")
    args = parser.parse_args()

    if args.action == "list":
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
        sys.exit(1)

    app_dir = find_app_dir(args.app)
    if not app_dir:
        print(f"❌ Erreur: Application '{args.app}' non trouvée")
        sys.exit(1)

    if args.action == "hibernate":
        # 1. Décommenter dans ArgoCD
        manage_argocd_status(args.app, activate=True)
        # 2. Mettre replicas: 0
        success, path = patch_kustomization(app_dir, 0)
        if success:
            print(f"✅ {args.app} hibernée (replicas: 0)")
            subprocess.run(["git", "add", path, ARGOCD_KUST_PATH])
            subprocess.run(["git", "commit", "-m", f"chore({args.app}): hibernate app (replicas=0)"])

    elif args.action == "unhibernate":
        # 1. S'assurer qu'elle est décommentée
        manage_argocd_status(args.app, activate=True)
        # 2. Mettre replicas: 1
        success, path = patch_kustomization(app_dir, 1)
        if success:
            print(f"🚀 {args.app} réactivée (replicas: 1)")
            subprocess.run(["git", "add", path, ARGOCD_KUST_PATH])
            subprocess.run(["git", "commit", "-m", f"chore({args.app}): unhibernate app (replicas=1)"])

if __name__ == "__main__":
    main()