package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultAIGatewayURL         = "http://txo-ai-gateway.txo-fabric-system.svc:4000"
	defaultAIGatewayModel       = "txo-default"
	tenantAIAgentModel         = "txo-agent"
	sharedModelAccessBackendID  = "shared"
	defaultAIEmbeddingModel     = "txo-embedding"
	defaultAIEmbeddingDimension = ""
	modelAccessSecretKey        = "OPENAI_API_KEY"
	modelAccessSecretSuffix     = "-model-access"
	hindsightEmbeddingSecretKey = "HINDSIGHT_API_EMBEDDINGS_OPENAI_API_KEY"

	AnnotationModelAccessRotation         = "fabric.truxonline.io/model-access-rotation"
	AnnotationModelAccessRevision         = "fabric.truxonline.io/model-access-revision"
	AnnotationModelAccessBackend              = "fabric.truxonline.io/model-access-backend"
	AnnotationModelAccessBackendURL           = "fabric.truxonline.io/model-access-backend-url"
	AnnotationModelAccessPendingRevokeBackend = "fabric.truxonline.io/model-access-pending-revoke-backend"
	AnnotationModelAccessPendingRevokeURL     = "fabric.truxonline.io/model-access-pending-revoke-url"
	AnnotationModelAccessRotationApplied      = "fabric.truxonline.io/model-access-rotation-applied"
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

func tenantModelAccessBackend(
	ctx context.Context,
	reader client.Reader,
	tenant *fabricv1alpha1.TenantBundle,
	model string,
) (modelAccessBackend, error) {
	if !tenantRunsAIPlane(tenant) {
		return modelAccessBackend{}, fmt.Errorf("tenant %q is parked; model access is unavailable", tenant.Name)
	}

	profileName := tenantAIGatewayProfileName(tenant)
	var profile fabricv1alpha1.AIGatewayProfile
	if err := reader.Get(ctx, types.NamespacedName{Name: profileName}, &profile); err != nil {
		return modelAccessBackend{}, fmt.Errorf("resolve tenant AI gateway profile %q: %w", profileName, err)
	}

	namespace := tenantNamespace(tenant.Name)
	var runtimeSecret corev1.Secret
	if err := reader.Get(ctx, types.NamespacedName{Name: tenantAIGatewayRuntimeSecretName, Namespace: namespace}, &runtimeSecret); err != nil {
		return modelAccessBackend{}, fmt.Errorf("read tenant AI gateway runtime Secret %s/%s: %w", namespace, tenantAIGatewayRuntimeSecretName, err)
	}
	adminToken := strings.TrimSpace(string(runtimeSecret.Data["LITELLM_MASTER_KEY"]))
	if adminToken == "" {
		return modelAccessBackend{}, fmt.Errorf("tenant AI gateway runtime Secret %s/%s is missing LITELLM_MASTER_KEY", namespace, tenantAIGatewayRuntimeSecretName)
	}

	port := aiGatewayPort(&profile)
	return modelAccessBackend{
		ID:         fmt.Sprintf("tenant:%s:%s:%d", tenant.Name, profile.Name, port),
		URL:        fmt.Sprintf("http://%s.%s.svc:%d", tenantAIGatewayName, namespace, port),
		AdminToken: adminToken,
		Model:      model,
	}, nil
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
