package controller

import (
	"fmt"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
)

const (
	defaultAIGatewayURL         = "http://txo-ai-gateway.txo-fabric-system.svc:4000"
	defaultAIGatewayModel       = "txo-default"
	defaultAIEmbeddingModel     = "txo-embedding"
	defaultAIEmbeddingDimension = ""
	modelAccessSecretKey        = "OPENAI_API_KEY"
	modelAccessSecretSuffix     = "-model-access"
	hindsightEmbeddingSecretKey = "HINDSIGHT_API_EMBEDDINGS_OPENAI_API_KEY"

	AnnotationModelAccessRotation           = "fabric.truxonline.io/model-access-rotation"
	AnnotationModelAccessRevision           = "fabric.truxonline.io/model-access-revision"
	AnnotationModelAccessSecretUID          = "fabric.truxonline.io/model-access-secret-uid"
	AnnotationHindsightEmbeddingRotation    = "fabric.truxonline.io/hindsight-embedding-rotation"
	AnnotationHindsightEmbeddingRevision    = "fabric.truxonline.io/hindsight-embedding-revision"
	AnnotationHindsightEmbeddingSecretUID   = "fabric.truxonline.io/hindsight-embedding-secret-uid"
)

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
