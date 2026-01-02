import json
import os
import re
import yaml

def parse_size(size_str):
    if not size_str: return 0
    units = {"k": 10**3, "M": 10**6, "G": 10**9, "T": 10**12, "Ki": 2**10, "Mi": 2**20, "Gi": 2**30, "Ti": 2**40}
    res = re.match(r'^(\d+)(\w+)?$', str(size_str))
    if not res: return 0
    val, unit = res.groups()
    return int(val) * units.get(unit, 1)

def format_size(size_bytes):
    if size_bytes == 0: return "N/A"
    for unit in ['B', 'Ki', 'Mi', 'Gi', 'Ti']:
        if size_bytes < 1024: return f"{size_bytes:.1f}{unit}"
        size_bytes /= 1024

def parse_cpu(cpu_str):
    if not cpu_str: return 0
    if str(cpu_str).endswith('m'):
        return int(cpu_str[:-1])
    return int(float(cpu_str) * 1000)

def get_resources_from_file(file_path):
    try:
        with open(file_path, 'r') as f:
            docs = yaml.safe_load_all(f)
            res_list = []
            for doc in docs:
                if not doc: continue
                # Search for containers in Deployment, StatefulSet, DaemonSet
                spec = doc.get('spec', {})
                if 'template' in spec:
                    containers = spec['template'].get('spec', {}).get('containers', [])
                    for c in containers:
                        if 'resources' in c:
                            res_list.append({
                                'name': c.get('name'),
                                'req_cpu': c['resources'].get('requests', {}).get('cpu'),
                                'req_mem': c['resources'].get('requests', {}).get('memory'),
                                'lim_cpu': c['resources'].get('limits', {}).get('cpu'),
                                'lim_mem': c['resources'].get('limits', {}).get('memory')
                            })
            return res_list
    except:
        return []

# Load VPA
with open('/tmp/vpa_prod.json', 'r') as f:
    vpa_raw = json.load(f)

vpa_map = {}
for item in vpa_raw.get('items', []):
    ns = item['metadata']['namespace']
    target_name = item['spec']['targetRef']['name']
    recs = item.get('status', {}).get('recommendation', {}).get('containerRecommendations', [])
    for r in recs:
        c_name = r['containerName']
        vpa_map[f"{ns}/{target_name}/{c_name}"] = {
            'cpu': parse_cpu(r['target'].get('cpu')),
            'mem': parse_size(r['target'].get('memory'))
        }

# Find all prod resource configs
print("| Namespace | Application | Container | Prod Req (CPU/RAM) | Prod Lim (CPU/RAM) | VPA Target (CPU/RAM) | Gap RAM | Status |")
print("| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |")

for key, vpa in vpa_map.items():
    ns, app, container = key.split('/')
    
    # Try to find the prod config file
    # Priority: overlays/prod/resources-patch.yaml -> overlays/prod/kustomization.yaml -> base/
    found = False
    prod_config = {'req_cpu': 'N/A', 'req_mem': 'N/A', 'lim_cpu': 'N/A', 'lim_mem': 'N/A'}
    
    search_paths = [
        f"apps/*/{app}/overlays/prod/resources-patch.yaml",
        f"apps/*/{app}/overlays/prod/deployment-patch.yaml",
        f"apps/*/{app}/base/deployment.yaml",
        f"apps/*/{app}/base/deployment-server.yaml",
        f"apps/*/{app}/base/deployment-worker.yaml"
    ]
    
    # This is a bit rough but works for a report
    import glob
    for p in search_paths:
        matches = glob.glob(p)
        if matches:
            file_res = get_resources_from_file(matches[0])
            for fr in file_res:
                if fr['name'] == container or len(file_res) == 1:
                    prod_config = fr
                    found = True
                    break
        if found: break

    # Calculate status
    status = "🟢 OK"
    gap_ram = ""
    if prod_config['lim_mem'] != 'N/A':
        lim_val = parse_size(prod_config['lim_mem'])
        if lim_val > 0:
            ratio = vpa['mem'] / lim_val
            if ratio > 0.9: status = "🟠 RISQUE"
            if ratio > 1.1: status = "🔴 CRITIQUE"
            gap_ram = f"{((vpa['mem'] - lim_val)/1024/1024):.0f} Mi"

    if prod_config['lim_cpu'] != 'N/A':
        lim_cpu = parse_cpu(prod_config['lim_cpu'])
        if lim_cpu > 0 and vpa['cpu'] > lim_cpu: status = "🔴 CPU BRIDÉ"

    print(f"| {ns} | {app} | {container} | {prod_config['req_cpu']}/{prod_config['req_mem']} | {prod_config['lim_cpu']}/{prod_config['lim_mem']} | {vpa['cpu']}m/{format_size(vpa['mem'])} | {gap_ram} | {status} |")
