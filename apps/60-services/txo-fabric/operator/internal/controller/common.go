package controller

import (
	"fmt"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TenantFinalizer = "fabric.truxonline.io/tenant-cleanup"
	AgentFinalizer  = "fabric.truxonline.io/agent-cleanup"

	LabelPartOf            = "app.kubernetes.io/part-of"
	LabelName              = "app.kubernetes.io/name"
	LabelInstance          = "app.kubernetes.io/instance"
	LabelManaged           = "fabric.truxonline.io/managed"
	LabelTenantID          = "fabric.truxonline.io/tenant-id"
	LabelTenantName        = "fabric.truxonline.io/tenant-name"
	LabelAgent             = "fabric.truxonline.io/agent"
	AnnotationRuntimeState = "fabric.truxonline.io/runtime-state"
)

func tenantNamespace(name string) string {
	return "tenant-" + name
}

func runtimeName(agentKey string) string {
	return "hermes-" + agentKey
}

func runtimePVCName(agentKey string) string {
	return runtimeName(agentKey) + "-data"
}

func setCondition(conditions *[]metav1.Condition, generation int64, conditionType string, status metav1.ConditionStatus, reason, message string) {
	apiMeta.SetStatusCondition(conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: generation,
		Reason:             reason,
		Message:            message,
	})
}

func tenantLabels(bundle *fabricv1alpha1.TenantBundle) map[string]string {
	return map[string]string{
		LabelPartOf:     "txo-fabric",
		LabelManaged:    "true",
		LabelTenantID:   bundle.Spec.TenantID,
		LabelTenantName: bundle.Name,
	}
}

func agentLabels(agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) map[string]string {
	return map[string]string{
		LabelPartOf:     "txo-fabric",
		LabelName:       "hermes-agent",
		LabelInstance:   agent.Spec.AgentKey,
		LabelTenantID:   tenant.Spec.TenantID,
		LabelTenantName: tenant.Name,
		LabelAgent:      agent.Spec.AgentKey,
	}
}

func resolvedBankID(agent *fabricv1alpha1.AgentIdentity) string {
	if agent.Spec.Memory.BankID != "" {
		return agent.Spec.Memory.BankID
	}
	return fmt.Sprintf("%s-%s", agent.Spec.TenantRef.Name, agent.Spec.AgentKey)
}

func normalizedProfileRef(agent *fabricv1alpha1.AgentIdentity) string {
	if strings.TrimSpace(agent.Spec.Runtime.ProfileRef) == "" {
		return "hermes-default"
	}
	return agent.Spec.Runtime.ProfileRef
}
