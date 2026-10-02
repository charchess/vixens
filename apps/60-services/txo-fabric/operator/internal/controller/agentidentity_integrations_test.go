package controller

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testIntegrationSecretValue = "super-secret-canary-token"

func TestResolveIntegrationAccessHappyPathAndRuntimeProjection(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	agent := testAgentIdentity()
	connection := testIntegrationConnection(tenant.Name)
	binding := testIntegrationBinding(agent.Name, tenant.Name, connection.Name)
	credential := testIntegrationCredential(tenant.Name, connection.Name, connection.Spec.CredentialRef.Name)

	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(connection, binding, credential).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: testScheme(t)}

	resolved, err := r.resolveIntegrationAccess(ctx, agent, tenant, tenantNamespace(tenant.Name))
	if err != nil {
		t.Fatal(err)
	}
	if !resolved.Ready || !resolved.HasBindings || len(resolved.Effective) != 1 || len(resolved.Status) != 1 {
		t.Fatalf("unexpected integration resolution: %#v", resolved)
	}
	if got := resolved.Status[0]; got.Phase != "Effective" || got.Reason != "Authorized" || got.Revision == "" {
		t.Fatalf("unexpected effective status: %#v", got)
	}

	profile := testRuntimeProfile()
	toolPolicy, err := resolveToolsetPolicy(agent, profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ensureDeploymentWithIntegrations(ctx, agent, tenant, profile, tenantNamespace(tenant.Name), "", "", toolPolicy, resolved); err != nil {
		t.Fatal(err)
	}

	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: tenantNamespace(tenant.Name)}, &deployment); err != nil {
		t.Fatal(err)
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	manifest := envValue(container.Env, integrationManifestEnv)
	if manifest == "" {
		t.Fatal("runtime integration manifest is missing")
	}
	var entries []runtimeIntegrationManifestEntry
	if err := json.Unmarshal([]byte(manifest), &entries); err != nil {
		t.Fatalf("integration manifest is not valid JSON: %v", err)
	}
	if len(entries) != 1 || entries[0].Connection != connection.Name || entries[0].CredentialPath != integrationCredentialMountRoot+"/"+connection.Name+"/"+integrationCredentialKey {
		t.Fatalf("unexpected integration manifest: %#v", entries)
	}
	if strings.Contains(manifest, testIntegrationSecretValue) {
		t.Fatal("runtime manifest leaked integration credential value")
	}
	if got := deployment.Spec.Template.Annotations[AnnotationIntegrationPolicyRevision]; got != resolved.Revision {
		t.Fatalf("integration revision annotation=%q want=%q", got, resolved.Revision)
	}
	if !deploymentReferencesSecret(&deployment, credential.Name) {
		t.Fatalf("deployment does not project credential Secret %q", credential.Name)
	}
	serialized, err := json.Marshal(deployment)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(serialized), testIntegrationSecretValue) {
		t.Fatal("Deployment leaked integration credential value")
	}

	if err := r.ensureEgressPolicy(ctx, agent, tenant, tenantNamespace(tenant.Name), resolved); err != nil {
		t.Fatal(err)
	}
	var policy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey) + "-egress", Namespace: tenantNamespace(tenant.Name)}, &policy); err != nil {
		t.Fatal(err)
	}
	if len(policy.Spec.Egress) != 4 {
		t.Fatalf("effective HTTP integration should add temporary POC HTTP(S) egress: got %d rules", len(policy.Spec.Egress))
	}
}

func TestResolveIntegrationAccessUnboundAgentHasNoAuthorization(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	agent := testAgentIdentity()
	connection := testIntegrationConnection(tenant.Name)
	binding := testIntegrationBinding("hairem-sandbox-other", tenant.Name, connection.Name)
	credential := testIntegrationCredential(tenant.Name, connection.Name, connection.Spec.CredentialRef.Name)

	r := integrationTestReconciler(t, connection, binding, credential)
	resolved, err := r.resolveIntegrationAccess(ctx, agent, tenant, tenantNamespace(tenant.Name))
	if err != nil {
		t.Fatal(err)
	}
	if resolved.HasBindings || !resolved.Ready || len(resolved.Effective) != 0 || len(resolved.Status) != 0 {
		t.Fatalf("unbound agent unexpectedly received integration authorization: %#v", resolved)
	}
}

func TestResolveIntegrationAccessTenantMismatchFailsClosed(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	agent := testAgentIdentity()
	connection := testIntegrationConnection(tenant.Name)
	binding := testIntegrationBinding(agent.Name, "other-tenant", connection.Name)
	credential := testIntegrationCredential(tenant.Name, connection.Name, connection.Spec.CredentialRef.Name)

	r := integrationTestReconciler(t, connection, binding, credential)
	resolved, err := r.resolveIntegrationAccess(ctx, agent, tenant, tenantNamespace(tenant.Name))
	if err != nil {
		t.Fatal(err)
	}
	assertDeniedIntegration(t, resolved, "TenantMismatch")
}

func TestResolveIntegrationAccessMissingConnectionFailsClosed(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	agent := testAgentIdentity()
	binding := testIntegrationBinding(agent.Name, tenant.Name, "missing-connection")

	r := integrationTestReconciler(t, binding)
	resolved, err := r.resolveIntegrationAccess(ctx, agent, tenant, tenantNamespace(tenant.Name))
	if err != nil {
		t.Fatal(err)
	}
	assertDeniedIntegration(t, resolved, "ConnectionNotFound")
}

func TestResolveIntegrationAccessInvalidBindingFailsClosed(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	agent := testAgentIdentity()
	connection := testIntegrationConnection(tenant.Name)
	binding := testIntegrationBinding(agent.Name, tenant.Name, connection.Name)
	binding.Spec.Operations = nil
	credential := testIntegrationCredential(tenant.Name, connection.Name, connection.Spec.CredentialRef.Name)

	r := integrationTestReconciler(t, connection, binding, credential)
	resolved, err := r.resolveIntegrationAccess(ctx, agent, tenant, tenantNamespace(tenant.Name))
	if err != nil {
		t.Fatal(err)
	}
	assertDeniedIntegration(t, resolved, "BindingInvalid")
}

func TestResolveIntegrationAccessMissingCredentialFailsClosed(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	agent := testAgentIdentity()
	connection := testIntegrationConnection(tenant.Name)
	binding := testIntegrationBinding(agent.Name, tenant.Name, connection.Name)

	r := integrationTestReconciler(t, connection, binding)
	resolved, err := r.resolveIntegrationAccess(ctx, agent, tenant, tenantNamespace(tenant.Name))
	if err != nil {
		t.Fatal(err)
	}
	assertDeniedIntegration(t, resolved, "CredentialMissing")
}

func TestIntegrationRevocationAndRestoreAreIndependentFromPVCState(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	agent := testAgentIdentity()
	connection := testIntegrationConnection(tenant.Name)
	binding := testIntegrationBinding(agent.Name, tenant.Name, connection.Name)
	credential := testIntegrationCredential(tenant.Name, connection.Name, connection.Spec.CredentialRef.Name)
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
		Name:      runtimePVCName(agent.Spec.AgentKey),
		Namespace: tenantNamespace(tenant.Name),
	}}

	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(connection, binding, credential, pvc).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}

	active, err := r.resolveIntegrationAccess(ctx, agent, tenant, tenantNamespace(tenant.Name))
	if err != nil {
		t.Fatal(err)
	}
	if len(active.Effective) != 1 {
		t.Fatalf("active binding not effective: %#v", active)
	}

	var current fabricv1alpha1.IntegrationBinding
	if err := c.Get(ctx, types.NamespacedName{Name: binding.Name}, &current); err != nil {
		t.Fatal(err)
	}
	current.Spec.State = fabricv1alpha1.IntegrationBindingStateRevoked
	if err := c.Update(ctx, &current); err != nil {
		t.Fatal(err)
	}

	revoked, err := r.resolveIntegrationAccess(ctx, agent, tenant, tenantNamespace(tenant.Name))
	if err != nil {
		t.Fatal(err)
	}
	if !revoked.Ready || len(revoked.Effective) != 0 || len(revoked.Status) != 1 || revoked.Status[0].Phase != "Revoked" {
		t.Fatalf("revoked binding was not removed from runtime authorization: %#v", revoked)
	}
	var retained corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, &retained); err != nil {
		t.Fatalf("revocation touched retained private PVC: %v", err)
	}

	profile := testRuntimeProfile()
	toolPolicy, err := resolveToolsetPolicy(agent, profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ensureDeploymentWithIntegrations(ctx, agent, tenant, profile, tenantNamespace(tenant.Name), "", "", toolPolicy, revoked); err != nil {
		t.Fatal(err)
	}
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: tenantNamespace(tenant.Name)}, &deployment); err != nil {
		t.Fatal(err)
	}
	if deploymentReferencesSecret(&deployment, credential.Name) {
		t.Fatal("revoked binding still projects integration credential")
	}
	if got := envValue(deployment.Spec.Template.Spec.Containers[0].Env, integrationManifestEnv); got != "[]" {
		t.Fatalf("revoked runtime manifest=%q want=[]", got)
	}
	if err := r.ensureEgressPolicy(ctx, agent, tenant, tenantNamespace(tenant.Name), revoked); err != nil {
		t.Fatal(err)
	}
	var policy networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey) + "-egress", Namespace: tenantNamespace(tenant.Name)}, &policy); err != nil {
		t.Fatal(err)
	}
	if len(policy.Spec.Egress) != 3 {
		t.Fatalf("revocation should remove temporary integration egress: got %d rules", len(policy.Spec.Egress))
	}

	if err := c.Get(ctx, types.NamespacedName{Name: binding.Name}, &current); err != nil {
		t.Fatal(err)
	}
	current.Spec.State = fabricv1alpha1.IntegrationBindingStateActive
	if err := c.Update(ctx, &current); err != nil {
		t.Fatal(err)
	}
	restored, err := r.resolveIntegrationAccess(ctx, agent, tenant, tenantNamespace(tenant.Name))
	if err != nil {
		t.Fatal(err)
	}
	if !restored.Ready || len(restored.Effective) != 1 || restored.Status[0].Phase != "Effective" {
		t.Fatalf("restored binding did not reconcile authorization: %#v", restored)
	}
}

func integrationTestReconciler(t *testing.T, objects ...client.Object) *AgentIdentityReconciler {
	t.Helper()
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
	return &AgentIdentityReconciler{Client: c, Scheme: scheme}
}

func testIntegrationConnection(tenantName string) *fabricv1alpha1.IntegrationConnection {
	return &fabricv1alpha1.IntegrationConnection{
		ObjectMeta: metav1.ObjectMeta{Name: "http-canary"},
		Spec: fabricv1alpha1.IntegrationConnectionSpec{
			TenantRef:      fabricv1alpha1.ObjectReference{Name: tenantName},
			Protocol:       fabricv1alpha1.IntegrationProtocolHTTP,
			Endpoint:       "http://http-canary.tenant-" + tenantName + ".svc:80/",
			Authentication: fabricv1alpha1.IntegrationAuthenticationBearer,
			CredentialRef:  fabricv1alpha1.ObjectReference{Name: "http-canary-credential"},
		},
	}
}

func testIntegrationBinding(agentName, tenantName, connectionName string) *fabricv1alpha1.IntegrationBinding {
	return &fabricv1alpha1.IntegrationBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "http-canary-binding"},
		Spec: fabricv1alpha1.IntegrationBindingSpec{
			TenantRef:     fabricv1alpha1.ObjectReference{Name: tenantName},
			AgentRef:      fabricv1alpha1.ObjectReference{Name: agentName},
			ConnectionRef: fabricv1alpha1.ObjectReference{Name: connectionName},
			State:         fabricv1alpha1.IntegrationBindingStateActive,
			Operations:    []string{"GET"},
			Scopes:        []string{"canary.read"},
		},
	}
}

func testIntegrationCredential(tenantName, connectionName, name string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       tenantNamespace(tenantName),
			UID:             types.UID("integration-credential-uid"),
			ResourceVersion: "7",
			Labels: map[string]string{
				LabelIntegrationCredential: "true",
				LabelTenantName:             tenantName,
				LabelIntegrationConnection: connectionName,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{integrationCredentialKey: []byte(testIntegrationSecretValue)},
	}
}

func assertDeniedIntegration(t *testing.T, resolved integrationResolution, reason string) {
	t.Helper()
	if resolved.Ready || !resolved.HasBindings || len(resolved.Effective) != 0 || len(resolved.Status) != 1 {
		t.Fatalf("integration did not fail closed: %#v", resolved)
	}
	if resolved.Status[0].Phase != "Denied" || resolved.Status[0].Reason != reason {
		t.Fatalf("denied status=%#v want reason=%s", resolved.Status[0], reason)
	}
}

func deploymentReferencesSecret(deployment *appsv1.Deployment, secretName string) bool {
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.Secret != nil && volume.Secret.SecretName == secretName {
			return true
		}
	}
	return false
}
