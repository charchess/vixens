package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	tenantAIGatewayName                  = "txo-ai-gateway"
	tenantAIGatewayConfigMapName         = "txo-ai-gateway-config"
	tenantAIGatewayNetworkPolicy         = "txo-ai-gateway"
	tenantAIGatewayDefaultPort      int32 = 4000
	tenantAIProviderSecretNamespace      = "txo-fabric-system"
	tenantAIProviderSecretPrefix         = "txo-ai-provider-"
	tenantAIOpenRouterSecretKey          = "OPENROUTER_API_KEY"
	tenantAIEmbeddingProviderModel       = "openrouter/baai/bge-m3"
)

type aiGatewayBackendState struct {
	CPAEnabled                 bool
	CPAPort                    int32
	OpenRouterEnabled          bool
	OpenRouterSecretRevision   string
}

func tenantAIProviderSecretName(tenantName string) string {
	return tenantAIProviderSecretPrefix + tenantName
}

func aiGatewayPort(profile *fabricv1alpha1.AIGatewayProfile) int32 {
	if profile.Spec.APIPort > 0 {
		return profile.Spec.APIPort
	}
	return tenantAIGatewayDefaultPort
}

func aiGatewayWorkloadLabels(bundle *fabricv1alpha1.TenantBundle) map[string]string {
	labels := aiGatewayRuntimeLabels(bundle)
	labels[LabelName] = tenantAIGatewayName
	labels["app.kubernetes.io/component"] = "tenant-ai-gateway"
	return labels
}

func (r *TenantBundleReconciler) resolveAIGatewayBackends(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (aiGatewayBackendState, string, error) {
	namespace := tenantNamespace(bundle.Name)
	var service corev1.Service
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: tenantAICredentialBrokerName}, &service); err != nil {
		if apierrors.IsNotFound(err) {
			return aiGatewayBackendState{}, "waiting for tenant AI credential broker Service", nil
		}
		return aiGatewayBackendState{}, "", err
	}
	if len(service.Spec.Ports) == 0 || service.Spec.Ports[0].Port <= 0 {
		return aiGatewayBackendState{}, "tenant AI credential broker Service has no usable port", nil
	}

	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: tenantAICredentialBrokerSecretName}, &secret); err != nil {
		if apierrors.IsNotFound(err) {
			return aiGatewayBackendState{}, "waiting for tenant AI credential broker runtime Secret", nil
		}
		return aiGatewayBackendState{}, "", err
	}
	if len(secret.Data["bootstrap-api-key"]) == 0 {
		return aiGatewayBackendState{}, "tenant AI credential broker runtime Secret has no bootstrap API key", nil
	}

	backends := aiGatewayBackendState{CPAEnabled: true, CPAPort: service.Spec.Ports[0].Port}
	var providerSecret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: tenantAIProviderSecretNamespace, Name: tenantAIProviderSecretName(bundle.Name)}, &providerSecret); err != nil {
		if !apierrors.IsNotFound(err) {
			return aiGatewayBackendState{}, "", err
		}
	} else if len(providerSecret.Data[tenantAIOpenRouterSecretKey]) > 0 {
		backends.OpenRouterEnabled = true
		backends.OpenRouterSecretRevision = strings.TrimSpace(providerSecret.ResourceVersion)
		if backends.OpenRouterSecretRevision == "" {
			backends.OpenRouterSecretRevision = strings.TrimSpace(string(providerSecret.UID))
		}
		if backends.OpenRouterSecretRevision == "" {
			// Real API objects always have a resourceVersion. This deterministic
			// fallback keeps fake-client/unit contracts explicit without hashing
			// provider secret material into metadata.
			backends.OpenRouterSecretRevision = "present"
		}
	}

	return backends, "", nil
}

func renderTenantLiteLLMConfig(backends aiGatewayBackendState) string {
	var builder strings.Builder
	builder.WriteString("model_list:\n")
	if backends.CPAEnabled {
		builder.WriteString(fmt.Sprintf(`  - model_name: txo-agent
    litellm_params:
      model: openai/gpt-5.6-sol
      api_base: http://%s:%d/v1
      api_key: os.environ/CPA_API_KEY
`, tenantAICredentialBrokerName, backends.CPAPort))
	}
	if backends.OpenRouterEnabled {
		builder.WriteString(`  - model_name: txo-embedding
    litellm_params:
      model: openrouter/baai/bge-m3
      api_key: os.environ/OPENROUTER_API_KEY
`)
	}
	if !backends.CPAEnabled && !backends.OpenRouterEnabled {
		builder.WriteString("  []\n")
	}
	builder.WriteString(`general_settings:
  master_key: os.environ/LITELLM_MASTER_KEY
`)
	return builder.String()
}

func aiGatewayConfigHash(config string) string {
	sum := sha256.Sum256([]byte(config))
	return hex.EncodeToString(sum[:8])
}

func (r *TenantBundleReconciler) ensureTenantAIGatewayConfig(
	ctx context.Context,
	bundle *fabricv1alpha1.TenantBundle,
	config string,
) (*corev1.ConfigMap, string, error) {
	namespace := tenantNamespace(bundle.Name)
	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayConfigMapName, Namespace: namespace}}
	hash := aiGatewayConfigHash(config)
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		configMap.Labels = mergeStringMap(configMap.Labels, aiGatewayWorkloadLabels(bundle))
		if err := controllerutil.SetControllerReference(bundle, configMap, r.Scheme); err != nil {
			return err
		}
		configMap.Data = map[string]string{"config.yaml": config}
		if configMap.Annotations == nil {
			configMap.Annotations = map[string]string{}
		}
		configMap.Annotations["fabric.truxonline.io/config-hash"] = hash
		return nil
	})
	return configMap, hash, err
}

func (r *TenantBundleReconciler) ensureTenantAIGatewayService(
	ctx context.Context,
	bundle *fabricv1alpha1.TenantBundle,
	profile *fabricv1alpha1.AIGatewayProfile,
) error {
	namespace := tenantNamespace(bundle.Name)
	labels := aiGatewayWorkloadLabels(bundle)
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayName, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		service.Labels = mergeStringMap(service.Labels, labels)
		if err := controllerutil.SetControllerReference(bundle, service, r.Scheme); err != nil {
			return err
		}
		service.Spec.Type = corev1.ServiceTypeClusterIP
		service.Spec.Selector = labels
		service.Spec.Ports = []corev1.ServicePort{{
			Name:       "http",
			Port:       aiGatewayPort(profile),
			TargetPort: intstr.FromString("http"),
			Protocol:   corev1.ProtocolTCP,
		}}
		return nil
	})
	return err
}

func (r *TenantBundleReconciler) ensureTenantAIGatewayNetworkPolicy(
	ctx context.Context,
	bundle *fabricv1alpha1.TenantBundle,
	profile *fabricv1alpha1.AIGatewayProfile,
	postgresqlProfile *fabricv1alpha1.PostgreSQLProfile,
	backends aiGatewayBackendState,
) error {
	namespace := tenantNamespace(bundle.Name)
	labels := aiGatewayWorkloadLabels(bundle)
	policy := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayNetworkPolicy, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, policy, func() error {
		policy.Labels = mergeStringMap(policy.Labels, labels)
		if err := controllerutil.SetControllerReference(bundle, policy, r.Scheme); err != nil {
			return err
		}
		policy.Spec.PodSelector = metav1.LabelSelector{MatchLabels: labels}
		policy.Spec.PolicyTypes = []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress}
		policy.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{{
			From: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{LabelManaged: "true"}},
				},
				{
					PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
						LabelPartOf:     "txo-fabric",
						LabelName:       "hermes-agent",
						LabelTenantName: bundle.Name,
					}},
				},
				{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
						"kubernetes.io/metadata.name": "txo-fabric-system",
					}},
					PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
						LabelName: "txo-fabric-operator",
					}},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{{
				Protocol: protocolPtr(corev1.ProtocolTCP),
				Port:     intOrStringPtr(int(aiGatewayPort(profile))),
			}},
		}}
		policy.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{
			{
				To: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "kube-system"}},
				}},
				Ports: []networkingv1.NetworkPolicyPort{
					{Protocol: protocolPtr(corev1.ProtocolUDP), Port: intOrStringPtr(53)},
					{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(53)},
				},
			},
			{
				To: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": postgresqlProfile.Spec.Shared.ClusterRef.Namespace}},
					PodSelector:       &metav1.LabelSelector{MatchLabels: map[string]string{"cnpg.io/cluster": postgresqlProfile.Spec.Shared.ClusterRef.Name}},
				}},
				Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(5432)}},
			},
		}
		if backends.CPAEnabled {
			policy.Spec.Egress = append(policy.Spec.Egress, networkingv1.NetworkPolicyEgressRule{
				To: []networkingv1.NetworkPolicyPeer{{
					PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
						LabelManaged:    "true",
						LabelTenantName: bundle.Name,
						"app.kubernetes.io/component": "tenant-ai-credential-broker",
					}},
				}},
				Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(int(backends.CPAPort))}},
			})
		}
		if backends.OpenRouterEnabled {
			// Provider egress exists only on the tenant gateway. Tenant workloads
			// never receive the provider credential or a direct-provider route.
			policy.Spec.Egress = append(policy.Spec.Egress, networkingv1.NetworkPolicyEgressRule{
				Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(443)}},
			})
		}
		return nil
	})
	return err
}

func (r *TenantBundleReconciler) ensureTenantAIGatewayDeployment(
	ctx context.Context,
	bundle *fabricv1alpha1.TenantBundle,
	profile *fabricv1alpha1.AIGatewayProfile,
	backends aiGatewayBackendState,
	configHash string,
) (*appsv1.Deployment, error) {
	namespace := tenantNamespace(bundle.Name)
	labels := aiGatewayWorkloadLabels(bundle)
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: tenantAIGatewayName, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Labels = mergeStringMap(deployment.Labels, labels)
		if err := controllerutil.SetControllerReference(bundle, deployment, r.Scheme); err != nil {
			return err
		}
		replicas := int32(1)
		runAsNonRoot := true
		runAsUser := int64(65534)
		runAsGroup := int64(65534)
		allowPrivilegeEscalation := false
		deployment.Spec.Replicas = &replicas
		deployment.Spec.Strategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
		deployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
		deployment.Spec.Template.ObjectMeta.Labels = labels
		if deployment.Spec.Template.ObjectMeta.Annotations == nil {
			deployment.Spec.Template.ObjectMeta.Annotations = map[string]string{}
		}
		deployment.Spec.Template.ObjectMeta.Annotations["fabric.truxonline.io/config-hash"] = configHash
		if backends.OpenRouterEnabled {
			deployment.Spec.Template.ObjectMeta.Annotations["fabric.truxonline.io/openrouter-secret-revision"] = backends.OpenRouterSecretRevision
		} else {
			delete(deployment.Spec.Template.ObjectMeta.Annotations, "fabric.truxonline.io/openrouter-secret-revision")
		}
		if cpuRequest, ok := profile.Spec.Resources.Requests[corev1.ResourceCPU]; ok && !cpuRequest.IsZero() {
			// LiteLLM cold start is CPU-bound. The V-scout label enables VPA with
			// RequestsAndLimits control, so without an explicit floor VPA may shrink
			// the serving container below the profile's proven startup baseline and
			// make the startup probe kill it before Uvicorn begins listening.
			deployment.Spec.Template.ObjectMeta.Annotations["vixens.io/vpa.min-cpu"] = cpuRequest.String()
		} else {
			delete(deployment.Spec.Template.ObjectMeta.Annotations, "vixens.io/vpa.min-cpu")
		}
		deployment.Spec.Template.Spec.AutomountServiceAccountToken = boolPtr(false)
		deployment.Spec.Template.Spec.PriorityClassName = profile.Spec.PriorityClassName
		deployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
			RunAsNonRoot: &runAsNonRoot,
			RunAsUser:    &runAsUser,
			RunAsGroup:   &runAsGroup,
			SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
		}

		env := []corev1.EnvVar{{Name: "DISABLE_SCHEMA_UPDATE", Value: "true"}}
		if backends.CPAEnabled {
			env = append(env, corev1.EnvVar{
				Name: "CPA_API_KEY",
				ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: tenantAICredentialBrokerSecretName},
					Key:                  "bootstrap-api-key",
				}},
			})
		}

		deployment.Spec.Template.Spec.Containers = []corev1.Container{{
			Name:            "litellm",
			Image:           profile.Spec.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/bin/sh", "-ec"},
			Args: []string{`
export DATABASE_URL="$(python -c 'import os, urllib.parse; u=urllib.parse.quote(os.environ["DB_USERNAME"], safe=""); p=urllib.parse.quote(os.environ["DB_PASSWORD"], safe=""); h=os.environ["DB_HOST"]; port=os.environ.get("DB_PORT", "5432"); d=os.environ["DB_NAME"]; print(f"postgresql://{u}:{p}@{h}:{port}/{d}")')"
exec litellm --config /app/config.yaml --port ` + fmt.Sprintf("%d", aiGatewayPort(profile))},
			Env:     env,
			EnvFrom: []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: tenantAIGatewayRuntimeSecretName}}}},
			Ports:   []corev1.ContainerPort{{Name: "http", ContainerPort: aiGatewayPort(profile), Protocol: corev1.ProtocolTCP}},
			VolumeMounts: []corev1.VolumeMount{{
				Name: "config", MountPath: "/app/config.yaml", SubPath: "config.yaml", ReadOnly: true,
			}},
			StartupProbe: &corev1.Probe{
				ProbeHandler:     corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/health/liveliness", Port: intstr.FromString("http")}},
				PeriodSeconds:    5,
				FailureThreshold: 60,
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler:     corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/health/readiness", Port: intstr.FromString("http")}},
				PeriodSeconds:    10,
				TimeoutSeconds:   3,
				FailureThreshold: 6,
			},
			LivenessProbe: &corev1.Probe{
				ProbeHandler:     corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/health/liveliness", Port: intstr.FromString("http")}},
				PeriodSeconds:    20,
				TimeoutSeconds:   3,
				FailureThreshold: 3,
			},
			Resources: profile.Spec.Resources,
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: &allowPrivilegeEscalation,
				Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			},
		}}
		deployment.Spec.Template.Spec.Volumes = []corev1.Volume{{
			Name: "config",
			VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: tenantAIGatewayConfigMapName},
			}},
		}}
		if profile.Spec.SizingLabel != "" {
			deployment.Spec.Template.ObjectMeta.Labels["vixens.io/sizing.litellm"] = profile.Spec.SizingLabel
		}
		return nil
	})
	return deployment, err
}
