package controller

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultAICredentialBrokerProfileName = "cliproxyapi-standard"
	tenantAICredentialBrokerName         = "txo-ai-credential-broker"
	tenantAICredentialBrokerSecretName   = "txo-ai-credential-broker-runtime"
	tenantAICredentialBrokerAuthPVCName  = "txo-ai-credential-broker-auth"
)

type aiCredentialBrokerResult struct {
	Ready   bool
	Status  *fabricv1alpha1.ComponentStatus
	Reason  string
	Message string
}

func (r *TenantBundleReconciler) reconcileAICredentialBroker(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (aiCredentialBrokerResult, error) {
	request := bundle.Spec.AICredentialBroker
	if request == nil {
		return aiCredentialBrokerResult{Ready: true, Reason: "NotRequested", Message: "tenant does not request an AI credential broker"}, nil
	}

	profileName := strings.TrimSpace(request.ProfileRef)
	if profileName == "" {
		profileName = defaultAICredentialBrokerProfileName
	}
	var profile fabricv1alpha1.AICredentialBrokerProfile
	if err := r.Get(ctx, types.NamespacedName{Name: profileName}, &profile); err != nil {
		if apierrors.IsNotFound(err) {
			status := &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: fmt.Sprintf("AICredentialBrokerProfile %q does not exist", profileName)}
			return aiCredentialBrokerResult{Status: status, Reason: "ProfileNotFound", Message: status.Message}, nil
		}
		return aiCredentialBrokerResult{}, err
	}
	if profile.Spec.Topology != "" && profile.Spec.Topology != "TenantScoped" {
		status := &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: fmt.Sprintf("AICredentialBrokerProfile %q topology %q is not supported", profileName, profile.Spec.Topology)}
		return aiCredentialBrokerResult{Status: status, Reason: "UnsupportedTopology", Message: status.Message}, nil
	}
	if profile.Spec.Implementation != "" && profile.Spec.Implementation != "CLIProxyAPI" {
		status := &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: fmt.Sprintf("AICredentialBrokerProfile %q implementation %q is not supported", profileName, profile.Spec.Implementation)}
		return aiCredentialBrokerResult{Status: status, Reason: "UnsupportedImplementation", Message: status.Message}, nil
	}
	if strings.TrimSpace(profile.Spec.AuthStorage.StorageClassName) == "" {
		status := &fabricv1alpha1.ComponentStatus{Phase: "Blocked", Message: fmt.Sprintf("AICredentialBrokerProfile %q must declare authStorage.storageClassName", profileName)}
		return aiCredentialBrokerResult{Status: status, Reason: "StorageClassRequired", Message: status.Message}, nil
	}

	namespace := tenantNamespace(bundle.Name)
	secret, err := r.ensureTenantAICredentialBrokerSecret(ctx, bundle, &profile, namespace)
	if err != nil {
		return aiCredentialBrokerResult{}, err
	}
	if err := r.ensureTenantAICredentialBrokerPVC(ctx, bundle, &profile, namespace); err != nil {
		return aiCredentialBrokerResult{}, err
	}
	if err := r.ensureTenantAICredentialBrokerService(ctx, bundle, &profile, namespace); err != nil {
		return aiCredentialBrokerResult{}, err
	}
	if err := r.ensureTenantAICredentialBrokerNetworkPolicy(ctx, bundle, &profile, namespace); err != nil {
		return aiCredentialBrokerResult{}, err
	}
	if err := r.ensureTenantAICredentialBrokerDeployment(ctx, bundle, &profile, namespace, secret); err != nil {
		return aiCredentialBrokerResult{}, err
	}

	var deployment appsv1.Deployment
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tenantAICredentialBrokerName}, &deployment); err != nil {
		return aiCredentialBrokerResult{}, err
	}
	endpoint := fmt.Sprintf("http://%s.%s.svc:%d", tenantAICredentialBrokerName, namespace, aiCredentialBrokerPort(&profile))
	if deployment.Status.AvailableReplicas > 0 && deployment.Status.ObservedGeneration == deployment.Generation {
		status := &fabricv1alpha1.ComponentStatus{Phase: "Ready", Endpoint: endpoint}
		return aiCredentialBrokerResult{Ready: true, Status: status, Reason: "DeploymentAvailable", Message: "tenant CLIProxyAPI credential broker is available"}, nil
	}
	status := &fabricv1alpha1.ComponentStatus{Phase: "Provisioning", Endpoint: endpoint, Message: "waiting for tenant CLIProxyAPI Deployment to become available"}
	return aiCredentialBrokerResult{Status: status, Reason: "DeploymentProgressing", Message: status.Message}, nil
}

func aiCredentialBrokerPort(profile *fabricv1alpha1.AICredentialBrokerProfile) int32 {
	if profile.Spec.APIPort > 0 {
		return profile.Spec.APIPort
	}
	return 8317
}

func aiCredentialBrokerLabels(bundle *fabricv1alpha1.TenantBundle) map[string]string {
	labels := tenantLabels(bundle)
	labels[LabelName] = tenantAICredentialBrokerName
	labels[LabelInstance] = strings.ToLower(bundle.Spec.TenantID)
	labels["app.kubernetes.io/component"] = "tenant-ai-credential-broker"
	return labels
}

func randomCredentialBrokerToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate tenant AI credential broker credential: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func renderCLIProxyAPIBrokerConfig(profile *fabricv1alpha1.AICredentialBrokerProfile, clientKey string) string {
	retention := profile.Spec.UsageQueueRetentionSeconds
	if retention <= 0 {
		retention = 900
	}
	return fmt.Sprintf(`config-version: 8
server:
  host: "0.0.0.0"
  port: %d
  discovery:
    enabled: false
management:
  allow-remote: true
  secret-key: ""
  disable-control-panel: true
  disable-auto-update-panel: true
access:
  api-keys:
    - %q
routing:
  strategy: "round-robin"
  session-affinity: false
  force-model-prefix: true
  retry:
    request-retry: 3
    max-retry-credentials: 0
    max-retry-interval: 30
oauth:
  auth-dir: "/data/auth"
observability:
  logs:
    debug: false
    logging-to-file: false
    request-log: false
  usage:
    usage-statistics-enabled: true
    redis-usage-queue-retention-seconds: %d
plugins:
  enabled: false
`, aiCredentialBrokerPort(profile), clientKey, retention)
}

func (r *TenantBundleReconciler) ensureTenantAICredentialBrokerSecret(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.AICredentialBrokerProfile, namespace string) (*corev1.Secret, error) {
	key := types.NamespacedName{Namespace: namespace, Name: tenantAICredentialBrokerSecretName}
	var secret corev1.Secret
	if err := r.Get(ctx, key, &secret); err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	if secret.Name == "" {
		secret = corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: tenantAICredentialBrokerSecretName, Namespace: namespace}, Type: corev1.SecretTypeOpaque}
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, &secret, func() error {
		secret.Labels = mergeStringMap(secret.Labels, aiCredentialBrokerLabels(bundle))
		if err := controllerutil.SetControllerReference(bundle, &secret, r.Scheme); err != nil {
			return err
		}
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		if len(secret.Data["bootstrap-api-key"]) == 0 {
			value, err := randomCredentialBrokerToken()
			if err != nil {
				return err
			}
			secret.Data["bootstrap-api-key"] = []byte(value)
		}
		if len(secret.Data["management-password"]) == 0 {
			value, err := randomCredentialBrokerToken()
			if err != nil {
				return err
			}
			secret.Data["management-password"] = []byte(value)
		}
		secret.Data["config.yaml"] = []byte(renderCLIProxyAPIBrokerConfig(profile, string(secret.Data["bootstrap-api-key"])))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &secret, nil
}

func (r *TenantBundleReconciler) ensureTenantAICredentialBrokerPVC(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.AICredentialBrokerProfile, namespace string) error {
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: tenantAICredentialBrokerAuthPVCName, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, pvc, func() error {
		pvc.Labels = mergeStringMap(pvc.Labels, aiCredentialBrokerLabels(bundle))
		pvc.Labels[LabelStorageRetention] = StorageRetentionDelete
		if err := controllerutil.SetControllerReference(bundle, pvc, r.Scheme); err != nil {
			return err
		}
		if pvc.CreationTimestamp.IsZero() {
			pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
			pvc.Spec.StorageClassName = &profile.Spec.AuthStorage.StorageClassName
			pvc.Spec.Resources.Requests = corev1.ResourceList{corev1.ResourceStorage: profile.Spec.AuthStorage.Size}
		}
		return nil
	})
	return err
}

func (r *TenantBundleReconciler) ensureTenantAICredentialBrokerService(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.AICredentialBrokerProfile, namespace string) error {
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: tenantAICredentialBrokerName, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Labels = mergeStringMap(svc.Labels, aiCredentialBrokerLabels(bundle))
		if err := controllerutil.SetControllerReference(bundle, svc, r.Scheme); err != nil {
			return err
		}
		svc.Spec.Type = corev1.ServiceTypeClusterIP
		svc.Spec.Selector = aiCredentialBrokerLabels(bundle)
		svc.Spec.Ports = []corev1.ServicePort{{Name: "http", Port: aiCredentialBrokerPort(profile), TargetPort: intstr.FromInt32(aiCredentialBrokerPort(profile)), Protocol: corev1.ProtocolTCP}}
		return nil
	})
	return err
}

func (r *TenantBundleReconciler) ensureTenantAICredentialBrokerDeployment(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.AICredentialBrokerProfile, namespace string, secret *corev1.Secret) error {
	labels := aiCredentialBrokerLabels(bundle)
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: tenantAICredentialBrokerName, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Labels = mergeStringMap(deployment.Labels, labels)
		if err := controllerutil.SetControllerReference(bundle, deployment, r.Scheme); err != nil {
			return err
		}
		replicas := int32(1)
		runAsNonRoot := true
		runAsUser := int64(65532)
		runAsGroup := int64(65532)
		fsGroup := int64(65532)
		deployment.Spec.Replicas = &replicas
		deployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
		deployment.Spec.Template.ObjectMeta.Labels = labels
		deployment.Spec.Template.Spec.PriorityClassName = profile.Spec.PriorityClassName
		deployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{RunAsNonRoot: &runAsNonRoot, RunAsUser: &runAsUser, RunAsGroup: &runAsGroup, FSGroup: &fsGroup}
		deployment.Spec.Template.Spec.Containers = []corev1.Container{{
			Name:            "cliproxyapi",
			Image:           profile.Spec.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Ports:           []corev1.ContainerPort{{Name: "http", ContainerPort: aiCredentialBrokerPort(profile), Protocol: corev1.ProtocolTCP}},
			Env: []corev1.EnvVar{{
				Name: "MANAGEMENT_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: secret.Name}, Key: "management-password"}},
			}},
			Resources: profile.Spec.Resources,
			VolumeMounts: []corev1.VolumeMount{
				{Name: "runtime-config", MountPath: "/CLIProxyAPI/config.yaml", SubPath: "config.yaml", ReadOnly: true},
				{Name: "oauth-state", MountPath: "/data/auth"},
			},
			ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromString("http")}}, InitialDelaySeconds: 2, PeriodSeconds: 5, FailureThreshold: 12},
			LivenessProbe:  &corev1.Probe{ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromString("http")}}, InitialDelaySeconds: 10, PeriodSeconds: 10, FailureThreshold: 6},
			SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: boolPtr(false), ReadOnlyRootFilesystem: boolPtr(false), Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}},
		}}
		deployment.Spec.Template.Spec.Volumes = []corev1.Volume{
			{Name: "runtime-config", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: secret.Name}}},
			{Name: "oauth-state", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: tenantAICredentialBrokerAuthPVCName}}},
		}
		if profile.Spec.SizingLabel != "" {
			deployment.Spec.Template.ObjectMeta.Labels["vixens.io/sizing"] = profile.Spec.SizingLabel
		}
		return nil
	})
	return err
}

func (r *TenantBundleReconciler) ensureTenantAICredentialBrokerNetworkPolicy(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.AICredentialBrokerProfile, namespace string) error {
	labels := aiCredentialBrokerLabels(bundle)
	policy := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: tenantAICredentialBrokerName, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, policy, func() error {
		policy.Labels = mergeStringMap(policy.Labels, labels)
		if err := controllerutil.SetControllerReference(bundle, policy, r.Scheme); err != nil {
			return err
		}
		policy.Spec.PodSelector = metav1.LabelSelector{MatchLabels: labels}
		policy.Spec.PolicyTypes = []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress}
		policy.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{{
			From: []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{LabelManaged: "true"}}}},
			Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(int(aiCredentialBrokerPort(profile)))}},
		}}
		policy.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{
			{
				To: []networkingv1.NetworkPolicyPeer{{NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "kube-system"}}}},
				Ports: []networkingv1.NetworkPolicyPort{
					{Protocol: protocolPtr(corev1.ProtocolUDP), Port: intOrStringPtr(53)},
					{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(53)},
				},
			},
			{To: []networkingv1.NetworkPolicyPeer{{}}, Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(443)}}},
		}
		return nil
	})
	return err
}

func (r *TenantBundleReconciler) cleanupAICredentialBroker(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (bool, error) {
	namespace := tenantNamespace(bundle.Name)
	pending := false
	objects := []client.Object{
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: tenantAICredentialBrokerName, Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: tenantAICredentialBrokerName, Namespace: namespace}},
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: tenantAICredentialBrokerName, Namespace: namespace}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: tenantAICredentialBrokerSecretName, Namespace: namespace}},
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: tenantAICredentialBrokerAuthPVCName, Namespace: namespace}},
	}
	for _, obj := range objects {
		if err := r.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return false, err
		}
		if obj.GetLabels()[LabelManaged] != "true" || obj.GetLabels()[LabelTenantName] != bundle.Name {
			return false, fmt.Errorf("%s %s/%s exists but is not owned by tenant %q", obj.GetObjectKind().GroupVersionKind().Kind, namespace, obj.GetName(), bundle.Name)
		}
		if obj.GetDeletionTimestamp().IsZero() {
			if err := r.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
				return false, err
			}
		}
		pending = true
	}
	return pending, nil
}
