CONTROLLER_GEN ?= go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.19.0

.PHONY: fmt vet test generate manifests build
fmt:
	gofmt -w $$(find . -name '*.go' -type f)

vet:
	go vet ./...

test:
	go test ./...

generate:
	$(CONTROLLER_GEN) object paths=./...

manifests:
	$(CONTROLLER_GEN) crd paths=./api/... output:crd:artifacts:config=config/crd/bases
	$(CONTROLLER_GEN) rbac:roleName=txo-fabric-operator paths=./internal/controller/... output:rbac:artifacts:config=config/rbac

build:
	CGO_ENABLED=0 go build -trimpath -o bin/manager ./cmd
