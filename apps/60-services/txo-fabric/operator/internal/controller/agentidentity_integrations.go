package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	integrationCredentialKey          = "credential"
	integrationCredentialMountRoot    = "/run/txo/integrations"
	integrationManifestEnv            = "TXO_INTEGRATIONS_JSON"
	integrationRefreshInterval        = 30 * time.Second
	AnnotationIntegrationPolicyRevision = "fabric.truxonline.io/integration-policy-revision"

	LabelIntegrationCredential = "fabric.truxonline.io/integration-credential"
	LabelIntegrationConnection = "fabric.truxonline.io/integration-connection"
)

type runtimeIntegration struct {
	BindingName     string
	ConnectionName  string
	Protocol        string
	Endpoint        string
	Authentication  string
	Operations      []string
	Scopes          []string
	CredentialName  string
	CredentialPath  string
	VolumeName      string
	Revision        string
}

type integrationResolution struct {
	Effective    []runtimeIntegration
	Status       []fabricv1alpha1.IntegrationAuthorizationStatus
	Revision     string
	HasBindings  bool
	Ready        bool
	Message      string
}

func (r *AgentIdentityReconciler) resolveIntegrationAccess(
	ctx context.Context,
	agent *fabricv1alpha1.AgentIdentity,
	tenant *fabricv1alpha1.TenantBundle,
	namespace string,
) (integrationResolution, error) {
	var bindings fabricv1alpha1.IntegrationBindingList
	if err := r.List(ctx, &bindings); err != nil {
		return integrationResolution{}, err
	}
	sort.Slice(bindings.Items, func(i, j int) bool { return bindings.Items[i].Name < bindings.Items[j].Name })

	result := integrationResolution{Ready: true}
	for i := range bindings.Items {
		binding := &bindings.Items[i]
		if binding.Spec.AgentRef.Name != agent.Name {
			continue
		}
		result.HasBindings = true
		status := fabricv1alpha1.IntegrationAuthorizationStatus{
			BindingName:    binding.Name,
			ConnectionName: binding.Spec.ConnectionRef.Name,
			Operations:     append([]string(nil), binding.Spec.Operations...),
			Scopes:         append([]string(nil), binding.Spec.Scopes...),
		}
		sort.Strings(status.Operations)
		sort.Strings(status.Scopes)

		state := binding.Spec.State
		if state == "" {
			state = fabricv1alpha1.IntegrationBindingStateActive
		}
		if state == fabricv1alpha1.IntegrationBindingStateRevoked {
			status.Phase = "Revoked"
			status.Reason = "BindingRevoked"
			result.Status = append(result.Status, status)
			continue
		}
		if state != fabricv1alpha1.IntegrationBindingStateActive {
			denyIntegration(&result, &status, "BindingInvalid", fmt.Sprintf("unsupported binding state %q", state))
			continue
		}
		if binding.Spec.TenantRef.Name != tenant.Name {
			denyIntegration(&result, &status, "TenantMismatch", "binding tenant does not match AgentIdentity tenant")
			continue
		}
		if strings.TrimSpace(binding.Spec.ConnectionRef.Name) == "" || !validIntegrationStringSet(binding.Spec.Operations, true) || !validIntegrationStringSet(binding.Spec.Scopes, false) {
			denyIntegration(&result, &status, "BindingInvalid", "binding connection, operations or scopes are invalid")
			continue
		}

		var connection fabricv1alpha1.IntegrationConnection
		if err := r.Get(ctx, types.NamespacedName{Name: binding.Spec.ConnectionRef.Name}, &connection); err != nil {
			if apierrors.IsNotFound(err) {
				denyIntegration(&result, &status, "ConnectionNotFound", "referenced IntegrationConnection does not exist")
				continue
			}
			return integrationResolution{}, err
		}
		if connection.Spec.TenantRef.Name != tenant.Name {
			denyIntegration(&result, &status, "TenantMismatch", "connection tenant does not match AgentIdentity tenant")
			continue
		}
		if connection.Spec.Protocol != fabricv1alpha1.IntegrationProtocolHTTP ||
			connection.Spec.Authentication != fabricv1alpha1.IntegrationAuthenticationBearer ||
			!validIntegrationEndpoint(connection.Spec.Endpoint) ||
			strings.TrimSpace(connection.Spec.CredentialRef.Name) == "" {
			denyIntegration(&result, &status, "ConnectionInvalid", "connection protocol, endpoint, authentication or credential reference is invalid")
			continue
		}

		var credential corev1.Secret
		secretKey := types.NamespacedName{Name: connection.Spec.CredentialRef.Name, Namespace: namespace}
		if err := r.Get(ctx, secretKey, &credential); err != nil {
			if apierrors.IsNotFound(err) {
				denyIntegration(&result, &status, "CredentialMissing", "platform-managed integration credential is not available")
				continue
			}
			return integrationResolution{}, err
		}
		if credential.Labels[LabelIntegrationCredential] != "true" ||
			credential.Labels[LabelTenantName] != tenant.Name ||
			credential.Labels[LabelIntegrationConnection] != connection.Name ||
			len(credential.Data[integrationCredentialKey]) == 0 {
			denyIntegration(&result, &status, "CredentialInvalid", "platform-managed integration credential metadata or required key is invalid")
			continue
		}

		revision := integrationRevision(binding, &connection, &credential)
		status.Phase = "Effective"
		status.Reason = "Authorized"
		status.Revision = revision
		result.Status = append(result.Status, status)
		result.Effective = append(result.Effective, runtimeIntegration{
			BindingName:    binding.Name,
			ConnectionName: connection.Name,
			Protocol:       connection.Spec.Protocol,
			Endpoint:       connection.Spec.Endpoint,
			Authentication: connection.Spec.Authentication,
			Operations:     append([]string(nil), status.Operations...),
			Scopes:         append([]string(nil), status.Scopes...),
			CredentialName: connection.Spec.CredentialRef.Name,
			CredentialPath: fmt.Sprintf("%s/%s/%s", integrationCredentialMountRoot, connection.Name, integrationCredentialKey),
			VolumeName:     integrationVolumeName(connection.Name),
			Revision:       revision,
		})
	}
	result.Revision = aggregateIntegrationRevision(result.Effective)
	switch {
	case !result.HasBindings:
		result.Message = "no integration bindings declared"
	case result.Ready:
		result.Message = fmt.Sprintf("%d effective integration binding(s) reconciled", len(result.Effective))
	default:
		result.Message = "one or more integration bindings are denied fail-closed"
	}
	return result, nil
}

func denyIntegration(result *integrationResolution, status *fabricv1alpha1.IntegrationAuthorizationStatus, reason, message string) {
	status.Phase = "Denied"
	status.Reason = reason
	result.Ready = false
	result.Status = append(result.Status, *status)
}

func validIntegrationStringSet(values []string, required bool) bool {
	if required && len(values) == 0 {
		return false
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return false
		}
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func validIntegrationEndpoint(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func integrationRevision(binding *fabricv1alpha1.IntegrationBinding, connection *fabricv1alpha1.IntegrationConnection, credential *corev1.Secret) string {
	input := fmt.Sprintf(
		"binding=%s/%d;connection=%s/%d;credential=%s/%s/%s",
		binding.Name, binding.Generation,
		connection.Name, connection.Generation,
		credential.Name, credential.UID, credential.ResourceVersion,
	)
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:8])
}

func aggregateIntegrationRevision(integrations []runtimeIntegration) string {
	if len(integrations) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(integrations))
	for _, integration := range integrations {
		parts = append(parts, integration.ConnectionName+"="+integration.Revision)
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, ";")))
	return hex.EncodeToString(sum[:8])
}

func integrationVolumeName(connectionName string) string {
	sum := sha256.Sum256([]byte(connectionName))
	return "integration-" + hex.EncodeToString(sum[:4])
}

func (r *AgentIdentityReconciler) requestsForIntegrationBinding(_ context.Context, obj client.Object) []reconcile.Request {
	binding, ok := obj.(*fabricv1alpha1.IntegrationBinding)
	if !ok || strings.TrimSpace(binding.Spec.AgentRef.Name) == "" {
		return nil
	}
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: binding.Spec.AgentRef.Name}}}
}

func (r *AgentIdentityReconciler) requestsForIntegrationConnection(ctx context.Context, obj client.Object) []reconcile.Request {
	connection, ok := obj.(*fabricv1alpha1.IntegrationConnection)
	if !ok {
		return nil
	}
	var bindings fabricv1alpha1.IntegrationBindingList
	if err := r.List(ctx, &bindings); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	seen := map[string]struct{}{}
	for i := range bindings.Items {
		binding := &bindings.Items[i]
		if binding.Spec.ConnectionRef.Name != connection.Name || strings.TrimSpace(binding.Spec.AgentRef.Name) == "" {
			continue
		}
		if _, exists := seen[binding.Spec.AgentRef.Name]; exists {
			continue
		}
		seen[binding.Spec.AgentRef.Name] = struct{}{}
		requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: binding.Spec.AgentRef.Name}})
	}
	return requests
}
