package controller

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const AnnotationFunctionalProfileRevision = "fabric.truxonline.io/functional-profile-revision"

type effectiveFunctionalProfile struct {
	Enabled bool
	ProfileName string
	Revision string
	Instructions string
}

// resolveFunctionalProfile never broadens admission. A profile is a tenant-scoped
// set of instructions only; capabilities, skills, IAM and models remain separately governed.
func (r *AgentIdentityReconciler) resolveFunctionalProfile(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) (effectiveFunctionalProfile, error) {
	name := agent.Spec.Functional.ProfileRef
	if name == "" {
		return effectiveFunctionalProfile{}, nil
	}
	var profile fabricv1alpha1.AgentFunctionalProfile
	if err := r.Get(ctx, types.NamespacedName{Name: name}, &profile); err != nil {
		return effectiveFunctionalProfile{}, fmt.Errorf("functional profile %q unavailable: %w", name, err)
	}
	if profile.Spec.TenantRef.Name != tenant.Name {
		return effectiveFunctionalProfile{}, fmt.Errorf("functional profile %q is not owned by tenant %q", name, tenant.Name)
	}
	if len(profile.Spec.Instructions) == 0 || len(profile.Spec.Instructions) > 8192 ||
		strings.TrimSpace(profile.Spec.Instructions) == "" {
		return effectiveFunctionalProfile{}, fmt.Errorf("functional profile %q has invalid instructions", name)
	}
	digest := sha256.Sum256([]byte(tenant.Name + "\x00" + name + "\x00" + profile.Spec.Instructions))
	return effectiveFunctionalProfile{
		Enabled: true,
		ProfileName: name,
		Revision: fmt.Sprintf("%x", digest[:12]),
		Instructions: profile.Spec.Instructions,
	}, nil
}

// No role can report Ready against an upstream or unverified Hermes image:
// such an image would silently ignore TXO_FUNCTIONAL_SYSTEM_PROMPT.
func requireFunctionalPromptCompatibility(role effectiveFunctionalProfile, runtime *fabricv1alpha1.AgentRuntimeProfile) error {
	if role.Enabled && !runtime.Spec.Compatibility.FunctionalPromptOverlay {
		return fmt.Errorf("AgentRuntimeProfile %q has no certified additive functional prompt support", runtime.Name)
	}
	return nil
}

// A previously authorized profile can disappear or be retargeted. Stop the
// existing Hermes runtime and revoke its human route; never silently start
// with broader default instructions or delete the retained private PVC.
// ownedAgentWithdrawalTarget ensures every target is still controlled by the
// exact AgentIdentity UID. An agentKey collision must never revoke the other
// identity's runtime or public route.
func (r *AgentIdentityReconciler) ownedAgentWithdrawalTarget(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, namespace string, target client.Object) (client.Object, error) {
	current, ok := target.DeepCopyObject().(client.Object)
	if !ok {
		return nil, fmt.Errorf("unsupported AgentIdentity withdrawal object %T", target)
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(target), current); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	owner := metav1.GetControllerOf(current)
	if agent.UID == "" || owner == nil || owner.APIVersion != fabricv1alpha1.GroupVersion.String() ||
		owner.Kind != "AgentIdentity" || owner.Name != agent.Name || owner.UID != agent.UID ||
		current.GetNamespace() != namespace ||
		current.GetLabels()[LabelAgent] != agent.Spec.AgentKey ||
		current.GetLabels()[LabelInstance] != agent.Spec.AgentKey ||
		current.GetLabels()[LabelPartOf] != "txo-fabric" {
		return nil, fmt.Errorf("refusing to withdraw unowned or foreign resource %T %s/%s", target, namespace, target.GetName())
	}
	// The Cilium policy is owner/agent labelled but does not historically carry
	// the full tenant label set; other resources must match the tenant exactly.
	if _, cilium := current.(*unstructured.Unstructured); !cilium &&
		current.GetLabels()[LabelTenantName] != agent.Spec.TenantRef.Name {
		return nil, fmt.Errorf("refusing to withdraw resource %T %s/%s with invalid tenant label", target, namespace, target.GetName())
	}
	return current, nil
}

func (r *AgentIdentityReconciler) withdrawInvalidFunctionalRuntime(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, namespace string) error {
	name := humanAccessResourceName(agent.Spec.AgentKey)
	// Preflight every object before any deletion: a conflicting agentKey may
	// select another agent's service or deployment by name.
	objects := []client.Object{
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}},
		&networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}},
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: name + "-ingress", Namespace: namespace}},
		humanOIDCEgressPolicyObject(agent.Spec.AgentKey, namespace),
	}
	var deployment client.Object
	for i, obj := range objects {
		current, err := r.ownedAgentWithdrawalTarget(ctx, agent, namespace, obj)
		if err != nil {
			return err
		}
		if i == 0 {
			deployment = current
		}
	}
	if err := r.ensureHumanAccessResources(ctx, agent, tenant, namespace, humanAccessResolution{}); err != nil {
		return err
	}
	if deployment == nil || !deployment.GetDeletionTimestamp().IsZero() {
		return nil
	}
	if deployment.GetUID() != "" {
		uid := deployment.GetUID()
		return client.IgnoreNotFound(r.Delete(ctx, deployment, client.Preconditions{UID: &uid}))
	}
	return client.IgnoreNotFound(r.Delete(ctx, deployment))
}

// A single tenant profile update fans out only to the agents whose declared
// immutable tenant and profile references both match.
func (r *AgentIdentityReconciler) requestsForFunctionalProfile(ctx context.Context, obj client.Object) []reconcile.Request {
	profile, ok := obj.(*fabricv1alpha1.AgentFunctionalProfile)
	if !ok {
		return nil
	}
	var agents fabricv1alpha1.AgentIdentityList
	if err := r.List(ctx, &agents); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range agents.Items {
		agent := &agents.Items[i]
		if agent.Spec.Functional.ProfileRef == profile.Name &&
			agent.Spec.TenantRef.Name == profile.Spec.TenantRef.Name {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: agent.Name}})
		}
	}
	return requests
}
