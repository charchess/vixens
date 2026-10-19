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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		message != "private Hermes API network substrate is reconciled; runtime launch and endpoint attestation are not implemented" {
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

func TestPrivateHermesAPIDisabledOptInDoesNotDeleteForeignResources(t *testing.T) {
	agent, tenant := privateAPIFixture()
	namespace := tenantNamespace(tenant.Name)
	name := runtimeName(agent.Spec.AgentKey)
	foreignOwner := metav1.OwnerReference{APIVersion: "v1", Kind: "ConfigMap", Name: "foreign", UID: types.UID("foreign-owner"), Controller: boolPtr(true)}
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, OwnerReferences: []metav1.OwnerReference{foreignOwner}}}
	policy := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: name + "-responses-ingress", Namespace: namespace, OwnerReferences: []metav1.OwnerReference{foreignOwner}}}
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(agent, service, policy).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: c.Scheme()}

	if err := r.ensurePrivateHermesAPIResources(context.Background(), agent, tenant, namespace); err == nil {
		t.Fatal("disabled private API must fail closed on a foreign canonical Service")
	}
	if err := c.Get(context.Background(), client.ObjectKeyFromObject(service), &corev1.Service{}); err != nil {
		t.Fatalf("foreign Service was deleted: %v", err)
	}
	if err := c.Get(context.Background(), client.ObjectKeyFromObject(policy), &networkingv1.NetworkPolicy{}); err != nil {
		t.Fatalf("foreign NetworkPolicy was deleted: %v", err)
	}
}

func TestPrivateHermesAPIOptInReconcilesCanonicalPrivateSubstrate(t *testing.T) {
	agent, tenant := privateAPIFixture()
	agent.Spec.Runtime.PrivateAPI = validFuturePrivateAPI(agent, tenant)
	ctx := context.Background()
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: c.Scheme()}
	namespace := tenantNamespace(tenant.Name)

	if err := r.ensureDeployment(ctx, agent, tenant, testRuntimeProfile(), namespace, "model-secret-uid", "baseline"); err != nil {
		t.Fatal(err)
	}
	if err := r.ensurePrivateHermesAPIResources(ctx, agent, tenant, namespace); err != nil {
		t.Fatal(err)
	}
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &deployment); err != nil {
		t.Fatal(err)
	}
	for _, container := range deployment.Spec.Template.Spec.Containers {
		for _, port := range container.Ports {
			if port.ContainerPort == privateHermesAPIPort {
				t.Fatal("network substrate must not enable a Responses listener")
			}
		}
		for _, env := range container.Env {
			if strings.HasPrefix(env.Name, "API_SERVER_") {
				t.Fatalf("network substrate must not add API feature env %q", env.Name)
			}
		}
	}

	var service corev1.Service
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &service); err != nil {
		t.Fatal(err)
	}
	if service.Spec.Type != corev1.ServiceTypeClusterIP || len(service.Spec.Ports) != 1 ||
		service.Spec.Ports[0].Name != "responses" || service.Spec.Ports[0].Port != 8642 ||
		service.Spec.Ports[0].TargetPort.IntVal != 8642 {
		t.Fatalf("private Hermes Service is not canonical: %#v", service.Spec)
	}
	if !reflect.DeepEqual(service.Spec.Selector, map[string]string{LabelName: "hermes-agent", LabelInstance: agent.Spec.AgentKey}) {
		t.Fatalf("service selector=%#v", service.Spec.Selector)
	}

	var ingress networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey) + "-responses-ingress", Namespace: namespace}, &ingress); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ingress.Spec.PodSelector.MatchLabels, service.Spec.Selector) ||
		!reflect.DeepEqual(ingress.Spec.PolicyTypes, []networkingv1.PolicyType{networkingv1.PolicyTypeIngress}) || len(ingress.Spec.Ingress) != 1 {
		t.Fatalf("private Hermes ingress policy=%#v", ingress.Spec)
	}
	rule := ingress.Spec.Ingress[0]
	if len(rule.Ports) != 1 || rule.Ports[0].Protocol == nil || *rule.Ports[0].Protocol != corev1.ProtocolTCP || rule.Ports[0].Port.IntVal != 8642 || len(rule.From) != 1 {
		t.Fatalf("private Hermes ingress rule=%#v", rule)
	}
	peer := rule.From[0]
	if peer.NamespaceSelector == nil || peer.NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] != "txo-fabric-system" ||
		peer.PodSelector == nil || !reflect.DeepEqual(peer.PodSelector.MatchLabels, map[string]string{
		LabelName: "txo-fabric-webui-bff", "app.kubernetes.io/component": "bff",
	}) {
		t.Fatalf("private Hermes ingress peer=%#v", peer)
	}

	agent.Spec.Runtime.PrivateAPI = nil
	if err := r.ensurePrivateHermesAPIResources(ctx, agent, tenant, namespace); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: namespace}, &corev1.Service{}); !apierrors.IsNotFound(err) {
		t.Fatalf("disabled opt-in left Service behind: %v", err)
	}
	if err := c.Get(ctx, types.NamespacedName{Name: ingress.Name, Namespace: namespace}, &networkingv1.NetworkPolicy{}); !apierrors.IsNotFound(err) {
		t.Fatalf("disabled opt-in left ingress policy behind: %v", err)
	}
}
