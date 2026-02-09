import os

def fix_trailing_newlines(filepath):
    try:
        with open(filepath, 'rb') as f:
            content = f.read()
        
        if not content:
            return

        # Remove trailing newlines/whitespace from end
        content = content.rstrip()
        # Add exactly one newline
        content = content + b'\n'
        
        with open(filepath, 'wb') as f:
            f.write(content)
            
    except Exception as e:
        print(f"Error fixing {filepath}: {e}")

# Walk through apps and argocd
for base_dir in ['apps', 'argocd']:
    for root, dirs, files in os.walk(base_dir):
        for file in files:
            if file.endswith('.yaml') or file.endswith('.yml'):
                fix_trailing_newlines(os.path.join(root, file))
