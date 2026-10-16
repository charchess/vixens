package controller

import (
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
)

func TestEnrollmentIntentAllowsOnlyExactImmutableIdentity(t *testing.T) {
	intents := []fabricv1alpha1.HumanEnrollmentIntent{{
		AuthentikUserUUID: "550e8400-e29b-41d4-a716-446655440000",
		OIDCSubject:       "opaque-authentik-subject",
	}}
	valid := trustedHumanOIDCProof{
		issuer:        "https://authentik.example/application/o/hairem/",
		authentikUUID: "550e8400-e29b-41d4-a716-446655440000",
		subject:       "opaque-authentik-subject",
	}
	if !enrollmentIntentAllows(intents, valid) {
		t.Fatal("exact reviewed enrollment intent denied")
	}
	for _, proof := range []trustedHumanOIDCProof{
		{issuer: valid.issuer, authentikUUID: valid.authentikUUID, subject: "other-subject"},
		{issuer: valid.issuer, authentikUUID: "550e8400-e29b-41d4-a716-446655440001", subject: valid.subject},
	} {
		if enrollmentIntentAllows(intents, proof) {
			t.Fatal("unreviewed identity was accepted")
		}
	}
}
