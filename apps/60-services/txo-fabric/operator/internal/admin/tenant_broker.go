package admin

import (
	"context"
	"fmt"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultBrokerProfileName = "cliproxyapi-standard"
	brokerServiceName        = "txo-ai-credential-broker"
	brokerRuntimeSecretName  = "txo-ai-credential-broker-runtime"
)

type TenantBrokerTarget struct {
	TenantName         string
	Namespace          string
	BaseURL            string
	ManagementCredential string
}

func ResolveTenantBroker(ctx context.Context, c client.Client, tenantName string) (TenantBrokerTarget, error) {
	tenantName = strings.TrimSpace(tenantName)
	if tenantName == "" {
		return TenantBrokerTarget{}, fmt.Errorf("tenant name is required")
	}

	var tenant fabricv1alpha1.TenantBundle
	if err := c.Get(ctx, types.NamespacedName{Name: tenantName}, &tenant); err != nil {
		return TenantBrokerTarget{}, fmt.Errorf("get TenantBundle %q: %w", tenantName, err)
	}
	if tenant.Spec.AICredentialBroker == nil {
		return TenantBrokerTarget{}, fmt.Errorf("TenantBundle %q has no aiCredentialBroker", tenantName)
	}

	profileName := strings.TrimSpace(tenant.Spec.AICredentialBroker.ProfileRef)
	if profileName == "" {
		profileName = defaultBrokerProfileName
	}
	var profile fabricv1alpha1.AICredentialBrokerProfile
	if err := c.Get(ctx, types.NamespacedName{Name: profileName}, &profile); err != nil {
		return TenantBrokerTarget{}, fmt.Errorf("get AICredentialBrokerProfile %q: %w", profileName, err)
	}
	if profile.Spec.Topology != "" && profile.Spec.Topology != "TenantScoped" {
		return TenantBrokerTarget{}, fmt.Errorf("AICredentialBrokerProfile %q topology %q is not tenant-scoped", profileName, profile.Spec.Topology)
	}
	if profile.Spec.Implementation != "" && profile.Spec.Implementation != "CLIProxyAPI" {
		return TenantBrokerTarget{}, fmt.Errorf("AICredentialBrokerProfile %q implementation %q is not CLIProxyAPI", profileName, profile.Spec.Implementation)
	}

	port := profile.Spec.APIPort
	if port <= 0 {
		port = 8317
	}
	namespace := "tenant-" + tenantName
	var secret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: brokerRuntimeSecretName}, &secret); err != nil {
		return TenantBrokerTarget{}, fmt.Errorf("get tenant CPA runtime Secret: %w", err)
	}
	managementCredential := strings.TrimSpace(string(secret.Data["management-password"]))
	if managementCredential == "" {
		return TenantBrokerTarget{}, fmt.Errorf("tenant CPA runtime Secret has no management-password")
	}

	return TenantBrokerTarget{
		TenantName:         tenantName,
		Namespace:          namespace,
		BaseURL:            fmt.Sprintf("http://%s.%s.svc:%d", brokerServiceName, namespace, port),
		ManagementCredential: managementCredential,
	}, nil
}
