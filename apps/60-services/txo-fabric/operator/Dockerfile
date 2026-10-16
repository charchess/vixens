# Consume the reviewed first-party GHCR builder mirror by immutable digest.
# CI authenticates to GHCR before this pull; no registry credential is in Git.
# No external BuildKit frontend is required by this Dockerfile.
FROM ghcr.io/charchess/txo-fabric-go-builder@sha256:7b234d8ecfd110d787d4dd83ac42bfaa933a5fe3d6cde139cac92b3245141e31 AS builder
WORKDIR /workspace
RUN apk add --no-cache ca-certificates
COPY go.mod ./
RUN go mod download
COPY . .
RUN go mod tidy
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags='-s -w' -o /out/manager ./cmd && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags='-s -w' -o /out/txo-fabric-admin ./cmd/admin

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/manager /manager
COPY --from=builder /out/txo-fabric-admin /txo-fabric-admin
USER 65532:65532
ENTRYPOINT ["/manager"]
