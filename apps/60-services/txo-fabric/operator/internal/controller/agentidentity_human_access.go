package controller

import (
	"context"
	"fmt"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const hermesDashboardPort int32 = 9119

type humanAccessResolution struct {
	Enabled        bool
	Host           string
	PublicURL      string
	FilesRoot      string
	OIDCIssuer     string
	OIDCClientID   string
	OIDCScopes     string
	IngressClass   string
	TLSIssuer      string
}

func resolveHumanAccess(agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) (humanAccessResolution, error) {
	if !agent.Spec.HumanAccess.Enabled {
		return humanAccessResolution{}, nil
	}
	if tenant.Spec.HumanAccess == nil || tenant.Spec.HumanAccess.Web == nil {
		return humanAccessResolution{}, fmt.Errorf("AgentIdentity %q enables human access but TenantBundle %q has no humanAccess.web policy", agent.Name, tenant.Name)
	}
	web := tenant.Spec.HumanAccess.Web
	domain := strings.Trim(strings.TrimSpace(web.DomainSuffix), ".")
	issuer := strings.TrimSpace(web.OIDC.Issuer)
	clientID := strings.TrimSpace(web.OIDC.ClientID)
	tlsIssuer := strings.TrimSpace(web.TLSClusterIssuer)
	if domain == "" || issuer == "" || clientID == "" || tlsIssuer == "" {
		return humanAccessResolution{}, fmt.Errorf("TenantBundle %q humanAccess.web requires domainSuffix, tlsClusterIssuer and OIDC issuer/clientId", tenant.Name)
	}
	if !strings.HasPrefix(issuer, "https://") {
		return humanAccessResolution{}, fmt.Errorf("TenantBundle %q humanAccess OIDC issuer must use https", tenant.Name)
	}
	if agent.Spec.Access.UserRef == "" {
		return humanAccessResolution{}, fmt.Errorf("AgentIdentity %q human access requires access.userRef so the dashboard file surface is bound to an authorized user workspace", agent.Name)
	}
	if tenant.Spec.Workspace == nil {
		return humanAccessResolution{}, fmt.Errorf("AgentIdentity %q human access requires TenantBundle %q workspace capability", agent.Name, tenant.Name)
	}
	var user *fabricv1alpha1.NamedWorkspaceScopeSpec
	for i := range tenant.Spec.Workspace.Users {
		if tenant.Spec.Workspace.Users[i].Name == agent.Spec.Access.UserRef {
			user = &tenant.Spec.Workspace.Users[i]
			break
		}
	}
	if user == nil || !user.Collaborative {
		return humanAccessResolution{}, fmt.Errorf("AgentIdentity %q human access requires collaborative workspace user %q in tenant %q", agent.Name, agent.Spec.Access.UserRef, tenant.Name)
	}

	host := fmt.Sprintf("%s-%s.%s", agent.Spec.AgentKey, tenant.Name, domain)
	if len(host) > 253 {
		return humanAccessResolution{}, fmt.Errorf("derived human access host %q exceeds DNS length", host)
	}
	scopes := strings.TrimSpace(web.OIDC.Scopes)
	if scopes == "" {
		scopes = "openid profile email"
	}
	ingressClass := strings.TrimSpace(web.IngressClassName)
	if ingressClass == "" {
		ingressClass = "traefik"
	}
	filesRoot := workspaceMountPath(workspaceScope{
		Domain: "shared",
		Scope:  "user",
		Key:    agent.Spec.Access.UserRef,
		Mode:   "collaborative",
	})
	return humanAccessResolution{
		Enabled:      true,
		Host:         host,
		PublicURL:    "https://" + host,
		FilesRoot:    filesRoot,
		OIDCIssuer:   issuer,
		OIDCClientID: clientID,
		OIDCScopes:   scopes,
		IngressClass: ingressClass,
		TLSIssuer:    tlsIssuer,
	}, nil
}

func humanAccessResourceName(agentKey string) string {
	return runtimeName(agentKey) + "-dashboard"
}

func (r *AgentIdentityReconciler) ensureHumanAccessResources(
	ctx context.Context,
	agent *fabricv1alpha1.AgentIdentity,
	tenant *fabricv1alpha1.TenantBundle,
	namespace string,
	access humanAccessResolution,
) error {
	name := humanAccessResourceName(agent.Spec.AgentKey)
	if !access.Enabled {
		for _, obj := range []client.Object{
			&networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}},
			&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}},
			&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: name + "-ingress", Namespace: namespace}},
		} {
			if err := r.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
				return err
			}
		}
		return nil
	}

	labels := agentLabels(agent, tenant)
	labels["app.kubernetes.io/component"] = "human-entry"

	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		service.Labels = mergeStringMap(service.Labels, labels)
		service.Spec.Type = corev1.ServiceTypeClusterIP
		service.Spec.Selector = map[string]string{LabelName: "hermes-agent", LabelInstance: agent.Spec.AgentKey}
		service.Spec.Ports = []corev1.ServicePort{{
			Name:       "dashboard",
			Protocol:   corev1.ProtocolTCP,
			Port:       hermesDashboardPort,
			TargetPort: intstr.FromInt32(hermesDashboardPort),
		}}
		return controllerutil.SetControllerReference(agent, service, r.Scheme)
	}); err != nil {
		return err
	}

	pathType := networkingv1.PathTypePrefix
	ingressClass := access.IngressClass
	ingress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		ingress.Labels = mergeStringMap(ingress.Labels, labels)
		ingress.Annotations = mergeStringMap(ingress.Annotations, map[string]string{
			"cert-manager.io/cluster-issuer":                      access.TLSIssuer,
			"traefik.ingress.kubernetes.io/router.entrypoints":   "web, websecure",
			"traefik.ingress.kubernetes.io/router.middlewares":   "traefik-redirect-https@kubernetescrd",
		})
		ingress.Spec = networkingv1.IngressSpec{
			IngressClassName: &ingressClass,
			Rules: []networkingv1.IngressRule{{
				Host: access.Host,
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{{
						Path:     "/",
						PathType: &pathType,
						Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
							Name: name,
							Port: networkingv1.ServiceBackendPort{Number: hermesDashboardPort},
						}},
					}},
				}},
			}},
			TLS: []networkingv1.IngressTLS{{
				Hosts:      []string{access.Host},
				SecretName: name + "-tls",
			}},
		}
		return controllerutil.SetControllerReference(agent, ingress, r.Scheme)
	}); err != nil {
		return err
	}

	policy := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: name + "-ingress", Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, policy, func() error {
		policy.Labels = mergeStringMap(policy.Labels, labels)
		policy.Spec = networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{LabelName: "hermes-agent", LabelInstance: agent.Spec.AgentKey}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "traefik"}},
				}},
				Ports: []networkingv1.NetworkPolicyPort{{
					Protocol: protocolPtr(corev1.ProtocolTCP),
					Port:     intOrStringPtr(int(hermesDashboardPort)),
				}},
			}},
		}
		return controllerutil.SetControllerReference(agent, policy, r.Scheme)
	}); err != nil {
		return err
	}
	return nil
}
