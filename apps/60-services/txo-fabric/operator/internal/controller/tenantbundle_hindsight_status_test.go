package controller

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHindsightEmbeddingStatusReportsAppliedRuntimeBackend(t *testing.T) {
	const tenantURL = "http://txo-ai-gateway.tenant-hairem.svc:4000/v1"
	const legacyURL = "http://txo-ai-gateway.txo-fabric-system.svc:4000/v1"
	for _, tc := range []struct {
		name, endpoint, model, revision, wantRevision string
	}{
		{name: "migrated tenant", endpoint: tenantURL, model: "txo-embedding", revision: "revision-tenant", wantRevision: "revision-tenant"},
		{name: "legacy shared", endpoint: legacyURL, model: "txo-embedding", wantRevision: "baseline"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runtime := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
					AnnotationHindsightEmbeddingRevision: tc.revision,
				}},
				Data: map[string][]byte{
					"HINDSIGHT_API_EMBEDDINGS_OPENAI_BASE_URL": []byte(tc.endpoint),
					"HINDSIGHT_API_EMBEDDINGS_OPENAI_MODEL":    []byte(tc.model),
					hindsightEmbeddingSecretKey:                 []byte("must-never-appear-in-status"),
				},
			}
			got := hindsightEmbeddingStatusMessage(runtime)
			want := "embeddings use scoped TXO AI gateway access (gateway=" + tc.endpoint +
				" model=" + tc.model + " rotation=" + tc.wantRevision + ")"
			if got != want {
				t.Fatalf("status=%q, want %q", got, want)
			}
			if strings.Contains(got, "must-never-appear-in-status") {
				t.Fatal("embedding credential leaked into status")
			}
			if tc.name == "migrated tenant" && strings.Contains(got, legacyURL) {
				t.Fatalf("tenant status unexpectedly advertises legacy shared gateway: %q", got)
			}
		})
	}
}
