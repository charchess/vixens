import os

filepath = 'apps/20-media/sabnzbd/base/kustomization.yaml'
with open(filepath, 'r') as f:
    lines = f.readlines()

trigger_keys = [
    'patches:', 'resources:', 'transformers:', 'components:', 'images:', 'replicas:',
    'secretGenerator:', 'configMapGenerator:', 'generators:', 'validators:'
]

i = 0
while i < len(lines):
    line = lines[i]
    stripped = line.strip()
    
    if not stripped:
        print(f"Line {i}: empty")
        i += 1
        continue
        
    current_indent = len(line) - len(line.lstrip())
    is_top_level = (current_indent == 0 and not line.startswith('-') and not line.startswith('#'))
    
    print(f"Line {i}: indent={current_indent}, top={is_top_level}, content={line.rstrip()}")
    
    if is_top_level and line.rstrip() in trigger_keys:
        j = i + 1
        next_indent = -1
        while j < len(lines):
            if lines[j].strip() and not lines[j].strip().startswith('#'):
                next_indent = len(lines[j]) - len(lines[j].lstrip())
                break
            j += 1
        print(f"  Trigger found. Next indent: {next_indent}")
    
    i += 1
