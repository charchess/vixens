# syntax=docker/dockerfile:1.7

ARG HERMES_IMAGE=nousresearch/hermes-agent:v2026.9.24

# Fetch the provider source from the exact commit selected by the upstream
# Hermes v2026.9.24 plugin catalog.  This stage is build-time only.
FROM ${HERMES_IMAGE} AS hindsight-source
USER root

ARG HINDSIGHT_COMMIT=176f8c2de1369f569c489b831d143b78128b5535
RUN set -eux; \
    rm -rf /tmp/hindsight /out/hindsight; \
    mkdir -p /tmp/hindsight /out/hindsight; \
    cd /tmp/hindsight; \
    git init; \
    git remote add origin https://github.com/vectorize-io/hindsight.git; \
    git fetch --depth=1 origin "${HINDSIGHT_COMMIT}"; \
    test "$(git rev-parse FETCH_HEAD)" = "${HINDSIGHT_COMMIT}"; \
    git checkout --detach FETCH_HEAD; \
    test "$(git rev-parse HEAD)" = "${HINDSIGHT_COMMIT}"; \
    test "$(awk '/^version:/ {gsub(/"/, "", $2); print $2}' hindsight-integrations/hermes/plugin.yaml)" = "1.0.1"; \
    cp -a hindsight-integrations/hermes/. /out/hindsight/; \
    printf '%s\n' "${HINDSIGHT_COMMIT}" > /out/hindsight/.txo-source-revision

# Keep the upstream runtime and entrypoint intact.  HERMES_HOME is /opt/data
# (a volume), so the provider is baked into the immutable bundled-provider
# tree under /opt/hermes instead of the mutable user plugin directory.
FROM ${HERMES_IMAGE}
USER root

COPY --from=hindsight-source /out/hindsight/ /opt/hermes/plugins/memory/hindsight/
COPY requirements.lock /tmp/txo-hindsight-requirements.lock
COPY verify-runtime.py /tmp/verify-runtime.py
COPY txo_functional_overlay.py /opt/hermes/txo_functional_overlay.py
COPY inject_functional_overlay.py /tmp/inject_functional_overlay.py

# The target deployment uses Hindsight local_external.  The provider's catalog
# manifest only requires hindsight-client for that mode; hindsight-all/embed is
# intentionally not installed.  --no-deps preserves Hermes' own exact-pinned
# dependency graph.  aiohttp-retry is added explicitly because it is a direct
# client dependency not supplied by the upstream runtime.
RUN set -eux; \
    uv pip install \
      --python /opt/hermes/.venv/bin/python \
      --no-deps \
      --requirement /tmp/txo-hindsight-requirements.lock; \
    /opt/hermes/.venv/bin/python /tmp/inject_functional_overlay.py; \
    /opt/hermes/.venv/bin/python /tmp/verify-runtime.py; \
    rm -f /tmp/txo-hindsight-requirements.lock /tmp/verify-runtime.py /tmp/inject_functional_overlay.py
