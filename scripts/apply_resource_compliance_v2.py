import json
import os
import yaml
import re

# Load Data
with open('/tmp/argo_apps_full.json', 'r') as f: argo_data = json.load(f)
with open('/tmp/vpa_prod.json', 'r') as f: vpa_data = json.load(f)

# Helpers
def parse_size(size_str):
    if not size_str: return 0
    size_str = str(size_str).strip().replace('"', '').replace("'", "")
    units = {"k": 10**3, "M": 10**6, "G": 10**9, "T": 10**12, "Ki": 2**10, "Mi": 2**20, "Gi": 2**30, "Ti": 2**40}
    res = re.match(r'^(\d+)([a-zA-Z]+)?$', size_str)
    if not res: return 0
    val, unit = res.groups()
    return int(val) * units.get(unit, 1)

def format_mem(bytes):
    return f"{int(bytes/1024/1024)}Mi"

def format_cpu(millis):
    return f"{int(millis)}m"

# Build VPA Map strictly by "namespace/deployment_name"
vpa_by_deployment = {}
for item in vpa_data.get('items', []):
    ns = item['metadata']['namespace']
    target = item['spec']['targetRef']['name'] # Deployment name
    
    key = f"{ns}/{target}"
    if key not in vpa_by_deployment:
        vpa_by_deployment[key] = []
    
    recs = item.get('status', {}).get('recommendation', {}).get('containerRecommendations', [])
    for r in recs:
        vpa_by_deployment[key].append({
            'name': r['containerName'],
            'cpu_target': int(parse_size(r['target'].get('cpu', '0').replace('m','')) if 'm' in r['target'].get('cpu', '0') else float(r['target'].get('cpu', '0'))*1000),
            'mem_target': parse_size(r['target'].get('memory', '0')),
            'cpu_upper': int(parse_size(r['upperBound'].get('cpu', '0').replace('m','')) if 'm' in r['upperBound'].get('cpu', '0') else float(r['upperBound'].get('cpu', '0'))*1000),
            'mem_upper': parse_size(r['upperBound'].get('memory', '0'))
        })

# Default Policies
DEFAULTS = {
    'infra': {'req_cpu': 50, 'req_mem': 64*1024*1024, 'lim_cpu': 500, 'lim_mem': 256*1024*1024},
    'app':   {'req_cpu': 20, 'req_mem': 128*1024*1024, 'lim_cpu': 500, 'lim_mem': 512*1024*1024},
    'db':    {'req_cpu': 100, 'req_mem': 256*1024*1024, 'lim_cpu': 1000, 'lim_mem': 512*1024*1024}
}

# Process Apps
for app in argo_data['items']:
    name = app['metadata']['name']
    ns = app['spec']['destination']['namespace']
    
    # Identify Source Path
    sources = []
    if 'source' in app['spec']: sources.append(app['spec']['source'])
    if 'sources' in app['spec']: sources.extend(app['spec']['sources'])
    
    path = None
    for s in sources:
        p = s.get('path')
        if p and os.path.exists(p):
            path = p
            break
    
    if not path: continue

    # Determine containers to patch
    containers_to_patch = []
    
    # Check strict match in VPA
    # Note: ArgoCD app name isn't always Deployment name.
    # But usually it is. We try precise match.
    # In VPA list, some names have prefixes like "goldilocks-". We need to look at targetRef name.
    
    vpa_key = f"{ns}/{name}"
    
    # Handle known divergences or prefixes if necessary
    # (Here we assume App Name = Deployment Name, which covers 90% cases)
    
    if vpa_key in vpa_by_deployment:
        # We have real data!
        for c in vpa_by_deployment[vpa_key]:
            # Smart sizing logic
            req_cpu = max(c['cpu_target'], 10)
            req_mem = max(c['mem_target'], 32*1024*1024)
            lim_cpu = max(c['cpu_upper'], c['cpu_target']*4, 500)
            lim_mem = max(c['mem_upper'], c['mem_target']*1.5, 128*1024*1024)
            
            containers_to_patch.append({
                'name': c['name'],
                'req_cpu': req_cpu, 'req_mem': req_mem,
                'lim_cpu': lim_cpu, 'lim_mem': lim_mem
            })
    else:
        # Fallback to Defaults
        c_type = 'app'
        if 'redis' in name or 'postgres' in name or 'db' in name: c_type = 'db'
        elif 'argocd' in name or 'traefik' in name or 'cert-manager' in name or 'kube-system' in ns: c_type = 'infra'
        
        d = DEFAULTS[c_type]
        # We patch a container with the same name as the app
        # This might fail if the container is named differently, but it's the best guess without parsing deployment.yaml
        containers_to_patch.append({
            'name': name,
            'req_cpu': d['req_cpu'], 'req_mem': d['req_mem'],
            'lim_cpu': d['lim_cpu'], 'lim_mem': d['lim_mem']
        })

    # Generate Patch YAML
    patch_file = os.path.join(path, 'resources-patch.yaml')
    
    c_specs = []
    for c in containers_to_patch:
        c_specs.append({
            'name': c['name'],
            'resources': {
                'requests': {
                    'cpu': format_cpu(c['req_cpu']),
                    'memory': format_mem(c['req_mem'])
                },
                'limits': {
                    'cpu': format_cpu(c['lim_cpu']),
                    'memory': format_mem(c['lim_mem'])
                }
            }
        })

    # Determine Target Kind
    target_kind = 'Deployment'
    # Simple heuristic
    if 'statefulset' in name or 'redis' in name or 'postgres' in name or 'mosquitto' in name:
        target_kind = 'StatefulSet'
    if 'daemonset' in name or 'node-exporter' in name or 'promtail' in name:
        target_kind = 'DaemonSet'

    patch_content = {
        'apiVersion': 'apps/v1',
        'kind': target_kind,
        'metadata': {'name': name},
        'spec': {
            'template': {
                'spec': {
                    'containers': c_specs
                }
            }
        }
    }

    with open(patch_file, 'w') as pf:
        yaml.dump(patch_content, pf, default_flow_style=False)
    
    # Update Kustomization (Idempotent)
    kust_file = os.path.join(path, 'kustomization.yaml')
    if os.path.exists(kust_file):
        with open(kust_file, 'r') as kf:
            kust = yaml.safe_load(kf) or {}
        
        needs_update = False
        
        # Check strategic merge
        if 'patchesStrategicMerge' not in kust:
            kust['patchesStrategicMerge'] = []
        
        if 'resources-patch.yaml' not in kust['patchesStrategicMerge']:
            kust['patchesStrategicMerge'].append('resources-patch.yaml')
            needs_update = True
            
        if needs_update:
            with open(kust_file, 'w') as kf:
                yaml.dump(kust, kf, default_flow_style=False)
