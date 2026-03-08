#!/bin/bash
***REMOVED***
# VIXENS REPOSITORY HISTORY CLEANUP
***REMOVED***
# 
# ⚠️  WARNING: This script rewrites git history!
#     - All collaborators must re-clone after running
#     - Requires force push to remote
#     - Make a backup first!
#
# Usage:
#   1. Review this script
#   2. Make backup: cp -r vixens vixens-backup
#   3. Run: ./scripts/cleanup-history.sh
#   4. Force push: git push --force --all
#   5. Notify collaborators to re-clone
#
***REMOVED***

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}=== VIXENS HISTORY CLEANUP ===${NC}"
echo ""

# Check for required tools
if ! command -v git-filter-repo &> /dev/null; then
    echo -e "${RED}ERROR: git-filter-repo not installed${NC}"
    echo "Install with: pip install git-filter-repo"
    exit 1
fi

# Confirm backup
echo -e "${YELLOW}⚠️  This will rewrite git history!${NC}"
echo "Have you made a backup? (cp -r vixens vixens-backup)"
read -p "Type 'yes' to continue: " confirm
if [ "$confirm" != "yes" ]; then
    echo "Aborted."
    exit 1
fi

***REMOVED***
# STEP 1: Delete files that should never have been committed
***REMOVED***
echo ""
echo -e "${GREEN}Step 1: Removing sensitive files from history...${NC}"

FILES_TO_DELETE=(
    # Old secret files (before Infisical migration)
    "overlays/prod/argocd/dex-github-secret.yaml"
    "frigate-config-prod.yml"
    "synology-csi-secret.yaml"
    "base/synology-csi/client-info-secret.yaml"
    "base/synology-csi/secret.yaml"
    "base/synology-csi/configs.yaml"
    "base/synology-csi/csi.yaml"
    
    # Terraform state files (contain secrets)
    "terraform/environments/dev/terraform.tfstate*"
    "terraform/environments/dev/current_state.json"
    
    # Local config with tokens
    ".gemini/settings.json"
)

for file in "${FILES_TO_DELETE[@]}"; do
    echo "  Removing: $file"
done

# Build the filter-repo command
PATHS_ARGS=""
for file in "${FILES_TO_DELETE[@]}"; do
    PATHS_ARGS="$PATHS_ARGS --path '$file'"
done

# Note: Uncomment to actually run
# eval "git filter-repo --invert-paths $PATHS_ARGS --force"

***REMOVED***
# STEP 2: Replace secrets in files that need to stay
***REMOVED***
echo ""
echo -e "${GREEN}Step 2: Creating replacement patterns...${NC}"

# Create replacements file for git-filter-repo
cat > /tmp/secret-replacements.txt << 'REPLACEMENTS'
REPLACEMENT==>REPLACEMENT

***REMOVED***
literal:REDACTED_GITHUB_PAT==>REDACTED_GITHUB_PAT

***REMOVED***
regex:access_token:\s*[a-zA-Z0-9]{14,}==>access_token: REDACTED_FREE_MOBILE_TOKEN

***REMOVED***
literal:REDACTED_SYNOLOGY_PWD==>REDACTED_SYNOLOGY_PASSWORD
literal:ugRqZu6zahkFPTfjfu2nu548oNmdW4kFNXXVaU9o4ac8cVdRoQ==>REDACTED_SYNOLOGY_PASSWORD

***REMOVED***
literal:REDACTED_ROBUSTA_KEY==>REDACTED_ROBUSTA_KEY

# Infisical client secrets (UUIDs stay, but secrets get redacted)
regex:[a-f0-9]{64}==>REDACTED_INFISICAL_SECRET

# TOOLS.md passwords
literal:REDACTED_API_PASSWORD==>REDACTED_PASSWORD
literal:REDACTED_PASSWORD==>REDACTED_PASSWORD
literal:REDACTED_PASSWORD==>REDACTED_PASSWORD
literal:REDACTED_PASSWORD==>REDACTED_PASSWORD

# Frigate password
literal:REDACTED_FRIGATE_PWD==>REDACTED_FRIGATE_PASSWORD

***REMOVED***
literal:REDACTED_ARGOCD_PWD==>REDACTED_ARGOCD_PASSWORD
REPLACEMENTS

echo "  Replacement patterns saved to /tmp/secret-replacements.txt"

***REMOVED***
# STEP 3: Run the cleanup
***REMOVED***
echo ""
echo -e "${YELLOW}Step 3: Ready to clean${NC}"
echo ""
echo "Commands to run manually (review first!):"
echo ""
echo "  # Delete sensitive files from history"
echo "  git filter-repo --invert-paths \\"
for file in "${FILES_TO_DELETE[@]}"; do
    echo "    --path '$file' \\"
done
echo "    --force"
echo ""
echo "  # Replace secrets in remaining files"
echo "  git filter-repo --replace-text /tmp/secret-replacements.txt --force"
echo ""
echo "  # Force push to remote"
echo "  git push --force --all"
echo "  git push --force --tags"
echo ""
echo -e "${RED}⚠️  After force push, ALL collaborators must re-clone!${NC}"
echo ""
echo "Done. Review and run commands manually."
