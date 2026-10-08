# syntax=docker/dockerfile:1.7

# Build *only* the plugin payload against the pinned official Hermes Python ABI.
# The deployed Hermes Pod uses the official image directly, never this image.
ARG HERMES_IMAGE=nousresearch/hermes-agent:v2026.9.24
FROM ${HERMES_IMAGE} AS build
USER root
ARG HINDSIGHT_COMMIT=176f8c2de1369f569c489b831d143b78128b5535
COPY requirements.lock /tmp/txo-hindsight-requirements.lock
RUN set -eux; \
    mkdir -p /bundle/hindsight /bundle/python /tmp/hindsight; \
    cd /tmp/hindsight; \
    git init; \
    git remote add origin https://github.com/vectorize-io/hindsight.git; \
    git fetch --depth=1 origin "${HINDSIGHT_COMMIT}"; \
    test "$(git rev-parse FETCH_HEAD)" = "${HINDSIGHT_COMMIT}"; \
    git checkout --detach FETCH_HEAD; \
    test "$(git rev-parse HEAD)" = "${HINDSIGHT_COMMIT}"; \
    test "$(awk '/^version:/ {gsub(/"/, "", $2); print $2}' hindsight-integrations/hermes/plugin.yaml)" = "1.0.1"; \
    cp -a hindsight-integrations/hermes/. /bundle/hindsight/; \
    printf '%s\n' "${HINDSIGHT_COMMIT}" > /bundle/hindsight/.txo-source-revision; \
    uv pip install --no-deps \
      --python /opt/hermes/.venv/bin/python \
      --target /bundle/python \
      --requirement /tmp/txo-hindsight-requirements.lock; \
    PYTHONPATH=/bundle/python /opt/hermes/.venv/bin/python -c 'import hindsight_client, aiohttp_retry'; \
    test -f /bundle/hindsight/__init__.py; \
    test -f /bundle/hindsight/plugin.yaml

# A tiny independent distribution artifact, not a Hermes derivative. It is
# executed only by the Kubernetes initContainer to copy its immutable payload.
FROM alpine:3.21.3
COPY --from=build /bundle /bundle
RUN test -f /bundle/hindsight/plugin.yaml && \
    test -f /bundle/hindsight/__init__.py && \
    test -d /bundle/python/hindsight_client && \
    test -d /bundle/python/aiohttp_retry
ENTRYPOINT ["/bin/sh"]
