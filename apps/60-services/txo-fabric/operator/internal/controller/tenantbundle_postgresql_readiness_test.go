package controller

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCNPGAppliedForCurrentGenerationRejectsStaleStatus(t *testing.T) {
	obj := cnpgObject(cnpgDatabaseGVK, "databases", "tenant-db")
	obj.SetGeneration(7)
	obj.Object["status"] = map[string]interface{}{
		"applied":            true,
		"observedGeneration": int64(6),
	}

	if cnpgAppliedForCurrentGeneration(obj) {
		t.Fatal("stale CNPG status must not be treated as ready")
	}

	if err := unstructured.SetNestedField(obj.Object, int64(7), "status", "observedGeneration"); err != nil {
		t.Fatal(err)
	}
	if !cnpgAppliedForCurrentGeneration(obj) {
		t.Fatal("current applied CNPG status should be ready")
	}
}

func TestRequiredExtensionsReadyRejectsReportedFailure(t *testing.T) {
	database := cnpgObject(cnpgDatabaseGVK, "databases", "tenant-db")
	if err := unstructured.SetNestedSlice(database.Object, []interface{}{
		map[string]interface{}{
			"name":    "vector",
			"applied": false,
			"message": "extension install failed",
		},
	}, "status", "extensions"); err != nil {
		t.Fatal(err)
	}

	ready, message := requiredExtensionsReady(database, []string{"vector"})
	if ready {
		t.Fatal("failed required extension must block readiness")
	}
	if !strings.Contains(message, "vector") || !strings.Contains(message, "extension install failed") {
		t.Fatalf("unexpected readiness message: %q", message)
	}

	if err := unstructured.SetNestedSlice(database.Object, []interface{}{
		map[string]interface{}{
			"name":    "vector",
			"applied": true,
		},
	}, "status", "extensions"); err != nil {
		t.Fatal(err)
	}
	ready, message = requiredExtensionsReady(database, []string{"vector"})
	if !ready || message != "" {
		t.Fatalf("ready extension reported as blocked: ready=%v message=%q", ready, message)
	}
}
