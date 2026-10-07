package controller

import "testing"

func TestDesiredHindsightSecretDataForcesBakedModelsOffline(t *testing.T) {
	data := desiredHindsightSecretData("postgresql://example", "api-key", 8888, "", "", false)

	if got := string(data["HF_HUB_OFFLINE"]); got != "1" {
		t.Fatalf("HF_HUB_OFFLINE=%q, want 1", got)
	}
	if got := string(data["TRANSFORMERS_OFFLINE"]); got != "1" {
		t.Fatalf("TRANSFORMERS_OFFLINE=%q, want 1", got)
	}
}
