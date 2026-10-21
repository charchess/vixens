package controller

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
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
func (r *AgentIdentityReconciler) ensurePrivateHermesAPIResources(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, namespace string, credentialReady bool) error {
	name := runtimeName(agent.Spec.AgentKey)
	ingressName := name + "-responses-ingress"
	reason, _ := privateHermesAPIIntent(agent, tenant)
	if reason != "NotReady" || !credentialReady {
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

const (
	privateHermesAPIIssuanceIDAnnotation       = "fabric.truxonline.io/private-hermes-api-issuance-id"
	privateHermesAPIIssuanceRevisionAnnotation = "fabric.truxonline.io/private-hermes-api-issuance-revision"
)

// ensurePrivateHermesAPICredential is the sole issuer for the per-agent API
// credential. It reads only PartialObjectMetadata through the uncached reader:
// credential bytes are written once and are never read, logged, or serialized.
func (r *AgentIdentityReconciler) ensurePrivateHermesAPICredential(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, namespace string) (bool, error) {
	reason, _ := privateHermesAPIIntent(agent, tenant)
	if reason != "NotReady" {
		return false, nil
	}
	name := privateHermesAPICredentialPrefix + agent.Spec.AgentKey
	metadata := &metav1.PartialObjectMetadata{}
	metadata.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
	reader := client.Reader(r.Client)
	if r.APIReader != nil {
		reader = r.APIReader
	}
	err := reader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, metadata)
	if err == nil {
		return validPrivateHermesAPICredentialMetadata(metadata, agent)
	}
	if !apierrors.IsNotFound(err) {
		return false, fmt.Errorf("read private Hermes API credential metadata: %w", err)
	}
	issuanceID, err := privateHermesAPIRandomToken(32)
	if err != nil {
		return false, fmt.Errorf("generate private Hermes API issuance identity: %w", err)
	}
	apiKey, err := privateHermesAPIRandomToken(32)
	if err != nil {
		return false, fmt.Errorf("generate private Hermes API credential: %w", err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				privateHermesAPIIssuanceIDAnnotation:       issuanceID,
				privateHermesAPIIssuanceRevisionAnnotation: strconv.FormatInt(agent.Generation, 10),
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"api-key": []byte(apiKey)},
	}
	if err := controllerutil.SetControllerReference(agent, secret, r.Scheme); err != nil {
		return false, err
	}
	if err := r.Create(ctx, secret); err != nil {
		return false, fmt.Errorf("issue private Hermes API credential: %w", err)
	}
	return true, nil
}

func validPrivateHermesAPICredentialMetadata(metadata *metav1.PartialObjectMetadata, agent *fabricv1alpha1.AgentIdentity) (bool, error) {
	owner := metav1.GetControllerOf(metadata)
	if owner == nil || owner.APIVersion != fabricv1alpha1.GroupVersion.String() || owner.Kind != "AgentIdentity" || owner.Name != agent.Name || owner.UID != agent.UID ||
		metadata.GetAnnotations()[privateHermesAPIIssuanceIDAnnotation] == "" ||
		metadata.GetAnnotations()[privateHermesAPIIssuanceRevisionAnnotation] != strconv.FormatInt(agent.Generation, 10) {
		return false, fmt.Errorf("private Hermes API credential provenance is invalid")
	}
	return true, nil
}

func privateHermesAPIRandomToken(bytes int) (string, error) {
	value := make([]byte, bytes)
	if _, err := cryptorand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

// ensurePrivateHermesAPIRuntime is the only place that can add the private
// API Server env/port. A SecretKeyRef is applied only after metadata-only
// provenance validation by ensurePrivateHermesAPICredential; secret bytes are
// neither fetched nor copied into the Deployment.
func (r *AgentIdentityReconciler) ensurePrivateHermesAPIRuntime(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, namespace string, credentialReady bool) error {
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: runtimeName(agent.Spec.AgentKey)}, deployment); err != nil {
		return err
	}
	if len(deployment.Spec.Template.Spec.Containers) != 1 || deployment.Spec.Template.Spec.Containers[0].Name != "hermes" {
		return fmt.Errorf("private Hermes API runtime has no canonical hermes container")
	}
	before := deployment.DeepCopy()
	container := &deployment.Spec.Template.Spec.Containers[0]
	container.Env = removePrivateHermesAPIEnv(container.Env)
	container.Ports = removePrivateHermesAPIPort(container.Ports)
	if credentialReady {
		container.Ports = append(container.Ports, corev1.ContainerPort{Name: "responses", ContainerPort: privateHermesAPIPort, Protocol: corev1.ProtocolTCP})
		container.Env = append(container.Env,
			corev1.EnvVar{Name: "API_SERVER_ENABLED", Value: "true"},
			corev1.EnvVar{Name: "API_SERVER_HOST", Value: "0.0.0.0"},
			corev1.EnvVar{Name: "API_SERVER_PORT", Value: strconv.Itoa(privateHermesAPIPort)},
			corev1.EnvVar{Name: "API_SERVER_KEY", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: privateHermesAPICredentialPrefix + agent.Spec.AgentKey}, Key: "api-key"}}},
		)
	}
	if reflect.DeepEqual(before.Spec.Template, deployment.Spec.Template) {
		return nil
	}
	return r.Update(ctx, deployment)
}

func removePrivateHermesAPIEnv(values []corev1.EnvVar) []corev1.EnvVar {
	result := make([]corev1.EnvVar, 0, len(values))
	for _, value := range values {
		if !strings.HasPrefix(value.Name, "API_SERVER_") {
			result = append(result, value)
		}
	}
	return result
}

func removePrivateHermesAPIPort(values []corev1.ContainerPort) []corev1.ContainerPort {
	result := make([]corev1.ContainerPort, 0, len(values))
	for _, value := range values {
		if value.ContainerPort != privateHermesAPIPort {
			result = append(result, value)
		}
	}
	return result
}

// privateHermesAPIWithdrawalComplete proves the API env/port has been removed
// from an available Deployment revision before network reachability or the
// credential can be withdrawn. A status condition alone is never evidence.
func (r *AgentIdentityReconciler) privateHermesAPIWithdrawalComplete(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, namespace string) (bool, error) {
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: runtimeName(agent.Spec.AgentKey)}, deployment); err != nil {
		return false, err
	}
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		return false, fmt.Errorf("private Hermes API withdrawal cannot identify hermes container")
	}
	for _, env := range deployment.Spec.Template.Spec.Containers[0].Env {
		if strings.HasPrefix(env.Name, "API_SERVER_") {
			return false, nil
		}
	}
	for _, port := range deployment.Spec.Template.Spec.Containers[0].Ports {
		if port.ContainerPort == privateHermesAPIPort {
			return false, nil
		}
	}
	replicas := int32(1)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}
	return deployment.Status.ObservedGeneration >= deployment.Generation && deployment.Status.UpdatedReplicas == replicas && deployment.Status.AvailableReplicas == replicas, nil
}

func (r *AgentIdentityReconciler) deletePrivateHermesAPICredential(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, namespace string) error {
	metadata := &metav1.PartialObjectMetadata{}
	metadata.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
	reader := client.Reader(r.Client)
	if r.APIReader != nil {
		reader = r.APIReader
	}
	key := client.ObjectKey{Namespace: namespace, Name: privateHermesAPICredentialPrefix + agent.Spec.AgentKey}
	if err := reader.Get(ctx, key, metadata); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("read private Hermes API credential metadata for withdrawal: %w", err)
	}
	if _, err := validPrivateHermesAPICredentialMetadata(metadata, agent); err != nil {
		return err
	}
	return r.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace}})
}

func (r *AgentIdentityReconciler) privateHermesAPICredentialExists(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, namespace string) (bool, error) {
	metadata := &metav1.PartialObjectMetadata{}
	metadata.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
	reader := client.Reader(r.Client)
	if r.APIReader != nil {
		reader = r.APIReader
	}
	err := reader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: privateHermesAPICredentialPrefix + agent.Spec.AgentKey}, metadata)
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	return err == nil, err
}
