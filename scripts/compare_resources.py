import json
import os
import re
import yaml

def parse_vpa(file_path):
    with open(file_path, 'r') as f:
        data = json.load(f)
    
    recs = {}
    for item in data.get('items', []):
        ns = item['metadata']['namespace']
        name = item['spec']['targetRef']['name']
        container_recs = item.get('status', {}).get('recommendation', {}).get('containerRecommendations', [])
        
        for cr in container_recs:
            c_name = cr['containerName']
            key = f"{ns}/{name}/{c_name}"
            recs[key] = {
                'target_cpu': cr['target'].get('cpu'),
                'target_mem': cr['target'].get('memory')
            }
    return recs

def get_prod_resources():
    # Scan apps directory for prod resources
    prod_res = {}
    for root, dirs, files in os.walk('apps'):
        if 'overlays/prod' in root:
            for file in files:
                if file.endswith('.yaml') or file.endswith('.yml'):
                    path = os.path.join(root, file)
                    try:
                        with open(path, 'r') as f:
                            content = f.read()
                            # Look for resources: section in Deployment/StatefulSet
                            # This is a bit simplified, but should catch most patches
                            if 'resources:' in content:
                                # We try to identify the app name from path or content
                                app_name = root.split(os.sep)[2]
                                # Store path for manual check if needed
                                prod_res[app_name] = path
                    except:
                        pass
    return prod_res

# I will focus on the comparison logic in the next step by combining 
# the precise VPA values with the scanned manifest values.

vpa_data = parse_vpa('/tmp/vpa_prod.json')
print(f"Loaded {len(vpa_data)} container recommendations from VPA.")

# Mapping table construction...
# (Full script will be executed to generate the markdown)
