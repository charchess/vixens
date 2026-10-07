package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
)

const (
	defaultAIGatewayURL         = "http://txo-ai-gateway.txo-fabric-system.svc:4000"
	defaultAIGatewayModel       = "txo-default"
	tenantAICodingModel         = "txo-coding"
	sharedModelAccessBackendID  = "shared"
	defaultAIEmbeddingModel     = "txo-embedding"
	defaultAIEmbeddingDimension = ""
	modelAccessSecretKey        = "OPENAI_API_KEY"
	modelAccessSecretSuffix     = "-model-access"
	hindsightEmbeddingSecretKey = "HINDSIGHT_API_EMBEDDINGS_OPENAI_API_KEY"

	AnnotationModelAccessRotation         = "fabric.truxonline.io/model-access-rotation"
	AnnotationModelAccessRevision         = "fabric.truxonline.io/model-access-revision"
	AnnotationModelAccessBackend          = "fabric.truxonline.io/model-access-backend"
	AnnotationModelAccessRotationApplied  = "fabric.truxonline.io/model-access-rotation-applied"
	AnnotationModelAccessSecretUID        = "fabric.truxonline.io/model-access-secret-uid"
	AnnotationHindsightEmbeddingRotation  = "fabric.truxonline.io/hindsight-embedding-rotation"
	AnnotationHindsightEmbeddingRevision  = "fabric.truxonline.io/hindsight-embedding-revision"
	AnnotationHindsightEmbeddingSecretUID = "fabric.truxonline.io/hindsight-embedding-secret-uid"
)


type modelRuntimeBinding struct {
	Model   string
	BaseURL string
}

type modelAccessBackend struct {
	ID         string
	URL        string
	AdminToken string
	Model      string
}

func sharedModelAccessBackend() modelAccessBackend {
	return modelAccessBackend{
		ID:         sharedModelAccessBackendID,
		URL:        aiGatewayURL(),
		AdminToken: aiGatewayAdminToken(),
		Model:      defaultAIGatewayModel,
	}
}

func (backend modelAccessBackend) runtimeBinding() modelRuntimeBinding {
	return modelRuntimeBinding{
		Model:   backend.Model,
		BaseURL: strings.TrimRight(backend.URL, "/") + "/v1",
	}
}

func modelAccessBackendRevision(backendID, rotationRevision string) string {
	if backendID == sharedModelAccessBackendID {
		return rotationRevision
	}
	sum := sha256.Sum256([]byte(backendID + "|" + rotationRevision))
	return hex.EncodeToString(sum[:8])
}

func modelAccessSecretName(agentKey string) string {
	return runtimeName(agentKey) + modelAccessSecretSuffix
}

func modelAccessKeyAlias(agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) string {
	return modelAccessKeyAliasForNames(tenant.Name, agent.Spec.AgentKey)
}

func modelAccessKeyAliasForNames(tenantName, agentKey string) string {
	return fmt.Sprintf("txo-fabric:%s:%s", tenantName, agentKey)
}

func hindsightEmbeddingKeyAlias(bundle *fabricv1alpha1.TenantBundle) string {
	return fmt.Sprintf("txo-fabric:%s:hindsight-embeddings", bundle.Name)
}

func rotationRevision(annotations map[string]string, requestAnnotation string) string {
	value := strings.TrimSpace(annotations[requestAnnotation])
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func modelAccessRotationRevision(agent *fabricv1alpha1.AgentIdentity) string {
	return rotationRevision(agent.GetAnnotations(), AnnotationModelAccessRotation)
}

func hindsightEmbeddingRotationRevision(bundle *fabricv1alpha1.TenantBundle) string {
	return rotationRevision(bundle.GetAnnotations(), AnnotationHindsightEmbeddingRotation)
}
