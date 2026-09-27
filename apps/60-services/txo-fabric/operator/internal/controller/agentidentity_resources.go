package controller

import (
	"context"
	"fmt"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *AgentIdentityReconciler) ensurePVC(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.AgentRuntimeProfile, namespace string) error {
	name := runtimePVCName(agent.Spec.AgentKey)
	var pvc corev1.PersistentVolumeClaim
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &pvc)
	if apierrors.IsNotFound(err) {
		pvc = corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: agentLabels(agent, tenant)},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				StorageClassName: stringPtr(profile.Spec.Storage.StorageClassName),
				Resources:        corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: profile.Spec.Storage.Size}},
			},
		}
		if err := controllerutil.SetControllerReference(agent, &pvc, r.Scheme); err != nil {
			return err
		}
		return r.Create(ctx, &pvc)
	}
	if err != nil {
		return err
	}
	if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != profile.Spec.Storage.StorageClassName {
		return fmt.Errorf("PVC %s/%s uses storageClass %q; profile %q requires %q (storageClassName is immutable)", namespace, name, valueOrEmpty(pvc.Spec.StorageClassName), profile.Name, profile.Spec.Storage.StorageClassName)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, &pvc, func() error {
		pvc.Labels = mergeStringMap(pvc.Labels, agentLabels(agent, tenant))
		return controllerutil.SetControllerReference(agent, &pvc, r.Scheme)
	})
	return err
}

func (r *AgentIdentityReconciler) ensureDeployment(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.AgentRuntimeProfile, namespace string) error {
	name := runtimeName(agent.Spec.AgentKey)
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		labels := agentLabels(agent, tenant)
		deployment.Labels = mergeStringMap(deployment.Labels, labels)
		deployment.Annotations = mergeStringMap(deployment.Annotations, map[string]string{AnnotationRuntimeState: "auth-blocked"})
		replicas := int32(1)
		revisionHistory := int32(2)
		deployment.Spec.Replicas = &replicas
		deployment.Spec.RevisionHistoryLimit = &revisionHistory
		deployment.Spec.Strategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
		deployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{LabelName: "hermes-agent", LabelInstance: agent.Spec.AgentKey}}

		podLabels := copyStringMap(labels)
		podLabels["vixens.io/sizing.hermes"] = defaultString(profile.Spec.SizingLabel, "V-small")
		podAnnotations := map[string]string{}
		if profile.Spec.Compatibility.S6Overlay {
			podAnnotations["vixens.io/explicitly-allow-root"] = "true"
		}
		probeCommand := []string{"/bin/sh", "-ec", `for f in /proc/[0-9]*/cmdline; do tr '\000' ' ' < "$f" 2>/dev/null | grep -q 'hermes gateway run' && exit 0; done; exit 1`}
		deployment.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: podLabels, Annotations: podAnnotations},
			Spec: corev1.PodSpec{
				PriorityClassName:             defaultString(profile.Spec.PriorityClassName, "vixens-medium"),
				TerminationGracePeriodSeconds: int64Ptr(30),
				InitContainers: []corev1.Container{{
					Name:            "bootstrap-profile",
					Image:           profile.Spec.Image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/bin/sh", "-lc", `if [ ! -d "/opt/data/profiles/${AGENT_NAME}" ]; then /opt/hermes/.venv/bin/hermes profile create "${AGENT_NAME}" --no-alias --no-skills --description "${AGENT_DISPLAY_NAME} - TXO Fabric agent"; fi`},
					Env: []corev1.EnvVar{{Name: "HERMES_HOME", Value: "/opt/data"}, {Name: "AGENT_NAME", Value: agent.Spec.AgentKey}, {Name: "AGENT_DISPLAY_NAME", Value: agent.Spec.DisplayName}},
					VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/opt/data"}},
				}},
				Containers: []corev1.Container{{
					Name:            "hermes",
					Image:           profile.Spec.Image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Args:            []string{"gateway", "run", "--replace"},
					Env: []corev1.EnvVar{
						{Name: "HERMES_HOME", Value: "/opt/data"},
						{Name: "TERMINAL_ENV", Value: "local"},
						{Name: "TXO_AGENT_ID", Value: agent.Name},
						{Name: "TXO_AGENT_KEY", Value: agent.Spec.AgentKey},
						{Name: "TXO_TENANT_ID", Value: tenant.Spec.TenantID},
						{Name: "TXO_TENANT_NAME", Value: tenant.Name},
						{Name: "TXO_MEMORY_BANK_ID", Value: resolvedBankID(agent)},
						{Name: "TXO_LLM_AUTH_MODE", Value: "unconfigured"},
					},
					Resources:      profile.Spec.Resources,
					VolumeMounts:   []corev1.VolumeMount{{Name: "data", MountPath: "/opt/data"}},
					StartupProbe:   &corev1.Probe{ProbeHandler: corev1.ProbeHandler{Exec: &corev1.ExecAction{Command: probeCommand}}, PeriodSeconds: 5, FailureThreshold: 30},
					ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{Exec: &corev1.ExecAction{Command: probeCommand}}, PeriodSeconds: 10, FailureThreshold: 3},
					LivenessProbe:  &corev1.Probe{ProbeHandler: corev1.ProbeHandler{Exec: &corev1.ExecAction{Command: probeCommand}}, PeriodSeconds: 20, FailureThreshold: 3},
				}},
				Volumes: []corev1.Volume{{Name: "data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: runtimePVCName(agent.Spec.AgentKey)}}}},
			},
		}
		return controllerutil.SetControllerReference(agent, deployment, r.Scheme)
	})
	return err
}

func (r *AgentIdentityReconciler) ensureEgressPolicy(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, namespace string) error {
	name := runtimeName(agent.Spec.AgentKey) + "-egress"
	np := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, np, func() error {
		np.Labels = mergeStringMap(np.Labels, agentLabels(agent, tenant))
		np.Spec = networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{LabelName: "hermes-agent", LabelInstance: agent.Spec.AgentKey}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress: []networkingv1.NetworkPolicyEgressRule{{
				To: []networkingv1.NetworkPolicyPeer{{NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "kube-system"}}}},
				Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolUDP), Port: intOrStringPtr(53)}, {Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(53)}},
			}},
		}
		if tenant.Spec.Memory.Hindsight != nil {
			np.Spec.Egress = append(np.Spec.Egress, networkingv1.NetworkPolicyEgressRule{
				To: []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{LabelName: "hindsight"}}}},
				Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(8888)}},
			})
		}
		return controllerutil.SetControllerReference(agent, np, r.Scheme)
	})
	return err
}
