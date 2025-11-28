import subprocess
import shutil

def run_command(command, cwd='.', capture=True):
    """
    Executes a shell command and returns the result.

    Args:
        command (str or list): The command to execute.
        cwd (str): The working directory to run the command in.
        capture (bool): Whether to capture stdout/stderr.

    Returns:
        subprocess.CompletedProcess: The result of the command execution.
    """
    return subprocess.run(
        command,
        shell=isinstance(command, str),
        cwd=cwd,
        capture_output=capture,
        text=True,
        check=True  # Will raise CalledProcessError if the command returns a non-zero exit code
    )

def check_dependencies(dependencies):
    """
    Checks if all required command-line tools are available in the system's PATH.

    Args:
        dependencies (list): A list of command names to check (e.g., ['terraform', 'kubectl']).

    Returns:
        bool: True if all dependencies are found, False otherwise.
    """
    all_found = True
    print("Checking for required tools...")
    for dep in dependencies:
        if shutil.which(dep):
            print(f"  [✓] {dep} found.")
        else:
            print(f"  [✗] {dep} not found. Please install it and ensure it's in your PATH.")
            all_found = False
    return all_found
