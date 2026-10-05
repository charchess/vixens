package controller

import (
	"reflect"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStableResourceNamingContracts(t *testing.T) {
	if got, want := tenantNamespace("hairem"), "tenant-hairem"; got != want {
		t.Fatalf("tenant namespace=%q want %q", got, want)
	}
	if got, want := runtimeName("usr000001-agt00012"), "hermes-usr000001-agt00012"; got != want {
		t.Fatalf("runtime name=%q want %q", got, want)
	}
	if got, want := runtimePVCName("usr000001-agt00012"), "hermes-usr000001-agt00012-data"; got != want {
		t.Fatalf("runtime PVC name=%q want %q", got, want)
	}
	if got, want := humanAccessResourceName("usr000001-agt00012"), "hermes-usr000001-agt00012-dashboard"; got != want {
		t.Fatalf("human access resource name=%q want %q", got, want)
	}
	if got, want := authentikApplicationName("indiba"), "txo-fabric-indiba"; got != want {
		t.Fatalf("Authentik application=%q want %q", got, want)
	}
	if got, want := authentikGroupName("indiba", "sales"), "txo-fabric-indiba-sales"; got != want {
		t.Fatalf("Authentik group=%q want %q", got, want)
	}
}

func TestStableIdentityAndBankResolutionContracts(t *testing.T) {
	agent := &fabricv1alpha1.AgentIdentity{
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: "hairem"},
			AgentKey:  "usr000001-agt00012",
		},
	}
	if got, want := resolvedBankID(agent), "hairem-usr000001-agt00012"; got != want {
		t.Fatalf("fallback bank=%q want %q", got, want)
	}

	agent.Spec.Memory.BankID = "ten00001-usr000001-agt00012"
	if got, want := resolvedBankID(agent), "ten00001-usr000001-agt00012"; got != want {
		t.Fatalf("explicit canonical bank=%q want %q", got, want)
	}

	if got, want := normalizedProfileRef(agent), "hermes-default"; got != want {
		t.Fatalf("default runtime profile=%q want %q", got, want)
	}
	agent.Spec.Runtime.ProfileRef = "hermes-restricted"
	if got, want := normalizedProfileRef(agent), "hermes-restricted"; got != want {
		t.Fatalf("explicit runtime profile=%q want %q", got, want)
	}
}

func TestTenantLabelsRemainStableAndTenantScoped(t *testing.T) {
	tenant := &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "indiba"},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID: "TEN00002",
		},
	}
	want := map[string]string{
		LabelPartOf:     "txo-fabric",
		LabelManaged:    "true",
		LabelTenantID:   "TEN00002",
		LabelTenantName: "indiba",
	}
	if got := tenantLabels(tenant); !reflect.DeepEqual(got, want) {
		t.Fatalf("tenant labels=%#v want %#v", got, want)
	}
}

func TestAgentLabelsRemainStableAndTenantScoped(t *testing.T) {
	tenant := &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "indiba"},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID: "TEN00002",
		},
	}
	agent := &fabricv1alpha1.AgentIdentity{
		Spec: fabricv1alpha1.AgentIdentitySpec{
			AgentKey: "usr000001-agt00001",
		},
	}
	want := map[string]string{
		LabelPartOf:     "txo-fabric",
		LabelName:       "hermes-agent",
		LabelInstance:   "usr000001-agt00001",
		LabelTenantID:   "TEN00002",
		LabelTenantName: "indiba",
		LabelAgent:      "usr000001-agt00001",
	}
	if got := agentLabels(agent, tenant); !reflect.DeepEqual(got, want) {
		t.Fatalf("agent labels=%#v want %#v", got, want)
	}
}
