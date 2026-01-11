import re
import subprocess
import json
import os

def parse_cpu(cpu_str):
    if not cpu_str or cpu_str == "N/A" or cpu_str == "None": return 0
    cpu_str = str(cpu_str).strip().replace('"', '').replace("'", "")
    if cpu_str.endswith('m'): return int(cpu_str[:-1])
    try: 
        if '.' in cpu_str:
            return int(float(cpu_str) * 1000)
        return int(cpu_str) * 1000
    except: return 0

def format_cpu(cpu_milli):
    if cpu_milli == 0: return "N/A"
    return f"{cpu_milli}m"

def parse_memory(size_str):
    if not size_str or size_str == "N/A" or size_str == "None": return 0
    size_str = str(size_str).strip().replace('"', '').replace("'", "")
    units = {
        "k": 10**3, "M": 10**6, "G": 10**9, "T": 10**12,
        "Ki": 2**10, "Mi": 2**20, "Gi": 2**30, "Ti": 2**40
    }
    res = re.match(r'^(\d+)([a-zA-Z]+)?$', size_str)
    if not res: return 0
    val, unit = res.groups()
    return int(val) * units.get(unit, 1)

def format_memory(size_bytes):
    if size_bytes == 0: return "N/A"
    for unit in ['B', 'Ki', 'Mi', 'Gi', 'Ti']:
        if size_bytes < 1024: return f"{size_bytes:.1f}{unit}"
        size_bytes /= 1024
    return f"{size_bytes:.1f}Pi"

def run_command(cmd, shell=False):
    try:
        result = subprocess.run(cmd, shell=shell, capture_output=True, text=True, check=True)
        return result.stdout
    except subprocess.CalledProcessError as e:
        # Silently fail for some cases or log to stderr
        # print(f"Error running command: {e.stderr}", file=sys.stderr)
        return None

def get_kubectl_json(args):
    cmd = ["kubectl"] + args + ["-o", "json"]
    extra = os.environ.get('KUBECONFIG_EXTRA_OPTS')
    if extra:
        cmd += extra.split()
    
    out = run_command(cmd)
    if out:
        try:
            return json.loads(out)
        except json.JSONDecodeError:
            return None
    return None

def parse_markdown_table(file_path):
    if not os.path.exists(file_path):
        return []
    
    with open(file_path, 'r') as f:
        lines = f.readlines()
    
    table_started = False
    headers = []
    rows = []
    
    for line in lines:
        line = line.strip()
        if line.startswith('|') and '|' in line:
            parts = [p.strip() for p in line.split('|')]
            if len(parts) > 2:
                parts = parts[1:-1] # Remove first and last empty parts
            
            if not table_started:
                # Check if it's the header line
                if any(c.isalnum() for c in "".join(parts)):
                    headers = parts
                    table_started = True
            elif line.replace(' ', '').replace('-', '').replace(':', '').replace('|', '') == '':
                # Separator line
                continue
            else:
                row = dict(zip(headers, parts))
                rows.append(row)
        elif table_started and line == "":
            # Table ended
            table_started = False
            
    return rows

def save_markdown_table(headers, rows):
    if not rows:
        return ""
    
    widths = {h: len(h) for h in headers}
    for row in rows:
        for h in headers:
            widths[h] = max(widths[h], len(str(row.get(h, ""))))
    
    header_line = "| " + " | ".join(h.ljust(widths[h]) for h in headers) + " |"
    sep_line = "| " + " | ".join("-" * widths[h] for h in headers) + " |"
    
    lines = [header_line, sep_line]
    for row in rows:
        row_line = "| " + " | ".join(str(row.get(h, "")).ljust(widths[h]) for h in headers) + " |"
        lines.append(row_line)
    
    return "\n".join(lines)