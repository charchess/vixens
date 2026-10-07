package controller

import (
	"context"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAgentReconcileAdoptsExistingPVCWithoutRecreating(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	profile := testRuntimeProfile()
	agent := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hairem-sandbox-tina",
			UID:  types.UID("agent-uid"),
		},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef:   fabricv1alpha1.ObjectReference{Name: tenant.Name},
			AgentKey:    "tina",
			DisplayName: "Tina",
			Runtime:     fabricv1alpha1.AgentRuntimeBinding{ProfileRef: profile.Name},
		},
	}
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "tenant-hairem-sandbox"}}
	legacyUID := types.UID("legacy-pvc-uid")
	storageClass := "truenas-iscsi-delete"
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hermes-tina-data",
			Namespace: namespace.Name,
			UID:       legacyUID,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &storageClass,
			Resources:        corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: profile.Spec.Storage.Size}},
		},
	}

	gatewayProfile := aiGatewayTestProfile()
	gatewayRuntimeSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayRuntimeSecretName, Namespace: namespace.Name},
		Data:       map[string][]byte{"LITELLM_MASTER_KEY": []byte("tenant-master")},
	}
	backendID := "tenant:" + tenant.Name + ":" + gatewayProfile.Name + ":4000"
	modelSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      modelAccessSecretName(agent.Spec.AgentKey),
			Namespace: namespace.Name,
			UID:       types.UID("adoption-model-access-uid"),
			Annotations: map[string]string{
				AnnotationModelAccessBackend:    backendID,
				AnnotationModelAccessBackendURL: "http://txo-ai-gateway." + namespace.Name + ".svc:4000",
				AnnotationModelAccessRevision:   modelAccessBackendRevision(backendID, ""),
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{modelAccessSecretKey: []byte("test-adoption-key")},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.TenantBundle{}, &fabricv1alpha1.AgentIdentity{}).
		WithObjects(tenant, profile, agent, namespace, pvc, gatewayProfile, gatewayRuntimeSecret, modelSecret).
		Build()

	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name}}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}

	var current corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, &current); err != nil {
		t.Fatal(err)
	}
	if current.UID != legacyUID {
		t.Fatalf("PVC was recreated: uid=%q, want %q", current.UID, legacyUID)
	}
	owner := metav1.GetControllerOf(&current)
	if owner == nil {
		t.Fatal("existing PVC was not adopted by AgentIdentity")
	}
	if owner.Name != agent.Name || owner.UID != agent.UID {
		t.Fatalf("unexpected PVC controller owner: %#v", owner)
	}
	if current.Labels[LabelInstance] != agent.Spec.AgentKey {
		t.Fatalf("PVC instance label = %q, want %q", current.Labels[LabelInstance], agent.Spec.AgentKey)
	}
}
