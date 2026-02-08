#!/usr/bin/env python3
"""
Generate APP-VERSIONS.md inventory from deployed applications.

Extracts version information from:
- Kubernetes deployment images
- Helm chart versions
- Application labels/annotations
"""

import subprocess
import json
import re
from datetime import datetime
from collections import defaultdict
import sys
import os

# Add lib to path
sys.path.append(os.path.join(os.path.dirname(__file__), '..', 'lib'))
from report_utils import save_markdown_table


def get_deployments():
    """Get all deployments from cluster."""
    result = subprocess.run(
        ["kubectl", "get", "deployments", "-A", "-o", "json"],
        capture_output=True,
        text=True
    )

    if result.returncode != 0:
        print(f"❌ Error fetching deployments: {result.stderr}")
        return []

    return json.loads(result.stdout).get('items', [])


def get_statefulsets():
    """Get all statefulsets from cluster."""
    result = subprocess.run(
        ["kubectl", "get", "statefulsets", "-A", "-o", "json"],
        capture_output=True,
        text=True
    )

    if result.returncode != 0:
        print(f"❌ Error fetching statefulsets: {result.stderr}")
        return []

    return json.loads(result.stdout).get('items', [])


def extract_version(image):
    """Extract version from container image."""
    # Format: registry/image:version or image:version
    if ':' not in image:
        return 'latest'

    version = image.split(':')[-1]

    # Clean up common prefixes
    version = re.sub(r'^v([0-9])', r'\1', version)  # Remove leading 'v'

    return version


def get_app_name(metadata):
    """Extract app name from metadata."""
    labels = metadata.get('labels', {})

    # Try various label keys
    for key in ['app.kubernetes.io/name', 'app', 'name']:
        if key in labels:
            return labels[key]

    # Fallback to metadata name
    return metadata.get('name', 'unknown')


def get_helm_chart_version(metadata):
    """Extract Helm chart version from labels."""
    labels = metadata.get('labels', {})
    annotations = metadata.get('annotations', {})

    # Check Helm labels
    if 'helm.sh/chart' in labels:
        chart = labels['helm.sh/chart']
        # Format: chartname-version
        parts = chart.rsplit('-', 1)
        if len(parts) == 2:
            return parts[1]

    # Check ArgoCD annotations
    if 'argocd.argoproj.io/tracking-id' in annotations:
        tracking = annotations['argocd.argoproj.io/tracking-id']
        # Try to extract version
        match = re.search(r':([^:]+)$', tracking)
        if match:
            return match.group(1)

    return None


def generate_app_versions():
    """Generate application version inventory."""
    print("📦 Generating application version inventory...")

    deployments = get_deployments()
    statefulsets = get_statefulsets()

    all_resources = deployments + statefulsets

    # Group by namespace and app
    apps = {}

    for resource in all_resources:
        metadata = resource.get('metadata', {})
        namespace = metadata.get('namespace', 'default')
        app_name = get_app_name(metadata)

        # Skip system namespaces
        if namespace.startswith('kube-'):
            continue

        spec = resource.get('spec', {})
        template = spec.get('template', {})
        containers = template.get('spec', {}).get('containers', [])

        if not containers:
            continue

        # Get first container image version
        image = containers[0].get('image', '')
        image_version = extract_version(image)

        # Try to get Helm chart version
        chart_version = get_helm_chart_version(metadata)

        # Use chart version if available, otherwise image version
        version = chart_version or image_version

        # Store in apps dict
        key = f"{namespace}/{app_name}"
        if key not in apps:
            apps[key] = {
                'namespace': namespace,
                'app': app_name,
                'version': version,
                'image': image
            }

    # Sort by namespace, then app name
    sorted_apps = sorted(apps.values(), key=lambda x: (x['namespace'], x['app']))

    print(f"   ✅ Found {len(sorted_apps)} applications")

    return sorted_apps


def save_report(apps, output_file):
    """Save version inventory to markdown."""

    content = "# Application Version Inventory\n\n"
    content += f"**Last Updated:** {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n"
    content += f"**Total Applications:** {len(apps)}\n\n"

    # Build table
    rows = []
    for app in apps:
        rows.append({
            'Application': f"**{app['app']}**",
            'Namespace': app['namespace'],
            'Version': app['version'],
            'Image': app['image']
        })

    content += save_markdown_table(['Application', 'Namespace', 'Version', 'Image'], rows)
    content += "\n"

    # Write file
    os.makedirs(os.path.dirname(output_file), exist_ok=True)
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(content)

    print(f"✅ Application versions saved: {output_file}")


def main():
    import argparse
    parser = argparse.ArgumentParser(description="Generate application version inventory")
    parser.add_argument("--output", default="docs/reports/APP-VERSIONS.md", help="Output file")
    args = parser.parse_args()

    apps = generate_app_versions()
    save_report(apps, args.output)


if __name__ == "__main__":
    main()
