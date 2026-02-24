#!/bin/bash
# Add sizing label to overlay
# Usage: ./add-sizing-label.sh <app-path> <sizing>
# Example: ./add-sizing-label.sh apps/10-media/jellyfin small

set -e

if [ $# -lt 2 ]; then
    echo "Usage: $0 <app-path> <sizing>"
    echo "Example: $0 apps/10-media/jellyfin small"
    echo ""
    echo "Available sizings: micro, small, medium, large, xlarge"
    exit 1
fi

APP_PATH=$1
SIZING=$2
OVERLAY_PATH="$APP_PATH/overlays/prod"
KUSTOMIZATION="$OVERLAY_PATH/kustomization.yaml"

if [ ! -f "$KUSTOMIZATION" ]; then
    echo "Error: kustomization.yaml not found at $KUSTOMIZATION"
    exit 1
fi

# Validate sizing
VALID_SIZINGS="micro small medium large xlarge"
if ! echo "$VALID_SIZINGS" | grep -w "$SIZING" > /dev/null; then
    echo "Error: Invalid sizing '$SIZING'"
    echo "Valid options: $VALID_SIZINGS"
    exit 1
fi

# Check if metadata section exists
if grep -q "^metadata:" "$KUSTOMIZATION"; then
    # Check if labels already exist
    if grep -A2 "^metadata:" "$KUSTOMIZATION" | grep -q "vixens.io/sizing"; then
        echo "Warning: sizing label already exists in $KUSTOMIZATION"
        echo "Current configuration:"
        grep -A2 "^metadata:" "$KUSTOMIZATION" | grep "vixens.io/sizing" || true
        exit 1
    fi
    
    # Add label to existing metadata
    # This is a simple approach - may need manual adjustment for complex files
    echo "Adding sizing label to existing metadata..."
    sed -i "/^metadata:/a\\  labels:\\n    vixens.io/sizing: $SIZING" "$KUSTOMIZATION"
else
    # Add metadata section at the top (after first ---)
    echo "Creating metadata section with sizing label..."
    sed -i "1,/^---/ { /^---/ a\\metadata:\\n  labels:\\n    vixens.io/sizing: $SIZING\\n
}" "$KUSTOMIZATION"
fi

echo "✅ Added sizing label 'vixens.io/sizing: $SIZING' to $KUSTOMIZATION"
echo ""
echo "Preview of changes:"
grep -A3 "^metadata:" "$KUSTOMIZATION" || true
echo ""
echo "Don't forget to:"
echo "1. Review the changes (git diff)"
echo "2. Commit and push"
echo "3. Wait for ArgoCD sync"
echo "4. Restart the pod to apply new resources"
