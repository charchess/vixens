#!/usr/bin/env just --justfile

set shell := ["bash", "-euo", "pipefail", "-c"]

default:
    @just --list

# Local validation of the GitOps repository.
lint:
    find apps argocd \( -name '*.yaml' -o -name '*.yml' \)       ! -path '*/charts/*'       ! -path '*/crds/*'       ! -name 'argocd-crds.yaml'       ! -name 'argocd-install.yaml'       ! -name 'manifests.yaml'       ! -name 'servicemonitors.yaml'       ! -name 'upstream.yaml'       ! -name 'poolers.yaml'       -print0 | xargs -0 -r yamllint -c yamllint-config.yml

validate:
    just lint
    python3 scripts/validation/validate-helm-values.py argocd/
    bash scripts/validation/validate-kustomization-refs.sh
    python3 scripts/validation/validate-repository.py

# GitHub Issues / PR helpers.
gh-resume:
    @echo "Draft PRs:"
    @gh pr list --state open --search 'is:draft'
    @echo
    @echo "In-progress issues:"
    @gh issue list --state open --label 'status:in-progress'

gh-tasks:
    gh issue list --state open --limit 50

gh-start issue:
    #!/usr/bin/env bash
    number="{{issue}}"
    title="$(gh issue view "$number" --json title --jq .title)"
    slug="$(printf '%s' "$title" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g;s/^-|-$//g' | cut -c1-40)"
    branch="feat/$number-$slug"
    git switch main
    git pull --ff-only origin main
    git switch -c "$branch"
    git push -u origin "$branch"
    gh pr create --draft --base main --head "$branch" --title "$title" --body "Closes #$number"
    gh issue edit "$number" --add-label status:in-progress

gh-done pr:
    gh pr ready "{{pr}}"
    gh pr merge "{{pr}}" --squash --auto --delete-branch

# Read-only wait helper. It never changes workload desired state.
wait-argocd app:
    #!/usr/bin/env bash
    app="{{app}}"
    timeout 600 bash -c '
      while true; do
        sync=$(kubectl -n argocd get application "'"$app"'" -o jsonpath="{.status.sync.status}" 2>/dev/null || true)
        health=$(kubectl -n argocd get application "'"$app"'" -o jsonpath="{.status.health.status}" 2>/dev/null || true)
        printf "%s: sync=%s health=%s\n" "'"$app"'" "$sync" "$health"
        [ "$sync" = Synced ] && [ "$health" = Healthy ] && exit 0
        sleep 5
      done
    '

# Trigger the canonical production promotion workflow.
promote-prod version:
    gh workflow run promote-prod.yaml -f version="{{version}}"
