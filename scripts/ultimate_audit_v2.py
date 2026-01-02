import json
import os
import subprocess
import re

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

def get_resources(path):
    if not path or not os.path.exists(path): return "N/A", "N/A"
    try:
        # Search for memory requests
        req = subprocess.check_output(f"grep -r 'memory:' {path} -B 2 | grep 'requests' -A 1 | grep 'memory' | head -n 1 | awk '{{print $2}}'", shell=True).decode().strip()
        # Search for memory limits
        lim = subprocess.check_output(f"grep -r 'memory:' {path} -B 2 | grep 'limits' -A 1 | grep 'memory' | head -n 1 | awk '{{print $2}}'", shell=True).decode().strip()
        return req or "N/A", lim or "N/A"
    except:
        return "N/A", "N/A"

with open('/tmp/argo_apps_full.json', 'r') as f: argo_data = json.load(f)
with open('/tmp/vpa_prod.json', 'r') as f: vpa_data = json.load(f)

vpa_map = {}
for item in vpa_data.get('items', []):
    ns = item['metadata']['namespace']
    target = item['spec']['targetRef']['name']
    recs = item.get('status', {}).get('recommendation', {}).get('containerRecommendations', [])
    for r in recs:
        vpa_map[f"{ns}/{target}/{r['containerName']}"] = parse_size(r['target'].get('memory', '0'))

print("# Audit de Production Total (74 Apps)")
print("| Application | Namespace | Git Path | Prod Req | Prod Lim | VPA Target | Status |")
print("| :--- | :--- | :--- | :--- | :--- | :--- | :--- |")

for app in argo_data['items']:
    name = app['metadata']['name']
    ns = app['spec']['destination']['namespace']
    sources = []
    if 'source' in app['spec']: sources.append(app['spec']['source'])
    if 'sources' in app['spec']: sources.extend(app['spec']['sources'])
    
    # Check all sources for a local path
    req, lim = "N/A", "N/A"
    main_path = "N/A"
    for s in sources:
        p = s.get('path')
        if p:
            main_path = p
            r, l = get_resources(p)
            if r != "N/A": req = r
            if l != "N/A": lim = l

    vpa_val = 0
    vpa_display = "No VPA"
    for k, v in vpa_map.items():
        if k.startswith(f"{ns}/{name}"):
            vpa_val = v
            vpa_display = format_size(v)
            break
    
    status = "⚪ NO LIMITS"
    if lim != "N/A":
        status = "🟢 OK"
        lim_bytes = parse_size(lim)
        if vpa_val > 0 and lim_bytes > 0:
            if vpa_val > lim_bytes: status = "🔴 OOM RISK"
            elif vpa_val > lim_bytes * 0.8: status = "🟠 WARNING"
    
    if vpa_display == "No VPA": status += " / NO VPA"

    print(f"| {name} | {ns} | {main_path} | {req} | {lim} | {vpa_display} | {status} |")
