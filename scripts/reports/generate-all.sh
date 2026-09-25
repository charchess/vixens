#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

REPORT_DIR="docs/reports"
TRASH_DIR="$REPORT_DIR/trash/$(date +%Y%m%d-%H%M%S)"
DEV_KUBECONFIG="${VIXENS_DEV_KUBECONFIG:-$ROOT_DIR/.secrets/dev/kubeconfig-dev}"
PROD_KUBECONFIG="${VIXENS_PROD_KUBECONFIG:-$ROOT_DIR/.secrets/prod/kubeconfig-prod}"

mkdir -p "$REPORT_DIR" "$TRASH_DIR"

generate_state() {
  local env="$1"
  local kubeconfig="$2"

  if [[ ! -f "$kubeconfig" ]]; then
    echo "[skip] $env: kubeconfig absent ($kubeconfig)"
    return 0
  fi

  echo "[state] $env"
  KUBECONFIG="$kubeconfig" python3 scripts/reports/generate_actual_state_vpa.py \
    --env "$env" \
    --output "$REPORT_DIR/STATE-ACTUAL-${env}.md" \
    --json-output "$REPORT_DIR/STATE-${env}.json"
}

echo "== Vixens reports =="

generate_state dev "$DEV_KUBECONFIG"
generate_state prod "$PROD_KUBECONFIG"

if [[ -f "$REPORT_DIR/STATE-ACTUAL-prod.md" ]]; then
  cp "$REPORT_DIR/STATE-ACTUAL-prod.md" "$REPORT_DIR/STATE-ACTUAL.md"
fi

if [[ -f "$PROD_KUBECONFIG" ]]; then
  echo "[versions] applications"
  KUBECONFIG="$PROD_KUBECONFIG" python3 scripts/reports/generate_app_versions.py \
    --output "$REPORT_DIR/APP-VERSIONS.md"
fi

echo "[quality] YAML / repository"
python3 scripts/reports/generate_lint_report.py \
  --paths apps argocd \
  --output "$REPORT_DIR/LINT-REPORT.md" \
  --fail-threshold 0 || true

if [[ -f "$REPORT_DIR/STATE-ACTUAL-dev.md" && -f "$REPORT_DIR/STATE-DESIRED.md" ]]; then
  echo "[conformity] dev"
  python3 scripts/reports/conformity_checker.py \
    --actual "$REPORT_DIR/STATE-ACTUAL-dev.md" \
    --desired "$REPORT_DIR/STATE-DESIRED.md" \
    --output "$REPORT_DIR/CONFORMITY-dev.md"
fi

if [[ -f "$REPORT_DIR/STATE-ACTUAL-prod.md" && -f "$REPORT_DIR/STATE-DESIRED.md" ]]; then
  echo "[conformity] prod"
  python3 scripts/reports/conformity_checker.py \
    --actual "$REPORT_DIR/STATE-ACTUAL-prod.md" \
    --desired "$REPORT_DIR/STATE-DESIRED.md" \
    --output "$REPORT_DIR/CONFORMITY-prod.md"
fi

if [[ -f "$REPORT_DIR/STATE-dev.json" && -f "$REPORT_DIR/STATE-prod.json" ]]; then
  echo "[status] consolidated dashboard"
  python3 scripts/reports/generate_status_report.py \
    --dev-state "$REPORT_DIR/STATE-dev.json" \
    --prod-state "$REPORT_DIR/STATE-prod.json" \
    --dev-conformity "$REPORT_DIR/CONFORMITY-dev.md" \
    --prod-conformity "$REPORT_DIR/CONFORMITY-prod.md" \
    --output "$REPORT_DIR/STATUS.md"
fi

if [[ -f "$PROD_KUBECONFIG" ]]; then
  echo "[management] report"
  KUBECONFIG="$PROD_KUBECONFIG" python3 scripts/reports/generate_management_report.py \
    --output "$REPORT_DIR/MANAGEMENT-REPORT.md"
fi

# Historical dated reports at the report root are archived; existing trash is untouched.
find "$REPORT_DIR" -maxdepth 1 -type f -name '20[0-9][0-9]-*.md' -print0 | \
  while IFS= read -r -d '' file; do
    mv "$file" "$TRASH_DIR/"
  done

rm -f "$REPORT_DIR/STATE-dev.json" "$REPORT_DIR/STATE-prod.json"

# Remove an empty archive directory so routine runs do not create noise.
rmdir "$TRASH_DIR" 2>/dev/null || true

echo "== Reports complete =="
echo "Output: $REPORT_DIR"
