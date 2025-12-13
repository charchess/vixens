import json
from pathlib import Path
from . import utils

# Project root, assuming this script is in scripts/testing/
PROJECT_ROOT = Path(__file__).parent.parent.parent

def get_service_urls(environment):
    """
    Generates the URLs for the main services based on environment.
    """
    # In a real scenario, this could come from a YAML file.
    services = {
        "argocd": "argocd",
        "traefik": "traefik",
        "whoami": "whoami",
        "homeassistant": "homeassistant",
        "adguard-home": "adguard"
    }
    base_domain = "truxonline.com"
    
    urls = {}
    for name, subdomain in services.items():
        urls[name] = f"https://{subdomain}.{environment}.{base_domain}"
    
    return urls

def load_config(environment):
    """
    Loads and builds the configuration object for a given environment.
    This is the core of the dynamic configuration.
    """
    tf_dir = PROJECT_ROOT / "terraform" / "environments" / environment
    if not tf_dir.exists():
        raise FileNotFoundError(f"Terraform directory for environment '{environment}' not found at {tf_dir}")

    config = {
        "name": environment,
        "project_root": PROJECT_ROOT,
        "terraform_path": str(tf_dir),
        "kubeconfig_path": str(tf_dir / f"kubeconfig-{environment}"),
        "verify_ssl": environment in ['staging', 'prod'],
        "node_ips": [],
        "cilium_lb_cidr": None,
        "urls": get_service_urls(environment)
    }

    # --- Use `terraform output` to get dynamic values ---
    try:
        # Note: This assumes `terraform init` has been run for the environment.
        # The tool could be enhanced to run it automatically if needed.
        result = utils.run_command("terraform output -json", cwd=str(tf_dir))
        tf_outputs = json.loads(result.stdout)

        # Extract values. The keys must match the 'output' names in your Terraform code.
        # Using .get() to avoid errors if an output is missing.
        node_ips = tf_outputs.get("k8s_node_ips", {}).get("value", [])
        if node_ips:
            config["node_ips"] = node_ips
        
        lb_cidr = tf_outputs.get("cilium_l2_announcement_ips", {}).get("value", None)
        if lb_cidr:
            config["cilium_lb_cidr"] = lb_cidr[0] # Assuming it's a list with one CIDR

    except Exception as e:
        # If terraform output fails, we can still proceed with a warning,
        # but tests depending on these values will fail.
        print(f"Warning: Could not fetch Terraform outputs for '{environment}'. "
              f"Tests depending on node IPs or LB CIDR will likely fail. Error: {e}")

    # Validate that the kubeconfig exists
    if not Path(config["kubeconfig_path"]).exists():
        raise FileNotFoundError(
            f"Kubeconfig file not found for environment '{environment}' at {config['kubeconfig_path']}. "
            "Ensure Terraform has been applied successfully."
        )

    return config
