import re
import subprocess
import json
import os
import unicodedata

def parse_cpu(cpu_str):
    if not cpu_str or cpu_str == "N/A": return 0
    cpu_str = str(cpu_str).strip().replace('"', '').replace("'", "")
    if cpu_str.endswith('m'): return int(cpu_str[:-1])
    try: return int(float(cpu_str) * 1000)
    except: return 0

def format_cpu(cpu_milli):
    if cpu_milli == 0: return "N/A"
    return f"{cpu_milli}m"

def parse_memory(size_str):
    if not size_str or size_str == "N/A": return 0
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
    return f"{size_bytes:.1f}Pi" # Fallback

def run_command(cmd, shell=False):
    try:
        result = subprocess.run(cmd, shell=shell, capture_output=True, text=True, check=True)
        return result.stdout
    except subprocess.CalledProcessError as e:
        print(f"Error running command {' '.join(cmd) if isinstance(cmd, list) else cmd}: {e.stderr}")
        return None

def get_kubectl_json(args):
    cmd = ["kubectl"] + args + ["-o", "json"]
    out = run_command(cmd)
    if out:
        return json.loads(out)
    return None

def parse_markdown_table(file_path, table_index=0, header_contains=None):
    """
    Parse markdown table from file.

    Args:
        file_path: Path to markdown file
        table_index: Index of table to parse (0 = first, 1 = second, etc.)
        header_contains: String that must be in header row to identify table (e.g., "App")
    """
    if not os.path.exists(file_path):
        return []

    with open(file_path, 'r', encoding='utf-8') as f:
        lines = f.readlines()

    table_started = False
    headers = []
    rows = []
    tables_found = 0

    for line in lines:
        line = line.strip()
        if line.startswith('|') and '|' in line:
            parts = [p.strip() for p in line.split('|')]
            if len(parts) > 2:
                parts = parts[1:-1] # Remove first and last empty parts

            if not table_started:
                # Check if it's the header line
                if any(c.isalnum() for c in "".join(parts)):
                    # Check if this is the table we want
                    if header_contains:
                        if not any(header_contains in p for p in parts):
                            continue  # Skip this table, not the one we want

                    if tables_found == table_index:
                        headers = parts
                        table_started = True
                    else:
                        # Skip to next table
                        headers = parts  # Temporary set to detect table end
                        table_started = False
            elif line.replace(' ', '').replace('-', '').replace(':', '').replace('|', '') == '':
                # Separator line
                continue
            else:
                if len(headers) > 0 and tables_found == table_index:
                    row = dict(zip(headers, parts))
                    rows.append(row)
        elif line == "":
            if table_started:
                # Table ended - return if it's the one we want
                return rows
            elif len(headers) > 0:
                # A table ended but it wasn't the one we want
                headers = []
                tables_found += 1

    return rows

def get_char_display_width(char):
    """Get display width of a character"""
    if not char: return 0
    # East Asian Width: W (Wide) or F (Fullwidth) = 2 spaces
    # N (Neutral), Na (Narrow), H (Halfwidth) = 1 space
    # A (Ambiguous) = context dependent, usually 1 in terminals
    eaw = unicodedata.east_asian_width(char)
    if eaw in ('W', 'F'):
        return 2
    return 1

def get_str_display_width(s):
    """Calculate display width of string considering wide characters"""
    width = 0
    for char in str(s):
        width += get_char_display_width(char)
    return width

def save_markdown_table(headers, rows):
    if not rows:
        return ""
    
    # Calculate column widths
    widths = {h: get_str_display_width(h) for h in headers}
    for row in rows:
        for h in headers:
            val = str(row.get(h, ""))
            widths[h] = max(widths[h], get_str_display_width(val))
    
    def pad(s, width):
        s_width = get_str_display_width(s)
        padding = width - s_width
        return s + " " * max(0, padding)

    header_line = "| " + " | ".join(pad(h, widths[h]) for h in headers) + " |"
    sep_line = "| " + " | ".join("-" * widths[h] for h in headers) + " |"
    
    lines = [header_line, sep_line]
    for row in rows:
        row_line = "| " + " | ".join(pad(str(row.get(h, "")), widths[h]) for h in headers) + " |"
        lines.append(row_line)
    
    return "\n".join(lines)
