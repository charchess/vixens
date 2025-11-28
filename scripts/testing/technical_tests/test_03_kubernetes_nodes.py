from testing import runners
from kubernetes import client

@runners.tag('kubernetes', 'nodes')
def test_all_nodes_are_ready(k8s_client: client.CoreV1Api):
    """
    Checks if all nodes in the cluster are in the 'Ready' state.
    """
    assert k8s_client, "Kubernetes client is not available."
    
    nodes = k8s_client.list_node()
    assert nodes.items, "No nodes found in the cluster."

    for node in nodes.items:
        # The 'Ready' condition is the last one in the list of conditions
        ready_condition = next((c for c in node.status.conditions if c.type == 'Ready'), None)
        
        assert ready_condition is not None, f"Node '{node.metadata.name}' does not have a 'Ready' condition."
        assert ready_condition.status == "True", f"Node '{node.metadata.name}' is not ready. Status is '{ready_condition.status}'."

@runners.tag('kubernetes', 'nodes')
def test_node_ips_match_terraform_output(k8s_client: client.CoreV1Api, config: dict):
    """
    Compares the internal IPs of Kubernetes nodes with the IPs from Terraform's output.
    """
    tf_node_ips = set(config.get('node_ips', []))
    assert tf_node_ips, "Node IPs not found in Terraform configuration."

    nodes = k8s_client.list_node()
    k8s_node_ips = set()
    for node in nodes.items:
        for address in node.status.addresses:
            if address.type == 'InternalIP':
                k8s_node_ips.add(address.address)

    assert tf_node_ips == k8s_node_ips, \
        f"IP mismatch between Terraform and Kubernetes.\n" \
        f"Terraform IPs: {sorted(list(tf_node_ips))}\n" \
        f"Kubernetes IPs: {sorted(list(k8s_node_ips))}"
