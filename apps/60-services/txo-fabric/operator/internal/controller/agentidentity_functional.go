package controller

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
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
func (r *AgentIdentityReconciler) withdrawInvalidFunctionalRuntime(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, namespace string) error {
	if err := r.ensureHumanAccessResources(ctx, agent, tenant, namespace, humanAccessResolution{}); err != nil {
		return err
	}
	var dep appsv1.Deployment
	key := types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace}
	if err := r.Get(ctx, key, &dep); err != nil {
		return client.IgnoreNotFound(err)
	}
	if dep.Labels[LabelInstance] != agent.Spec.AgentKey || dep.Labels[LabelTenantName] != tenant.Name {
		return fmt.Errorf("refusing to withdraw unmanaged deployment %s/%s", namespace, dep.Name)
	}
	if !dep.DeletionTimestamp.IsZero() {
		return nil
	}
	return r.Delete(ctx, &dep)
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
