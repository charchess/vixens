import requests
import urllib3
from testing import runners

@runners.tag('network', 'http')
def test_service_urls_are_accessible(config):
    """
    Checks that all main service URLs are accessible and return a 200 OK status.
    It conditionally verifies the SSL certificate based on the environment.
    """
    urls_to_check = config.get('urls', {})
    verify_ssl = config.get('verify_ssl', True)

    assert urls_to_check, "No service URLs found in configuration."

    if not verify_ssl:
        # Disable the insecure request warning when we are intentionally bypassing SSL verification
        urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
        print("  -> SSL verification is DISABLED for this environment.")


    for service_name, url in urls_to_check.items():
        try:
            response = requests.get(url, timeout=15, verify=verify_ssl)
            assert response.status_code == 200, \
                f"Service '{service_name}' at {url} returned status code {response.status_code}."
        
        except requests.exceptions.SSLError as e:
            raise AssertionError(f"SSL certificate validation failed for '{service_name}' at {url}. Error: {e}")
        except requests.exceptions.RequestException as e:
            raise AssertionError(f"Failed to connect to '{service_name}' at {url}. Error: {e}")

# You could add more functional tests here, for example:
# def test_argocd_self_heal_simulation(config):
#     """
#     Simulates a manual change to test ArgoCD's self-healing capability.
#     NOTE: This is a destructive test and requires careful implementation.
#     """
#     # This would require user confirmation to run.
#     # For now, we leave it as a placeholder.
#     pass
