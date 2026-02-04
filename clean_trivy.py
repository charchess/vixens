import sys
import yaml

def clean_yaml(input_file, output_file):
    with open(input_file, 'r') as f:
        docs = list(yaml.safe_load_all(f))
    
    clean_docs = []
    for doc in docs:
        if not doc:
            continue
        kind = doc.get('kind')
        group = doc.get('apiVersion', '').split('/')[0]
        
        # Exclude ServiceMonitor and ClusterComplianceReport
        if kind == 'ServiceMonitor' or kind == 'ClusterComplianceReport':
            continue
        
        clean_docs.append(doc)
        
    with open(output_file, 'w') as f:
        yaml.safe_dump_all(clean_docs, f)

if __name__ == '__main__':
    clean_yaml(sys.argv[1], sys.argv[2])
