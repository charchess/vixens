import importlib
import inspect
import pkgutil
from pathlib import Path
from termcolor import colored

# --- Test Decorator for Tagging ---
def tag(*tags):
    def decorator(func):
        if not hasattr(func, 'tags'):
            func.tags = set()
        func.tags.update(tags)
        return func
    return decorator

# --- Test Discovery and Execution ---

def discover_tests(base_path, test_type):
    """
    Discovers all test functions in a given directory.
    A test function is any function starting with 'test_'.
    """
    tests = []
    package_path = base_path / f"{test_type}_tests"
    
    for _, name, _ in pkgutil.iter_modules([str(package_path)]):
        module = importlib.import_module(f"testing.{test_type}_tests.{name}")
        for func_name, func_obj in inspect.getmembers(module, inspect.isfunction):
            if func_name.startswith('test_'):
                tests.append(func_obj)
    
    # Sort tests based on their filename (e.g., test_01_, test_02_)
    tests.sort(key=lambda f: f.__module__)
    return tests


def run_tests(test_type, config, reporter, tags=None):
    """
    Discovers and runs tests, collecting results.

    Args:
        test_type (str): 'fonc' or 'tech'.
        config (dict): The environment configuration object.
        reporter (Reporter): The reporter instance for output.
        tags (set): A set of tags to filter which tests to run.
    """
    base_path = Path(__file__).parent
    
    # Map 'fonc' to 'functional' and 'tech' to 'technical' for directory names
    test_dir_name = "functional" if test_type == 'fonc' else "technical"
    
    all_tests = discover_tests(base_path, test_dir_name)
    results = []

    k8s_client_v1 = None
    if test_type == 'tech':
        # Lazy load kubernetes client only for technical tests
        try:
            from kubernetes import client, config as k8s_config
            k8s_config.load_kube_config(config_file=config['kubeconfig_path'])
            k8s_client_v1 = client.CoreV1Api()
        except Exception as e:
            reporter.print_error(f"Failed to initialize Kubernetes client: {e}")
            # Append a failure result and stop
            results.append({
                "test_name": "Kubernetes Client Initialization",
                "success": False,
                "message": "Could not connect to the cluster.",
                "details": str(e)
            })
            return results

    for test_func in all_tests:
        test_name = test_func.__name__.replace('_', ' ').capitalize()
        test_doc = inspect.getdoc(test_func) or "No description."

        # Tag filtering
        test_tags = getattr(test_func, 'tags', set())
        if tags and not tags.intersection(test_tags):
            reporter.print_message(f"[SKIP] {test_name} (tags not matched)", "yellow")
            continue

        reporter.print_message(f"Running: {test_name}...", "white")
        reporter.print_message(f"  -> {test_doc}")

        try:
            # Pass required fixtures/dependencies to the test function
            # This is a simple form of dependency injection.
            sig = inspect.signature(test_func)
            params_to_pass = {}
            if 'config' in sig.parameters:
                params_to_pass['config'] = config
            if 'k8s_client' in sig.parameters:
                params_to_pass['k8s_client'] = k8s_client_v1
            
            test_func(**params_to_pass)
            
            result = reporter.record_test_result(test_name, True, "Test completed successfully.")
            results.append(result)

        except AssertionError as e:
            result = reporter.record_test_result(test_name, False, "Assertion Failed.", details=str(e))
            results.append(result)
        except Exception as e:
            result = reporter.record_test_result(test_name, False, "An unexpected error occurred.", details=f"{type(e).__name__}: {e}")
            results.append(result)

    return results
