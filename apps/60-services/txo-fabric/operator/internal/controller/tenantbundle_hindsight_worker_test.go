package controller

import (
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDesiredHindsightEnvUsesStableTenantWorkerID(t *testing.T) {
	bundle := &fabricv1alpha1.TenantBundle{ObjectMeta: metav1.ObjectMeta{Name: "hairem-sandbox"}}
	names := hindsightNames{Deployment: "hindsight"}

	first := desiredHindsightEnv(bundle, names)
	second := desiredHindsightEnv(bundle, names)
	if len(first) != 1 {
		t.Fatalf("env count=%d, want 1", len(first))
	}
	if first[0].Name != "HINDSIGHT_API_WORKER_ID" {
		t.Fatalf("env name=%q, want HINDSIGHT_API_WORKER_ID", first[0].Name)
	}
	if first[0].Value != "hairem-sandbox-hindsight" {
		t.Fatalf("worker id=%q, want hairem-sandbox-hindsight", first[0].Value)
	}
	if len(second) != 1 || second[0].Value != first[0].Value {
		t.Fatalf("worker id changed across identical reconciles: first=%v second=%v", first, second)
	}

	other := desiredHindsightEnv(&fabricv1alpha1.TenantBundle{ObjectMeta: metav1.ObjectMeta{Name: "fabric-smoke"}}, names)
	if len(other) != 1 || other[0].Value == first[0].Value {
		t.Fatalf("worker id is not tenant-specific: first=%v other=%v", first, other)
	}
}
