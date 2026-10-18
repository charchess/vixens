package controller

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func privateAPIFixture() (*fabricv1alpha1.AgentIdentity, *fabricv1alpha1.TenantBundle) {
	tenant := humanAccessTestTenant()
	tenant.UID = types.UID("e0e0e0e0-0000-4000-8000-000000000001")
	tenant.Spec.TenantID = "TEN00001"
	agent := humanAccessTestAgent()
	agent.UID = types.UID("f0f0f0f0-0000-4000-8000-000000000001")
	agent.Generation = 3
	return agent, tenant
}

func validFuturePrivateAPI(agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) *fabricv1alpha1.AgentPrivateAPIBinding {
	return &fabricv1alpha1.AgentPrivateAPIBinding{
		Enabled:       true,
		TenantID:      tenant.Spec.TenantID,
		CredentialRef: "hermes-api-" + agent.Spec.AgentKey,
	}
}

func TestPrivateHermesAPIIntentFailClosedAndSourceOnly(t *testing.T) {
	a, tenant := privateAPIFixture()
	if reason, _ := privateHermesAPIIntent(a, tenant); reason != "Disabled" {
		t.Fatalf("absent private API reason=%s", reason)
	}
	a.Spec.Runtime.PrivateAPI = &fabricv1alpha1.AgentPrivateAPIBinding{}
	if reason, _ := privateHermesAPIIntent(a, tenant); reason != "Disabled" {
		t.Fatalf("disabled private API reason=%s", reason)
	}
	a.Spec.Runtime.PrivateAPI = validFuturePrivateAPI(a, tenant)
	if reason, message := privateHermesAPIIntent(a, tenant); reason != "NotReady" ||
		message != "private Hermes API runtime launch contract and endpoint attestation are not implemented" {
		t.Fatalf("future-only opt-in must remain truthful and not report Ready: reason=%s message=%q", reason, message)
	}
	setPrivateHermesAPICondition(a, tenant)
	condition := apiMeta.FindStatusCondition(a.Status.Conditions, privateHermesAPICondition)
	if condition == nil || condition.Status != metav1.ConditionFalse ||
		condition.Reason != "NotReady" || condition.ObservedGeneration != a.Generation {
		t.Fatalf("incorrect disabled status: %#v", condition)
	}
	// A second reconciliation is idempotent and doesn't change the status.
	before := a.DeepCopy().Status
	setPrivateHermesAPICondition(a, tenant)
	if !reflect.DeepEqual(before, a.Status) {
		t.Fatalf("source-only condition changed on repeated reconciliation")
	}
}

func TestPrivateHermesAPIIntentRejectsForeignMissingAndIncompleteBindings(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*fabricv1alpha1.AgentIdentity, *fabricv1alpha1.TenantBundle)
	}{
		{"missing credential", func(a *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
			a.Spec.Runtime.PrivateAPI.CredentialRef = ""
		}},
		{"cross tenant", func(a *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
			a.Spec.Runtime.PrivateAPI.TenantID = "TEN99999"
		}},
		{"not canonical", func(a *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
			a.Spec.Runtime.PrivateAPI.TenantID = "group-admin"
		}},
		{"arbitrary secret name", func(a *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
			a.Spec.Runtime.PrivateAPI.CredentialRef = "other-agent-api"
		}},
		{"URL as credential", func(a *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
			a.Spec.Runtime.PrivateAPI.CredentialRef = "https://example.invalid"
		}},
		{"key-like credential", func(a *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
			a.Spec.Runtime.PrivateAPI.CredentialRef = strings.Repeat("X", 45)
		}},
		{"foreign tenant source", func(a *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
			a.Spec.TenantRef.Name = "different"
		}},
		{"tenant identity mutated", func(a *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
			tenant.Spec.TenantID = "TEN99999"
		}},
		{"missing agent UID", func(a *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
			a.UID = ""
		}},
		{"missing tenant UID", func(a *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
			tenant.UID = ""
		}},
		{"agent zero generation", func(a *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
			a.Generation = 0
		}},
		{"agent deleting", func(a *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
			now := metav1.Now()
			a.DeletionTimestamp = &now
		}},
		{"tenant deleting", func(a *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
			now := metav1.Now()
			tenant.DeletionTimestamp = &now
		}},
		{"unsafe agent key", func(a *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
			a.Spec.AgentKey = "../alice"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, tenant := privateAPIFixture()
			a.Spec.Runtime.PrivateAPI = validFuturePrivateAPI(a, tenant)
			tc.mutate(a, tenant)
			reason, message := privateHermesAPIIntent(a, tenant)
			if reason != "InvalidConfiguration" {
				t.Fatalf("reason=%s expected InvalidConfiguration", reason)
			}
			if strings.Contains(message, "https://") || strings.Contains(message, "TEN99999") {
				t.Fatalf("condition leaks caller data: %s", message)
			}
		})
	}
	a, tenant := privateAPIFixture()
	a.Spec.Runtime.PrivateAPI = &fabricv1alpha1.AgentPrivateAPIBinding{CredentialRef: "hermes-api-" + a.Spec.AgentKey}
	if reason, _ := privateHermesAPIIntent(a, tenant); reason != "InvalidConfiguration" {
		t.Fatalf("disabled spec with extraneous authority accepted: %s", reason)
	}
}

func TestPrivateHermesAPIContractRejectsUnknownBrowserAndSecretFields(t *testing.T) {
	for _, field := range []string{"host", "url", "ip", "port", "namespace", "sessionId", "oidcSub", "agentKey", "group", "apiKey", "token", "secretData", "enabledByBrowser"} {
		input := `{"enabled":true,"tenantId":"TEN00001","credentialRef":"hermes-api-tesla","` + field + `":"invalid"}`
		decoder := json.NewDecoder(strings.NewReader(input))
		decoder.DisallowUnknownFields()
		var value fabricv1alpha1.AgentPrivateAPIBinding
		if err := decoder.Decode(&value); err == nil {
			t.Fatalf("unknown field %q accepted by strict API contract", field)
		}
	}
	// This local decoder is a contract test; Kubernetes structural schemas
	// prune unknown fields unless clients request fieldValidation=Strict.
	// The API contract must never authorize any unrecognized field.
}

func TestPrivateHermesAPIGeneratorInertAndLegacySpecIdentical(t *testing.T) {
	a, tenant := privateAPIFixture()
	profile := testRuntimeProfile()
	ctx := context.Background()
	type snapshot struct {
		spec     appsv1.DeploymentSpec
		services corev1.ServiceList
		policies networkingv1.NetworkPolicyList
	}
	collect := func(api *fabricv1alpha1.AgentPrivateAPIBinding) snapshot {
		t.Helper()
		agent := a.DeepCopy()
		agent.Spec.Runtime.PrivateAPI = api
		c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(agent).Build()
		r := &AgentIdentityReconciler{Client: c, Scheme: testScheme(t)}
		// Use one scheme for both fake client and reconciler.
		r.Scheme = c.Scheme()
		namespace := tenantNamespace(tenant.Name)
		if err := r.ensureDeployment(ctx, agent, tenant, profile, namespace, "model-secret-uid", "baseline"); err != nil {
			t.Fatal(err)
		}
		if err := r.ensureEgressPolicy(ctx, agent, tenant, namespace); err != nil {
			t.Fatal(err)
		}
		var deployment appsv1.Deployment
		if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &deployment); err != nil {
			t.Fatal(err)
		}
		var services corev1.ServiceList
		if err := c.List(ctx, &services); err != nil {
			t.Fatal(err)
		}
		var policies networkingv1.NetworkPolicyList
		if err := c.List(ctx, &policies); err != nil {
			t.Fatal(err)
		}
		if reason, _ := privateHermesAPIIntent(agent, tenant); reason == "Ready" {
			t.Fatal("source-only API incorrectly marked ready")
		}
		for _, c := range deployment.Spec.Template.Spec.Containers {
			for _, v := range c.Env {
				if strings.HasPrefix(v.Name, "API_SERVER_") {
					t.Fatalf("new API env %s", v.Name)
				}
			}
			for _, p := range c.Ports {
				if p.ContainerPort == 8642 {
					t.Fatalf("new Hermes API listener port")
				}
			}
		}
		if len(services.Items) != 0 {
			t.Fatalf("unexpected private API service: %#v", services.Items)
		}
		return snapshot{spec: deployment.Spec, services: services, policies: policies}
	}
	baseline := collect(nil)
	for name, api := range map[string]*fabricv1alpha1.AgentPrivateAPIBinding{
		"explicit disabled":                     {},
		"enabled and complete but not runnable": validFuturePrivateAPI(a, tenant),
		"enabled missing credential":            {Enabled: true, TenantID: tenant.Spec.TenantID},
		"foreign tenant":                        {Enabled: true, TenantID: "TEN99999", CredentialRef: "hermes-api-" + a.Spec.AgentKey},
	} {
		t.Run(name, func(t *testing.T) {
			next := collect(api)
			if !reflect.DeepEqual(baseline.spec, next.spec) ||
				!reflect.DeepEqual(baseline.services, next.services) ||
				!reflect.DeepEqual(baseline.policies, next.policies) {
				t.Fatal("private API source intent changed legacy gateway or generated resource")
			}
		})
	}
}
