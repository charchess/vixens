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
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type privateHermesAPIMetadataReader struct {
	client.Reader
	metadata map[types.NamespacedName]*metav1.PartialObjectMetadata
}

func (r privateHermesAPIMetadataReader) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	metadata, ok := r.metadata[key]
	if !ok {
		return apierrors.NewNotFound(corev1.Resource("secrets"), key.Name)
	}
	target, ok := obj.(*metav1.PartialObjectMetadata)
	if !ok {
		return apierrors.NewBadRequest("metadata reader requires PartialObjectMetadata")
	}
	metadata.DeepCopyInto(target)
	return nil
}

func secretMetadata(secret *corev1.Secret) *metav1.PartialObjectMetadata {
	return &metav1.PartialObjectMetadata{
		TypeMeta:   metav1.TypeMeta{APIVersion: corev1.SchemeGroupVersion.String(), Kind: "Secret"},
		ObjectMeta: *secret.ObjectMeta.DeepCopy(),
	}
}

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

	if err := r.ensurePrivateHermesAPIResources(context.Background(), agent, tenant, namespace, true); err == nil {
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
	if err := r.ensurePrivateHermesAPIResources(ctx, agent, tenant, namespace, true); err != nil {
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
	if len(rule.Ports) != 1 || rule.Ports[0].Protocol == nil || *rule.Ports[0].Protocol != corev1.ProtocolTCP || rule.Ports[0].Port.IntVal != 8642 || len(rule.From) != 2 {
		t.Fatalf("private Hermes ingress rule=%#v", rule)
	}
	peer := rule.From[0]
	if peer.NamespaceSelector == nil || peer.NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] != "txo-fabric-system" ||
		peer.PodSelector == nil || !reflect.DeepEqual(peer.PodSelector.MatchLabels, map[string]string{
		LabelName: "txo-fabric-webui-bff", "app.kubernetes.io/component": "bff",
	}) {
		t.Fatalf("private Hermes ingress peer=%#v", peer)
	}
	openWebUIPeer := rule.From[1]
	if openWebUIPeer.NamespaceSelector != nil || openWebUIPeer.PodSelector == nil || !reflect.DeepEqual(openWebUIPeer.PodSelector.MatchLabels, map[string]string{
		"app.kubernetes.io/name": "hairem-openwebui",
	}) {
		t.Fatalf("private Hermes Open WebUI ingress peer=%#v", openWebUIPeer)
	}

	agent.Spec.Runtime.PrivateAPI = nil
	if err := r.ensurePrivateHermesAPIResources(ctx, agent, tenant, namespace, true); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: namespace}, &corev1.Service{}); !apierrors.IsNotFound(err) {
		t.Fatalf("disabled opt-in left Service behind: %v", err)
	}
	if err := c.Get(ctx, types.NamespacedName{Name: ingress.Name, Namespace: namespace}, &networkingv1.NetworkPolicy{}); !apierrors.IsNotFound(err) {
		t.Fatalf("disabled opt-in left ingress policy behind: %v", err)
	}
}

func TestPrivateHermesAPICredentialIssuesOnlyCanonicalAgentOwnedSecret(t *testing.T) {
	agent, tenant := privateAPIFixture()
	agent.Spec.Runtime.PrivateAPI = validFuturePrivateAPI(agent, tenant)
	ctx := context.Background()
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, APIReader: c, Scheme: c.Scheme()}

	ready, err := r.ensurePrivateHermesAPICredential(ctx, agent, tenant, tenantNamespace(tenant.Name))
	if err != nil || !ready {
		t.Fatalf("canonical opt-in credential was not issued: ready=%t err=%v", ready, err)
	}
	var credential corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Name: "hermes-api-" + agent.Spec.AgentKey, Namespace: tenantNamespace(tenant.Name)}, &credential); err != nil {
		t.Fatal(err)
	}
	if credential.Type != corev1.SecretTypeOpaque || len(credential.Data) != 1 || len(credential.Data["api-key"]) < 32 {
		t.Fatalf("credential must be an opaque single random api-key secret: %#v", credential)
	}
	owner := metav1.GetControllerOf(&credential)
	if owner == nil || owner.APIVersion != fabricv1alpha1.GroupVersion.String() || owner.Kind != "AgentIdentity" || owner.Name != agent.Name || owner.UID != agent.UID {
		t.Fatalf("credential owner=%#v, want exact AgentIdentity controller owner", owner)
	}
	if credential.Annotations[privateHermesAPIIssuanceIDAnnotation] == "" || credential.Annotations[privateHermesAPIIssuanceRevisionAnnotation] != "3" {
		t.Fatalf("credential lacks opaque issuance provenance: %#v", credential.Annotations)
	}
}

func TestPrivateHermesAPIRuntimeEnvAndPortRequireIssuedCredential(t *testing.T) {
	agent, tenant := privateAPIFixture()
	agent.Spec.Runtime.PrivateAPI = validFuturePrivateAPI(agent, tenant)
	ctx := context.Background()
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, APIReader: c, Scheme: c.Scheme()}
	namespace := tenantNamespace(tenant.Name)
	if err := r.ensureDeployment(ctx, agent, tenant, testRuntimeProfile(), namespace, "model-secret-uid", "baseline"); err != nil {
		t.Fatal(err)
	}
	if err := r.ensurePrivateHermesAPIRuntime(ctx, agent, tenant, namespace, false); err != nil {
		t.Fatal(err)
	}
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &deployment); err != nil {
		t.Fatal(err)
	}
	for _, item := range deployment.Spec.Template.Spec.Containers[0].Env {
		if strings.HasPrefix(item.Name, "API_SERVER_") {
			t.Fatalf("unissued credential enabled API env: %s", item.Name)
		}
	}
	if err := r.ensurePrivateHermesAPIRuntime(ctx, agent, tenant, namespace, true); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &deployment); err != nil {
		t.Fatal(err)
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	hasAPIPort := false
	for _, port := range container.Ports {
		hasAPIPort = hasAPIPort || (port.Name == "responses" && port.ContainerPort == privateHermesAPIPort && port.Protocol == corev1.ProtocolTCP)
	}
	if !hasAPIPort {
		t.Fatalf("issued credential must expose canonical API port: %#v", container.Ports)
	}
	var apiKey *corev1.EnvVar
	for i := range container.Env {
		if container.Env[i].Name == "API_SERVER_KEY" {
			apiKey = &container.Env[i]
		}
	}
	if apiKey == nil || apiKey.ValueFrom == nil || apiKey.ValueFrom.SecretKeyRef == nil || apiKey.ValueFrom.SecretKeyRef.Name != "hermes-api-"+agent.Spec.AgentKey || apiKey.ValueFrom.SecretKeyRef.Key != "api-key" {
		t.Fatalf("API key env is not canonical SecretKeyRef: %#v", apiKey)
	}
}

func TestPrivateHermesAPIRuntimeDoesNotUpdateUnchangedDeployment(t *testing.T) {
	agent, tenant := privateAPIFixture()
	agent.Spec.Runtime.PrivateAPI = validFuturePrivateAPI(agent, tenant)
	ctx := context.Background()
	namespace := tenantNamespace(tenant.Name)
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, APIReader: c, Scheme: c.Scheme()}
	if err := r.ensureDeployment(ctx, agent, tenant, testRuntimeProfile(), namespace, "model-secret-uid", "baseline"); err != nil {
		t.Fatal(err)
	}
	if err := r.ensurePrivateHermesAPIRuntime(ctx, agent, tenant, namespace, true); err != nil {
		t.Fatal(err)
	}
	var first appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &first); err != nil {
		t.Fatal(err)
	}
	if err := r.ensurePrivateHermesAPIRuntime(ctx, agent, tenant, namespace, true); err != nil {
		t.Fatal(err)
	}
	var second appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}, &second); err != nil {
		t.Fatal(err)
	}
	if second.ResourceVersion != first.ResourceVersion {
		t.Fatalf("unchanged private Hermes API runtime rewrote Deployment: first=%s second=%s", first.ResourceVersion, second.ResourceVersion)
	}
}

func TestPrivateHermesAPICredentialRejectsForgedLabelsAndMissingIssuance(t *testing.T) {
	for _, tc := range []struct {
		name   string
		secret *corev1.Secret
	}{
		{"forged labels", &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "hermes-api-tesla", Namespace: "tenant-acme", Labels: map[string]string{LabelName: "hermes-agent", LabelInstance: "tesla"}}}},
		{"missing issuance", &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "hermes-api-tesla", Namespace: "tenant-acme", OwnerReferences: []metav1.OwnerReference{{APIVersion: fabricv1alpha1.GroupVersion.String(), Kind: "AgentIdentity", Name: "acme-tesla", UID: types.UID("f0f0f0f0-0000-4000-8000-000000000001"), Controller: boolPtr(true)}}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			agent, tenant := privateAPIFixture()
			agent.Spec.Runtime.PrivateAPI = validFuturePrivateAPI(agent, tenant)
			tc.secret.Name = privateHermesAPICredentialPrefix + agent.Spec.AgentKey
			tc.secret.Namespace = tenantNamespace(tenant.Name)
			c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(agent, tc.secret).Build()
			r := &AgentIdentityReconciler{Client: c, APIReader: privateHermesAPIMetadataReader{Reader: c, metadata: map[types.NamespacedName]*metav1.PartialObjectMetadata{client.ObjectKeyFromObject(tc.secret): secretMetadata(tc.secret)}}, Scheme: c.Scheme()}
			ready, err := r.ensurePrivateHermesAPICredential(context.Background(), agent, tenant, tenantNamespace(tenant.Name))
			if err == nil || ready {
				t.Fatalf("forged or issuer-less credential must fail closed: ready=%t err=%v", ready, err)
			}
		})
	}
}

func TestPrivateHermesAPIEndpointAttestationRequiresCompleteRuntimeChain(t *testing.T) {
	ctx := context.Background()
	agent, tenant := privateAPIFixture()
	agent.Spec.Runtime.PrivateAPI = validFuturePrivateAPI(agent, tenant)
	namespace := tenantNamespace(tenant.Name)
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, APIReader: c, Scheme: c.Scheme()}
	if err := r.ensureDeployment(ctx, agent, tenant, testRuntimeProfile(), namespace, "model-secret-uid", "baseline"); err != nil {
		t.Fatal(err)
	}
	if err := r.ensurePrivateHermesAPIRuntime(ctx, agent, tenant, namespace, true); err != nil {
		t.Fatal(err)
	}
	if err := r.ensurePrivateHermesAPIResources(ctx, agent, tenant, namespace, true); err != nil {
		t.Fatal(err)
	}
	ready, err := r.privateHermesAPIEndpointAttested(ctx, agent, tenant, namespace)
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("missing rollout status, EndpointSlice and Ready Pod must fail closed")
	}
}

func TestPrivateHermesAPIEndpointAttestationAcceptsCanonicalReadyRuntime(t *testing.T) {
	ctx := context.Background()
	agent, tenant := privateAPIFixture()
	agent.Spec.Runtime.PrivateAPI = validFuturePrivateAPI(agent, tenant)
	namespace := tenantNamespace(tenant.Name)
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithStatusSubresource(&appsv1.Deployment{}).WithObjects(agent).Build()
	r := &AgentIdentityReconciler{Client: c, APIReader: c, Scheme: c.Scheme()}
	if err := r.ensureDeployment(ctx, agent, tenant, testRuntimeProfile(), namespace, "model-secret-uid", "baseline"); err != nil {
		t.Fatal(err)
	}
	if err := r.ensurePrivateHermesAPIRuntime(ctx, agent, tenant, namespace, true); err != nil {
		t.Fatal(err)
	}
	if err := r.ensurePrivateHermesAPIResources(ctx, agent, tenant, namespace, true); err != nil {
		t.Fatal(err)
	}

	name := runtimeName(agent.Spec.AgentKey)
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &deployment); err != nil {
		t.Fatal(err)
	}
	deployment.UID = types.UID("f0f0f0f0-0000-4000-8000-000000000010")
	if err := c.Update(ctx, &deployment); err != nil {
		t.Fatal(err)
	}
	deployment.Status.ObservedGeneration = deployment.Generation
	deployment.Status.UpdatedReplicas = 1
	deployment.Status.AvailableReplicas = 1
	if err := c.Status().Update(ctx, &deployment); err != nil {
		t.Fatal(err)
	}
	var service corev1.Service
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &service); err != nil {
		t.Fatal(err)
	}
	service.UID = types.UID("f0f0f0f0-0000-4000-8000-000000000011")
	if err := c.Update(ctx, &service); err != nil {
		t.Fatal(err)
	}
	replicaSet := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
		Name:            name + "-unit",
		Namespace:       namespace,
		UID:             types.UID("f0f0f0f0-0000-4000-8000-000000000012"),
		OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(&deployment, appsv1.SchemeGroupVersion.WithKind("Deployment"))},
	}}
	replicaSet.Spec.Template = *deployment.Spec.Template.DeepCopy()
	replicaSet.Spec.Template.Labels[appsv1.DefaultDeploymentUniqueLabelKey] = "unit"

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name + "-unit",
			Namespace:       namespace,
			UID:             types.UID("f0f0f0f0-0000-4000-8000-000000000099"),
			Labels:          map[string]string{LabelName: "hermes-agent", LabelInstance: agent.Spec.AgentKey},
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(replicaSet, appsv1.SchemeGroupVersion.WithKind("ReplicaSet"))},
		},
		Spec:   corev1.PodSpec{Containers: []corev1.Container{{Name: "hermes", Ports: []corev1.ContainerPort{{Name: "responses", ContainerPort: privateHermesAPIPort, Protocol: corev1.ProtocolTCP}}}}},
		Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}},
	}
	port := int32(privateHermesAPIPort)
	slice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-unit",
			Namespace: namespace,
			Labels: map[string]string{
				discoveryv1.LabelServiceName: name,
				discoveryv1.LabelManagedBy:   "endpointslice-controller.k8s.io",
			},
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(&service, corev1.SchemeGroupVersion.WithKind("Service"))},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Ports:       []discoveryv1.EndpointPort{{Name: stringPtr("responses"), Protocol: protocolPtr(corev1.ProtocolTCP), Port: &port}},
		Endpoints:   []discoveryv1.Endpoint{{Addresses: []string{"10.42.0.99"}, Conditions: discoveryv1.EndpointConditions{Ready: boolPtr(true)}, TargetRef: &corev1.ObjectReference{Kind: "Pod", Namespace: namespace, Name: pod.Name, UID: pod.UID}}},
	}
	if err := c.Create(ctx, replicaSet); err != nil {
		t.Fatal(err)
	}
	if err := c.Create(ctx, pod); err != nil {
		t.Fatal(err)
	}
	if err := c.Create(ctx, slice); err != nil {
		t.Fatal(err)
	}
	ready, err := r.privateHermesAPIEndpointAttested(ctx, agent, tenant, namespace)
	if err != nil || !ready {
		t.Fatalf("canonical Ready runtime must attest: ready=%t err=%v", ready, err)
	}
}

func TestPrivateHermesAPIEndpointConditionRequiresAttestation(t *testing.T) {
	agent, tenant := privateAPIFixture()
	agent.Spec.Runtime.PrivateAPI = validFuturePrivateAPI(agent, tenant)
	setPrivateHermesAPIEndpointCondition(agent, tenant, false)
	condition := apiMeta.FindStatusCondition(agent.Status.Conditions, privateHermesAPICondition)
	if condition == nil || condition.Status != metav1.ConditionFalse || condition.Reason != "NotReady" {
		t.Fatalf("unattested condition=%#v", condition)
	}
	setPrivateHermesAPIEndpointCondition(agent, tenant, true)
	condition = apiMeta.FindStatusCondition(agent.Status.Conditions, privateHermesAPICondition)
	if condition == nil || condition.Status != metav1.ConditionTrue || condition.Reason != "EndpointAttested" {
		t.Fatalf("attested condition=%#v", condition)
	}
}

func TestPrivateHermesAPIEndpointAttestationRejectsForeignReplicaSetPod(t *testing.T) {
	ctx := context.Background()
	agent, tenant := privateAPIFixture()
	agent.Spec.Runtime.PrivateAPI = validFuturePrivateAPI(agent, tenant)
	namespace := tenantNamespace(tenant.Name)
	name := runtimeName(agent.Spec.AgentKey)
	replicas := int32(1)
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: types.UID("f0f0f0f0-0000-4000-8000-000000000020"), Generation: 1, OwnerReferences: []metav1.OwnerReference{{APIVersion: fabricv1alpha1.GroupVersion.String(), Kind: "AgentIdentity", Name: agent.Name, UID: agent.UID, Controller: boolPtr(true)}}}, Spec: appsv1.DeploymentSpec{Replicas: &replicas}, Status: appsv1.DeploymentStatus{ObservedGeneration: 1, UpdatedReplicas: 1, AvailableReplicas: 1}}
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: types.UID("f0f0f0f0-0000-4000-8000-000000000021"), OwnerReferences: []metav1.OwnerReference{{APIVersion: fabricv1alpha1.GroupVersion.String(), Kind: "AgentIdentity", Name: agent.Name, UID: agent.UID, Controller: boolPtr(true)}}}, Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, Selector: map[string]string{LabelName: "hermes-agent", LabelInstance: agent.Spec.AgentKey}, Ports: []corev1.ServicePort{{Name: "responses", Protocol: corev1.ProtocolTCP, Port: privateHermesAPIPort, TargetPort: *intOrStringPtr(privateHermesAPIPort)}}}}
	foreignDeployment := deployment.DeepCopy()
	foreignDeployment.UID = types.UID("f0f0f0f0-0000-4000-8000-000000000022")
	foreignReplicaSet := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: name + "-stale", Namespace: namespace, UID: types.UID("f0f0f0f0-0000-4000-8000-000000000023"), OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(foreignDeployment, appsv1.SchemeGroupVersion.WithKind("Deployment"))}}}
	foreignReplicaSet.Spec.Template.Labels = map[string]string{"revision": "stale"}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name + "-stale", Namespace: namespace, UID: types.UID("f0f0f0f0-0000-4000-8000-000000000024"), Labels: map[string]string{LabelName: "hermes-agent", LabelInstance: agent.Spec.AgentKey}, OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(foreignReplicaSet, appsv1.SchemeGroupVersion.WithKind("ReplicaSet"))}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "hermes", Ports: []corev1.ContainerPort{{Name: "responses", ContainerPort: privateHermesAPIPort, Protocol: corev1.ProtocolTCP}}}}}, Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}}}
	port := int32(privateHermesAPIPort)
	slice := &discoveryv1.EndpointSlice{ObjectMeta: metav1.ObjectMeta{Name: name + "-stale", Namespace: namespace, Labels: map[string]string{discoveryv1.LabelServiceName: name, discoveryv1.LabelManagedBy: "endpointslice-controller.k8s.io"}, OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(service, corev1.SchemeGroupVersion.WithKind("Service"))}}, AddressType: discoveryv1.AddressTypeIPv4, Ports: []discoveryv1.EndpointPort{{Name: stringPtr("responses"), Protocol: protocolPtr(corev1.ProtocolTCP), Port: &port}}, Endpoints: []discoveryv1.Endpoint{{Addresses: []string{"10.42.0.20"}, Conditions: discoveryv1.EndpointConditions{Ready: boolPtr(true)}, TargetRef: &corev1.ObjectReference{Kind: "Pod", Namespace: namespace, Name: pod.Name, UID: pod.UID}}}}
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(agent, deployment, service, foreignReplicaSet, pod, slice).Build()
	ready, err := privateHermesAPIEndpointAttested(ctx, c, agent, tenant, namespace)
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("foreign or stale correctly labeled Pod and EndpointSlice must not attest")
	}
	foreignReplicaSet.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(deployment, appsv1.SchemeGroupVersion.WithKind("Deployment"))}
	if err := c.Update(ctx, foreignReplicaSet); err != nil {
		t.Fatal(err)
	}
	ready, err = privateHermesAPIEndpointAttested(ctx, c, agent, tenant, namespace)
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("stale ReplicaSet Pod must not attest after the Deployment has rolled forward")
	}
	foreignReplicaSet.Spec.Template = *deployment.Spec.Template.DeepCopy()
	foreignReplicaSet.Spec.Template.Labels = map[string]string{appsv1.DefaultDeploymentUniqueLabelKey: "current"}
	if err := c.Update(ctx, foreignReplicaSet); err != nil {
		t.Fatal(err)
	}
	slice.Labels[discoveryv1.LabelManagedBy] = "foreign-endpoint-controller"
	if err := c.Update(ctx, slice); err != nil {
		t.Fatal(err)
	}
	ready, err = privateHermesAPIEndpointAttested(ctx, c, agent, tenant, namespace)
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("foreign or unmanaged Service-labeled EndpointSlice must not attest")
	}
	slice.Labels[discoveryv1.LabelManagedBy] = "endpointslice-controller.k8s.io"
	slice.OwnerReferences = nil
	if err := c.Update(ctx, slice); err != nil {
		t.Fatal(err)
	}
	ready, err = privateHermesAPIEndpointAttested(ctx, c, agent, tenant, namespace)
	if err != nil {
		t.Fatal(err)
	}
	if ready {
		t.Fatal("unowned Service-labeled EndpointSlice must not attest")
	}
}
