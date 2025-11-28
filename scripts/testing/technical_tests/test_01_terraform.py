from testing import runners
from testing import utils

@runners.tag('terraform', 'iac')
def test_terraform_plan_is_clean(config):
    """
    Runs `terraform plan` to ensure the live infrastructure matches the configuration.
    """
    tf_path = config.get('terraform_path')
    assert tf_path, "Terraform path not found in configuration."

    try:
        result = utils.run_command("terraform plan -detailed-exitcode", cwd=tf_path)
        # The -detailed-exitcode flag returns:
        # 0 = Succeeded with empty diff
        # 1 = Error
        # 2 = Succeeded with non-empty diff
        # We expect 0. The check=True in run_command will catch 1, but we must check for 2.
        assert result.returncode == 0, \
            f"Terraform plan has detected changes. Run 'terraform plan' in {tf_path} to see the diff."
    except Exception as e:
        # Provide more context on failure
        details = getattr(e, 'stderr', str(e))
        raise AssertionError(f"Terraform plan command failed. Error: {details}")

