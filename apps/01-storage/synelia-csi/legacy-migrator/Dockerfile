FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/truenas-legacy-migrator ./main.go

# The same immutable executor image provides the short-lived migration bridge
# and archive-mode rsync. No package is fetched at Job runtime.
FROM alpine:3.21.3
RUN apk add --no-cache ca-certificates rsync
COPY --from=build /out/truenas-legacy-migrator /truenas-legacy-migrator
USER 65532:65532
ENTRYPOINT ["/truenas-legacy-migrator"]
