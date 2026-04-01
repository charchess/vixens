# Custom NetBox image with community plugins pre-installed.
# Built automatically by GitHub Actions on any change to this directory.
# Base image version is tracked by Renovate (dockerfile manager).
#
# Tag strategy: ghcr.io/charchess/netbox-plugins:v{NETBOX_VERSION}
# Plugin updates within the same NetBox version require a manual rollout restart.

FROM netboxcommunity/netbox:v4.5.6

# Install plugins into the active venv (VIRTUAL_ENV is set in the base image)
COPY requirements.txt /tmp/plugin-requirements.txt
RUN uv pip install --no-cache -r /tmp/plugin-requirements.txt
