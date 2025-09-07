# PROCEDURES.md  
Repo : `charchess/vixens`  
Objectif : promouvoir une modification « dev » → « prod » uniquement via Pull-Request (PR) et en ligne de commande.

---

## 0. Prérequis locaux

| Outil | Installation rapide | Vérification |
|-------|---------------------|--------------|
| `git` | déjà présent | `git --version` |
| GitHub CLI (`gh`) | `brew install gh` (mac) ou `sudo apt install gh` | `gh --version` |
| Accès écriture au repo | générer un **Personal Access Token** (classic) avec scopes `repo` + `workflow` | `gh auth login` |

---

## 1. Travailler sur la branche `dev`

```bash
git clone https://github.com/charchess/vixens.git
cd vixens
git checkout dev                     # branche de développement
# …éditer vos YAML…
git add -A
git commit -m "feat: nouvelle app nginx"
git push origin dev

Dès le push :

    ArgoCD dev synchronise automatiquement (targetRevision: dev)
    Vérifiez le rendu : kubectl --context=vixens-dev get pod

2. Ouvrir une Pull Request (CLI)
bash
Copy

gh pr create --title "Promote nginx to prod" \
             --body "Validé sur le cluster **vixens-dev** :
- Pods Ready ✔
- Ingress OK ✔" \
             --base main \
             --head dev

Sortie : https://github.com/charchess/vixens/pull/42
3. Suivre les checks obligatoires
bash
Copy

gh pr checks 42 --watch   # attend que la CI soit verte

4. Approuver la PR (review obligatoire)
4.a Vous êtes le relecteur
bash
Copy

gh pr review 42 --approve --body "LGTM, tests passés sur dev"

4.b Vous êtes le demandeur mais vous n’avez pas le droit de vous approuver vous-même → passez la main à un collègue ou désactivez temporairement la règle « require 1 approving review ».
5. Merger la PR
Une seule commande après approbation :
bash
Copy

gh pr merge 42 --squash --delete-branch

Résultat :

    branche dev supprimée sur le remote
    commit squashé dans main
    ArgoCD prod déclenche le sync (manuel ou auto selon votre config)

6. Ré-initialiser dev pour le prochain ticket
bash
Copy

git checkout main
git pull origin main
git checkout -b dev
git push --set-upstream origin dev

7. Commandes rapides « Cheat-Sheet »
Table
Copy
Action	Commande
Lister les PR ouvertes	gh pr list
Voir détail + diff	gh pr view 42 --web
Commenter sans approuver	gh pr review 42 --comment -b "nit: typo"
Forcer re-run checks	gh pr checks 42 --watch
Merge automatique dès checks OK	gh pr merge 42 --auto --squash
8. Sécurité & bonnes pratiques (phase 2)

    Branch protection main : activée (Settings → Branches)
    CI : kubeconform + helm unittest + policy kyverno
    Secrets : remplacer les YAML en clair par Sealed Secrets ou External Secrets Operator
    Auto-sync : désactivé sur le projet prod (sync manuel via ArgoCD UI ou CLI)

9. En cas de problème
Table
Copy
Symptôme	Solution
« Review required »	attendre une approbation externe ou modifier les règles
CI rouge	gh pr view 42 --checks puis corriger sur dev
Conflits de merge	git checkout dev && git rebase main && git push -f
Mauvaise branche	fermer la PR et la re-créer avec --base main --head dev
