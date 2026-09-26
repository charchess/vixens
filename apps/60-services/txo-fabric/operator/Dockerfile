# syntax=docker/dockerfile:1.7
FROM golang:1.26.0-alpine AS builder
WORKDIR /workspace
RUN apk add --no-cache ca-certificates
COPY go.mod ./
RUN go mod download
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags='-s -w' -o /out/manager ./cmd

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/manager /manager
USER 65532:65532
ENTRYPOINT ["/manager"]
