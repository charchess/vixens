import os

def fix_file(filepath):
    with open(filepath, 'r') as f:
        lines = f.readlines()
    
    new_lines = []
    shift_amount = 0
    trigger_keys = [
        'patches:', 'resources:', 'transformers:', 'components:', 'images:', 'replicas:',
        'secretGenerator:', 'configMapGenerator:', 'generators:', 'validators:', 'labels:'
    ]
    
    i = 0
    while i < len(lines):
        line = lines[i]
        stripped = line.strip()
        
        # Pass through empty lines
        if not stripped:
            new_lines.append(line)
            i += 1
            continue
            
        current_indent = len(line) - len(line.lstrip())
        
        # Check for document separator
        if line.startswith('---'):
            shift_amount = 0
            new_lines.append(line)
            i += 1
            continue

        # Check if it's a top-level key
        # A top level key must be at indent 0, not start with '-', and contain ':'
        # (Though some might not have :, like list items, but we filtered -)
        is_top_level = (current_indent == 0 and not line.startswith('-') and not line.startswith('#'))
        
        if is_top_level:
            # It is a root key.
            new_lines.append(line)
            
            # Check if it's a trigger for a list that needs indentation
            if line.rstrip() in trigger_keys:
                # Look ahead to decide if we need to indent the children
                j = i + 1
                next_indent = -1
                found_child = False
                while j < len(lines):
                    if lines[j].strip() and not lines[j].strip().startswith('#'):
                        next_indent = len(lines[j]) - len(lines[j].lstrip())
                        found_child = True
                        break
                    j += 1
                
                if found_child:
                    # If the child is at 0 indent, we MUST indent it (shift=2)
                    if next_indent == 0:
                        shift_amount = 2
                    # If the child is at 2 indent (or more), we assume it's correct
                    else:
                        shift_amount = 0
                else:
                    # Empty list or end of file
                    shift_amount = 0
            else:
                # Top level but not a trigger key (e.g. apiVersion, kind, metadata)
                shift_amount = 0
        else:
            # Not a top level key. It's a child or comment or continuation.
            # Apply current shift.
            if shift_amount > 0:
                new_lines.append(' ' * shift_amount + line)
            else:
                new_lines.append(line)
        
        i += 1

    with open(filepath, 'w') as f:
        f.writelines(new_lines)

count = 0
for root, dirs, files in os.walk('apps'):
    for file in files:
        if file == 'kustomization.yaml':
            fix_file(os.path.join(root, file))
            count += 1

print(f"Processed {count} kustomization.yaml files.")
