import json
import os
import yaml
import re

# Load Data
with open('/tmp/argo_apps_full.json', 'r') as f: argo_data = json.load(f)
with open('/tmp/vpa_prod.json', 'r') as f: vpa_data = json.load(f)

# Helper: Parse size
def parse_size(size_str):
    if not size_str: return 0
    size_str = str(size_str).strip().replace('"', '').replace("'", "")
    units = {"k": 10**3, "M": 10**6, "G": 10**9, "T": 10**12, "Ki": 2**10, "Mi": 2**20, "Gi": 2**30, "Ti": 2**40}
    res = re.match(r'^(\d+)([a-zA-Z]+)?$', size_str)
    if not res: return 0
    val, unit = res.groups()
    return int(val) * units.get(unit, 1)

def parse_cpu(cpu_str):
    if not cpu_str: return 0
    cpu_str = str(cpu_str).strip().replace('"', '').replace("'", "")
    if cpu_str.endswith('m'): return int(cpu_str[:-1])
    try: return int(float(cpu_str) * 1000)
    except: return 0

# Helper: Format size
def format_mem(bytes):
    return f"{int(bytes/1024/1024)}Mi"

def format_cpu(millis):
    return f"{int(millis)}m"

# Build VPA Map
vpa_map = {}
for item in vpa_data.get('items', []):
    ns = item['metadata']['namespace']
    target = item['spec']['targetRef']['name']
    recs = item.get('status', {}).get('recommendation', {}).get('containerRecommendations', [])
    for r in recs:
        key = f"{ns}/{target}/{r['containerName']}"
        vpa_map[key] = {
            'cpu_target': parse_cpu(r['target'].get('cpu', '0')),
            'mem_target': parse_size(r['target'].get('memory', '0')),
            'cpu_upper': parse_cpu(r['upperBound'].get('cpu', '0')),
            'mem_upper': parse_size(r['upperBound'].get('memory', '0'))
        }

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
    
    if not path:
        print(f"Skipping {name}: No local path found.")
        continue

    # Identify Containers (from VPA or heuristic)
    # If VPA exists, we use it to patch specific containers
    # If not, we guess 'app' or use the app name
    
    containers_to_patch = {}
    
    # 1. Check VPA matches
    has_vpa = False
    for key, vpa in vpa_map.items():
        if key.startswith(f"{ns}/"):
            # Fuzzy match or exact match on deployment name? 
            # VPA key is ns/deployment_name/container_name
            # We assume deployment name ~ app name for simplicity in this mass script
            # But sometimes it's diff (e.g. authetik-server)
            parts = key.split('/')
            c_name = parts[2]
            
            # Smart sizing
            req_cpu = max(vpa['cpu_target'], 10)
            req_mem = max(vpa['mem_target'], 32*1024*1024)
            
            # Limits: Give room!
            lim_cpu = max(vpa['cpu_upper'], vpa['cpu_target']*4, 500) # CPU is compressible, let it burst
            lim_mem = max(vpa['mem_upper'], vpa['mem_target']*1.5, 128*1024*1024) # RAM is not, +50% safety
            
            containers_to_patch[c_name] = {
                'req_cpu': req_cpu, 'req_mem': req_mem,
                'lim_cpu': lim_cpu, 'lim_mem': lim_mem
            }
            has_vpa = True

    # 2. If NO VPA, apply defaults based on type
    if not has_vpa:
        # Heuristic for type
        c_type = 'app'
        if 'redis' in name or 'postgres' in name or 'db' in name: c_type = 'db'
        elif 'argocd' in name or 'traefik' in name or 'cert-manager' in name: c_type = 'infra'
        
        d = DEFAULTS[c_type]
        # We assume container name matches app name or is 'app' or 'server'
        # We will create a patch for the 'app name' container AND generic names
        # But patching non-existent containers is ignored by K8s usually? No, it might fail validation.
        # Safer: Patch the app name.
        containers_to_patch[name] = {
            'req_cpu': d['req_cpu'], 'req_mem': d['req_mem'],
            'lim_cpu': d['lim_cpu'], 'lim_mem': d['lim_mem']
        }

    # Generate Patch YAML
    if not containers_to_patch: continue

    patch_file = os.path.join(path, 'resources-patch.yaml')
    
    # Prepare container specs
    c_specs = []
    for c_name, res in containers_to_patch.items():
        c_specs.append({
            'name': c_name,
            'resources': {
                'requests': {
                    'cpu': format_cpu(res['req_cpu']),
                    'memory': format_mem(res['req_mem'])
                },
                'limits': {
                    'cpu': format_cpu(res['lim_cpu']),
                    'memory': format_mem(res['lim_mem'])
                }
            }
        })

    # Construct the Patch
    # We need to know the Kind/Name of the target.
    # Assumption: Deployment with name = app name. 
    # Exception: StatefulSet (redis, postgres). 
    # We will try to detect it or default to Deployment.
    
    target_kind = 'Deployment'
    target_name = name
    
    # Refine target based on existing files
    try:
        if os.path.exists(os.path.join(path, 'statefulset.yaml')): target_kind = 'StatefulSet'
        # Check existing kustomization for patches
        with open(os.path.join(path, 'kustomization.yaml'), 'r') as kf:
            k_content = kf.read()
            if 'StatefulSet' in k_content: target_kind = 'StatefulSet'
            if 'DaemonSet' in k_content: target_kind = 'DaemonSet'
    except: pass

    patch_content = {
        'apiVersion': 'apps/v1',
        'kind': target_kind,
        'metadata': {'name': target_name},
        'spec': {
            'template': {
                'spec': {
                    'containers': c_specs
                }
            }
        }
    }

    # Write Patch
    with open(patch_file, 'w') as pf:
        yaml.dump(patch_content, pf, default_flow_style=False)
    
    print(f"Generated patch for {name} ({target_kind}) at {patch_file}")

    # Update Kustomization
    kust_file = os.path.join(path, 'kustomization.yaml')
    if os.path.exists(kust_file):
        with open(kust_file, 'r') as kf:
            kust = yaml.safe_load(kf)
        
        # Check if patch is already included
        has_patch = False
        
        # Check patchesStrategicMerge
        if 'patchesStrategicMerge' in kust:
            if 'resources-patch.yaml' in kust['patchesStrategicMerge']:
                has_patch = True
        
        # Check patches
        if 'patches' in kust:
            for p in kust['patches']:
                if p.get('path') == 'resources-patch.yaml':
                    has_patch = True
        
        if not has_patch:
            # Prefer patchesStrategicMerge for simple updates
            if 'patchesStrategicMerge' not in kust:
                kust['patchesStrategicMerge'] = []
            kust['patchesStrategicMerge'].append('resources-patch.yaml')
            
            with open(kust_file, 'w') as kf:
                yaml.dump(kust, kf, default_flow_style=False)
            print(f"Updated kustomization for {name}")

