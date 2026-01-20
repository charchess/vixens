import json
import os
import subprocess
import re
import glob

def parse_size(size_str):
    if not size_str: return 0
    size_str = str(size_str).strip().replace('"', '').replace("'", "")
    units = {"k": 10**3, "M": 10**6, "G": 10**9, "T": 10**12, "Ki": 2**10, "Mi": 2**20, "Gi": 2**30, "Ti": 2**40}
    res = re.match(r'^(\d+)([a-zA-Z]+)?$', size_str)
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
    cpu_str = str(cpu_str).strip().replace('"', '').replace("'", "")
    if cpu_str.endswith('m'): return int(cpu_str[:-1])
    try: return int(float(cpu_str) * 1000)
    except: return 0

def find_app_dir(app_name):
    # Search for the directory named app_name inside apps/
    for root, dirs, files in os.walk('apps'):
        if app_name in dirs:
            return os.path.join(root, app_name)
    return None

def extract_resources_from_dir(app_dir):
    # Look for files in overlays/prod or base
    res = {'req_cpu': 'N/A', 'req_mem': 'N/A', 'lim_cpu': 'N/A', 'lim_mem': 'N/A'}
    
    # Priority search in overlays/prod
    files = glob.glob(f"{app_dir}/overlays/prod/*.yaml") + glob.glob(f"{app_dir}/base/*.yaml")
    
    for f in files:
        try:
            with open(f, 'r') as file_content:
                content = file_content.read()
                if 'resources:' in content:
                    # Very basic regex to find values
                    m_req = re.search(r'requests:.*?memory:\s*"?([^\s"]+)"?', content, re.S)
                    m_lim = re.search(r'limits:.*?memory:\s*"?([^\s"]+)"?', content, re.S)
                    c_req = re.search(r'requests:.*?cpu:\s*"?([^\s"]+)"?', content, re.S)
                    c_lim = re.search(r'limits:.*?cpu:\s*"?([^\s"]+)"?', content, re.S)
                    
                    if m_req and res['req_mem'] == 'N/A': res['req_mem'] = m_req.group(1)
                    if m_lim and res['lim_mem'] == 'N/A': res['lim_mem'] = m_lim.group(1)
                    if c_req and res['req_cpu'] == 'N/A': res['req_cpu'] = c_req.group(1)
                    if c_lim and res['lim_cpu'] == 'N/A': res['lim_cpu'] = c_lim.group(1)
        except: pass
    return res

with open('/tmp/argo_apps_full.json', 'r') as f: argo_data = json.load(f)
with open('/tmp/vpa_prod.json', 'r') as f: vpa_data = json.load(f)

vpa_map = {}
for item in vpa_data.get('items', []):
    ns = item['metadata']['namespace']
    target = item['spec']['targetRef']['name']
    recs = item.get('status', {}).get('recommendation', {}).get('containerRecommendations', [])
    for r in recs:
        vpa_map[f"{ns}/{target}/{r['containerName']}"] = {
            'cpu': parse_cpu(r['target'].get('cpu', '0')),
            'mem': parse_size(r['target'].get('memory', '0'))
        }

print("# Audit de Production Total (74 Apps)")
print("| App | Namespace | Prod Req | Prod Lim | VPA Target | Status |")
print("| :--- | :--- | :--- | :--- | :--- | :--- |")

for app in argo_data['items']:
    name = app['metadata']['name']
    ns = app['spec']['destination']['namespace']
    
    app_dir = find_app_dir(name)
    res = extract_resources_from_dir(app_dir) if app_dir else {'req_cpu': 'N/A', 'req_mem': 'N/A', 'lim_cpu': 'N/A', 'lim_mem': 'N/A'}

    vpa_val = {'cpu': 0, 'mem': 0}
    vpa_display = "No VPA"
    for k, v in vpa_map.items():
        if k.startswith(f"{ns}/{name}"):
            vpa_val = v
            vpa_display = f"{v['cpu']}m / {format_size(v['mem'])}"
            break
    
    status = "⚪ NO LIMITS"
    if res['lim_mem'] != 'N/A':
        status = "🟢 OK"
        lim_bytes = parse_size(res['lim_mem'])
        lim_cpu = parse_cpu(res['lim_cpu'])
        if vpa_val['mem'] > 0 and lim_bytes > 0:
            if vpa_val['mem'] > lim_bytes: status = "🔴 OOM RISK"
            elif vpa_val['mem'] > lim_bytes * 0.8: status = "🟠 WARNING"
        if vpa_val['cpu'] > 0 and lim_cpu > 0 and vpa_val['cpu'] > lim_cpu:
            status = "🔴 CPU BRIDÉ"
    
    if vpa_display == "No VPA": status += " / NO VPA"

    print(f"| {name} | {ns} | {res['req_cpu']}/{res['req_mem']} | {res['lim_cpu']}/{res['lim_mem']} | {vpa_display} | {status} |")
