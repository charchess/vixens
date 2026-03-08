#!/usr/bin/env bash
***REMOVED***
# validate-kustomization-refs.sh
# Validates that all resources referenced in kustomization.yaml files exist
***REMOVED***
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

ERRORS=0
WARNINGS=0

log_error() {
    echo -e "${RED}❌ ERROR:${NC} $1"
    ((ERRORS++))
}

log_warning() {
    echo -e "${YELLOW}⚠️ WARNING:${NC} $1"
    ((WARNINGS++))
}

log_ok() {
    echo -e "${GREEN}✅${NC} $1"
}

# Find all kustomization.yaml files
find_kustomizations() {
    find apps argocd -name "kustomization.yaml" -type f 2>/dev/null
}

# Validate resources exist
validate_resources() {
    local kustomization_file="$1"
    local kustomization_dir
    kustomization_dir=$(dirname "$kustomization_file")
    
    # Extract resources using yq
    local resources
    resources=$(yq eval '.resources[]? // empty' "$kustomization_file" 2>/dev/null || true)
    
    for resource in $resources; do
        local resource_path="$kustomization_dir/$resource"
        
        # Skip remote resources (URLs)
        if [[ "$resource" =~ ^https?:// ]] || [[ "$resource" =~ ^git@ ]]; then
            continue
        fi
        
        # Check if it's a file or directory
        if [[ -f "$resource_path" ]] || [[ -d "$resource_path" ]]; then
            : # OK
        else
            log_error "Missing resource: $resource (referenced in $kustomization_file)"
        fi
    done
}

# Validate patches exist
validate_patches() {
    local kustomization_file="$1"
    local kustomization_dir
    kustomization_dir=$(dirname "$kustomization_file")
    
    # Extract patch files from various patch formats
    local patches
    patches=$(yq eval '
        (.patches[]?.path // empty),
        (.patchesStrategicMerge[]? // empty),
        (.patchesJson6902[]?.path // empty)
    ' "$kustomization_file" 2>/dev/null | grep -v '^$' || true)
    
    for patch in $patches; do
        # Skip inline patches (no path)
        if [[ -z "$patch" ]] || [[ "$patch" == "null" ]]; then
            continue
        fi
        
        local patch_path="$kustomization_dir/$patch"
        
        if [[ ! -f "$patch_path" ]]; then
            log_error "Missing patch file: $patch (referenced in $kustomization_file)"
        fi
    done
}

# Validate configMapGenerator files exist
validate_configmap_files() {
    local kustomization_file="$1"
    local kustomization_dir
    kustomization_dir=$(dirname "$kustomization_file")
    
    # Extract files from configMapGenerator
    local files
    files=$(yq eval '.configMapGenerator[]?.files[]? // empty' "$kustomization_file" 2>/dev/null || true)
    
    for file_entry in $files; do
        # Handle key=value format (key=path/to/file)
        local file_path
        if [[ "$file_entry" =~ = ]]; then
            file_path="${file_entry#*=}"
        else
            file_path="$file_entry"
        fi
        
        local full_path="$kustomization_dir/$file_path"
        
        if [[ ! -f "$full_path" ]]; then
            log_error "Missing configMap file: $file_path (referenced in $kustomization_file)"
        fi
    done
}

# Validate components exist
validate_components() {
    local kustomization_file="$1"
    local kustomization_dir
    kustomization_dir=$(dirname "$kustomization_file")
    
    local components
    components=$(yq eval '.components[]? // empty' "$kustomization_file" 2>/dev/null || true)
    
    for component in $components; do
        local component_path="$kustomization_dir/$component"
        
        # Skip remote components
        if [[ "$component" =~ ^https?:// ]] || [[ "$component" =~ ^git@ ]]; then
            continue
        fi
        
        if [[ ! -d "$component_path" ]] || [[ ! -f "$component_path/kustomization.yaml" ]]; then
            log_error "Missing component: $component (referenced in $kustomization_file)"
        fi
    done
}

# Main validation loop
main() {
    echo "🔍 Validating kustomization.yaml references..."
    echo ""
    
    local kustomization_files
    kustomization_files=$(find_kustomizations)
    
    if [[ -z "$kustomization_files" ]]; then
        log_warning "No kustomization.yaml files found"
        exit 0
    fi
    
    local count=0
    while IFS= read -r kustomization_file; do
        ((count++))
        echo "📋 Checking: $kustomization_file"
        
        validate_resources "$kustomization_file"
        validate_patches "$kustomization_file"
        validate_configmap_files "$kustomization_file"
        validate_components "$kustomization_file"
    done <<< "$kustomization_files"
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📊 Summary: $count kustomization files checked"
    
    if [[ $ERRORS -gt 0 ]]; then
        echo -e "${RED}❌ $ERRORS error(s) found${NC}"
        exit 1
    elif [[ $WARNINGS -gt 0 ]]; then
        echo -e "${YELLOW}⚠️ $WARNINGS warning(s) found${NC}"
        exit 0
    else
        echo -e "${GREEN}✅ All references valid${NC}"
        exit 0
    fi
}

main "$@"
