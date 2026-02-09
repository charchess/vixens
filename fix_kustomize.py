import os

def fix_file(filepath):
    with open(filepath, 'r') as f:
        lines = f.readlines()
    
    new_lines = []
    indent_mode = False
    
    # Keys that start a list that needs indentation in yamllint
    # We only care about top-level keys (indent 0)
    trigger_keys = [
        'patches:', 
        'resources:', 
        'transformers:', 
        'components:', 
        'images:', 
        'replicas:',
        'secretGenerator:',
        'configMapGenerator:',
        'generators:',
        'validators:'
    ]
    
    for line in lines:
        stripped = line.strip()
        
        # If empty line, preserve it and don't change mode
        if not stripped:
            new_lines.append(line)
            continue
            
        # Detect indentation of the line
        current_indent = len(line) - len(line.lstrip())
        
        # Check if this line is a top-level key
        # A top-level key has 0 indent and ends with :
        if current_indent == 0 and not line.startswith('-') and not line.startswith('#'):
            # It is a root key.
            # Check if it triggers indentation mode
            # We match the key. 
            # Note: "resources:" matches. "resources: []" does not.
            if line.rstrip() in trigger_keys:
                indent_mode = True
                new_lines.append(line)
                continue
            else:
                # If it's a root key but not a trigger (e.g. apiVersion, kind, commonLabels),
                # we disable indent mode.
                indent_mode = False
                new_lines.append(line)
                continue
        
        # If we encounter '---', reset
        if line.startswith('---'):
            indent_mode = False
            new_lines.append(line)
            continue

        # If we are in indent mode, we indent the line by 2 spaces
        if indent_mode:
            new_lines.append('  ' + line)
        else:
            new_lines.append(line)

    with open(filepath, 'w') as f:
        f.writelines(new_lines)

# Walk through apps directory
count = 0
for root, dirs, files in os.walk('apps'):
    for file in files:
        if file == 'kustomization.yaml':
            fix_file(os.path.join(root, file))
            count += 1

print(f"Processed {count} kustomization.yaml files.")
