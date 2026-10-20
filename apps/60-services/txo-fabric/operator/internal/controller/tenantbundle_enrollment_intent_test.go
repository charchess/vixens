package controller

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
)

func TestGlobalEnrollmentIntentRequiresOneExactReviewedProof(t *testing.T) {
	intents := []fabricv1alpha1.HumanEnrollmentIntent{{
		AuthentikUserUUID: "550e8400-e29b-41d4-a716-446655440000",
		OIDCSubject:       "opaque-tenant-subject",
		GlobalOIDCIssuer:  "https://authentik.example/application/o/txo-fabric-webui/",
		GlobalOIDCSubject: "opaque-global-subject",
	}}
	valid := trustedHumanOIDCProof{
		globalIssuer:  "https://authentik.example/application/o/txo-fabric-webui/",
		globalSubject: "opaque-global-subject",
		authentikUUID: "550e8400-e29b-41d4-a716-446655440000",
	}
	intent, ok := globalEnrollmentIntent(intents, valid)
	if !ok || intent.OIDCSubject != "opaque-tenant-subject" {
		t.Fatal("exact reviewed global enrollment proof denied")
	}
	for _, proof := range []trustedHumanOIDCProof{
		{globalIssuer: valid.globalIssuer, globalSubject: "other-global-subject", authentikUUID: valid.authentikUUID},
		{globalIssuer: "https://authentik.example/application/o/other/", globalSubject: valid.globalSubject, authentikUUID: valid.authentikUUID},
		{globalIssuer: valid.globalIssuer, globalSubject: valid.globalSubject, authentikUUID: "550e8400-e29b-41d4-a716-446655440001"},
	} {
		if _, ok := globalEnrollmentIntent(intents, proof); ok {
			t.Fatal("unreviewed global identity was accepted")
		}
	}
}

func TestGlobalEnrollmentIntentRejectsMissingOrAmbiguousMapping(t *testing.T) {
	proof := trustedHumanOIDCProof{
		globalIssuer:  "https://authentik.example/application/o/txo-fabric-webui/",
		globalSubject: "opaque-global-subject", authentikUUID: "550e8400-e29b-41d4-a716-446655440000",
	}
	if _, ok := globalEnrollmentIntent([]fabricv1alpha1.HumanEnrollmentIntent{{
		AuthentikUserUUID: proof.authentikUUID, OIDCSubject: "opaque-tenant-subject",
	}}, proof); ok {
		t.Fatal("legacy intent without an explicit global tuple authorized enrollment")
	}
	intent := fabricv1alpha1.HumanEnrollmentIntent{
		AuthentikUserUUID: proof.authentikUUID, OIDCSubject: "opaque-tenant-subject",
		GlobalOIDCIssuer: proof.globalIssuer, GlobalOIDCSubject: proof.globalSubject,
	}
	if _, ok := globalEnrollmentIntent([]fabricv1alpha1.HumanEnrollmentIntent{intent, intent}, proof); ok {
		t.Fatal("ambiguous global enrollment mappings were accepted")
	}
}

func TestGeneratedEnrollmentIntentSchemaKeepsGlobalTupleOptionalForLegacyObjects(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	crd, err := os.ReadFile(filepath.Join(root, "config/crd/bases/fabric.truxonline.io_tenantbundles.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	const legacyRequired = "- authentikUserUUID\n                      - globalOidcIssuer\n                      - globalOidcSubject\n                      - oidcSubject"
	if strings.Contains(string(crd), legacyRequired) {
		t.Fatal("generated CRD rejects legacy enrollment intents before fail-closed runtime validation")
	}
}
