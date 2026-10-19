package controller

import (
	"context"
	"fmt"
	"regexp"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	privateHermesAPICondition        = "PrivateHermesAPIReady"
	privateHermesAPICredentialPrefix = "hermes-api-"
	privateHermesAPIPort             = 8642
	privateHermesAPIBFFNamespace     = "txo-fabric-system"
)

var (
	privateHermesAPITenantID   = regexp.MustCompile(`^TEN[0-9]{5,}$`)
	privateHermesAPIObjectName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
)

// privateHermesAPIIntent validates non-secret opt-in intent. A valid opt-in
// gets only the operator-owned Service and ingress NetworkPolicy substrate;
// it never launches a listener, reads/mounts a credential, or attests an
// endpoint. Endpoint readiness remains a separate reviewed contract.
func privateHermesAPIIntent(agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) (string, string) {
	if agent == nil || tenant == nil {
		return "InvalidConfiguration", "canonical AgentIdentity and TenantBundle are required"
	}
	api := agent.Spec.Runtime.PrivateAPI
	if api == nil {
		return "Disabled", "private Hermes API is disabled"
	}
	if !api.Enabled {
		if api.TenantID != "" || api.CredentialRef != "" {
			return "InvalidConfiguration", "disabled private API cannot declare credential or tenant bindings"
		}
		return "Disabled", "private Hermes API is disabled"
	}
	// Identity must be read from reconciled Kubernetes objects. No browser
	// tenant/agent/path/endpoint/Service label is accepted as authority.
	if agent.Spec.TenantRef.Name != tenant.Name ||
		agent.Name == "" || agent.UID == "" ||
		agent.Generation < 1 || tenant.UID == "" ||
		tenant.Spec.TenantID == "" ||
		api.TenantID != tenant.Spec.TenantID ||
		len(api.TenantID) > 64 || !privateHermesAPITenantID.MatchString(api.TenantID) ||
		len(agent.Spec.AgentKey) < 1 || len(agent.Spec.AgentKey) > 63 ||
		!privateHermesAPIObjectName.MatchString(agent.Spec.AgentKey) ||
		!privateHermesAPIObjectName.MatchString(tenant.Name) ||
		tenant.DeletionTimestamp != nil || agent.DeletionTimestamp != nil {
		return "InvalidConfiguration", "private API canonical tenant or AgentIdentity binding is invalid"
	}
	expectedRef := privateHermesAPICredentialPrefix + agent.Spec.AgentKey
	if len(expectedRef) > 63 || api.CredentialRef != expectedRef ||
		!privateHermesAPIObjectName.MatchString(api.CredentialRef) {
		return "InvalidConfiguration", "private API credential reference must use the canonical agent key"
	}
	return "NotReady", "private Hermes API network substrate is reconciled; runtime launch and endpoint attestation are not implemented"
}

// ensurePrivateHermesAPIResources materializes the non-secret, private network
// substrate only for a canonical opt-in. The Service name is intentionally
// runtimeName(agentKey), producing hermes-<agent>.tenant-<tenant>.svc:8642.
// The ingress policy admits only the pinned Fabric BFF workload to that exact
// agent pod. It does not imply that a Responses listener or credential exists.
func (r *AgentIdentityReconciler) ensurePrivateHermesAPIResources(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, namespace string) error {
	name := runtimeName(agent.Spec.AgentKey)
	ingressName := name + "-responses-ingress"
	reason, _ := privateHermesAPIIntent(agent, tenant)
	if reason != "NotReady" {
		if err := r.deletePrivateHermesAPIResource(ctx, &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: ingressName, Namespace: namespace}}, agent); err != nil {
			return err
		}
		return r.deletePrivateHermesAPIResource(ctx, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}, agent)
	}

	selector := map[string]string{LabelName: "hermes-agent", LabelInstance: agent.Spec.AgentKey}
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		service.Labels = mergeStringMap(service.Labels, agentLabels(agent, tenant))
		service.Spec.Type = corev1.ServiceTypeClusterIP
		service.Spec.Selector = selector
		service.Spec.Ports = []corev1.ServicePort{{
			Name:       "responses",
			Protocol:   corev1.ProtocolTCP,
			Port:       privateHermesAPIPort,
			TargetPort: *intOrStringPtr(privateHermesAPIPort),
		}}
		return controllerutil.SetControllerReference(agent, service, r.Scheme)
	}); err != nil {
		return err
	}

	ingress := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: ingressName, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		ingress.Labels = mergeStringMap(ingress.Labels, agentLabels(agent, tenant))
		ingress.Spec = networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: selector},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": privateHermesAPIBFFNamespace}},
					PodSelector:       &metav1.LabelSelector{MatchLabels: map[string]string{LabelName: "txo-fabric-webui-bff", "app.kubernetes.io/component": "bff"}},
				}},
				Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(privateHermesAPIPort)}},
			}},
		}
		return controllerutil.SetControllerReference(agent, ingress, r.Scheme)
	})
	return err
}

// deletePrivateHermesAPIResource only removes a resource previously owned by
// this AgentIdentity. An invalid or withdrawn opt-in must never delete a
// colliding Service or NetworkPolicy belonging to another controller.
func (r *AgentIdentityReconciler) deletePrivateHermesAPIResource(ctx context.Context, object client.Object, agent *fabricv1alpha1.AgentIdentity) error {
	if err := r.Get(ctx, client.ObjectKeyFromObject(object), object); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	owner := metav1.GetControllerOf(object)
	if owner == nil || owner.UID != agent.UID || owner.Kind != "AgentIdentity" {
		return fmt.Errorf("private Hermes API resource %s/%s is not owned by this AgentIdentity", object.GetNamespace(), object.GetName())
	}
	return r.Delete(ctx, object)
}

func setPrivateHermesAPICondition(agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) {
	reason, message := privateHermesAPIIntent(agent, tenant)
	setCondition(&agent.Status.Conditions, agent.Generation, privateHermesAPICondition,
		metav1.ConditionFalse, reason, message)
}
