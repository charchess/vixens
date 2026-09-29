package controller

import (
	"fmt"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
)

const (
	defaultAIGatewayURL         = "http://txo-ai-gateway.txo-fabric-system.svc:4000"
	defaultAIGatewayModel       = "txo-default"
	defaultAIEmbeddingModel     = "txo-embedding"
	defaultAIEmbeddingDimension = "384"
	modelAccessSecretKey        = "OPENAI_API_KEY"
	modelAccessSecretSuffix     = "-model-access"
	hindsightEmbeddingSecretKey = "HINDSIGHT_API_EMBEDDINGS_OPENAI_API_KEY"
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
