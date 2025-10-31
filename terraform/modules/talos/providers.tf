# Providers are configured at the root module level, not in this reusable module.
# The Talos provider must be configured in the environment that calls this module.
#
# Required provider configuration (to be added in root module):
#
# provider "talos" {}
#
# This module will inherit the provider configuration from the calling module.
