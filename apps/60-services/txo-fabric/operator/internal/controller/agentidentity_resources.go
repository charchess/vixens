package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *AgentIdentityReconciler) ensurePVC(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.AgentRuntimeProfile, namespace string) error {
	name := runtimePVCName(agent.Spec.AgentKey)
	var pvc corev1.PersistentVolumeClaim
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &pvc)
	if apierrors.IsNotFound(err) {
		labels := agentLabels(agent, tenant)
		labels[LabelStorageRetention] = storageRetentionPolicy(agent)
		pvc = corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				StorageClassName: stringPtr(profile.Spec.Storage.StorageClassName),
				Resources:        corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: profile.Spec.Storage.Size}},
			},
		}
		// Retained storage must NEVER be garbage-collected with AgentIdentity.
		// Keep an ownerReference only for deliberately disposable workspaces.
		if err := r.setRuntimePVCOwnership(agent, &pvc); err != nil {
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
	if tenantName := pvc.Labels[LabelTenantName]; tenantName != "" && tenantName != tenant.Name {
		return fmt.Errorf("PVC %s/%s belongs to tenant %q, not %q", namespace, name, tenantName, tenant.Name)
	}
	if agentKey := pvc.Labels[LabelAgent]; agentKey != "" && agentKey != agent.Spec.AgentKey {
		return fmt.Errorf("PVC %s/%s belongs to agentKey %q, not %q", namespace, name, agentKey, agent.Spec.AgentKey)
	}
	if owner := metav1.GetControllerOf(&pvc); owner != nil && owner.UID != agent.UID {
		return fmt.Errorf("PVC %s/%s is controlled by %s %q and cannot be adopted", namespace, name, owner.Kind, owner.Name)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, &pvc, func() error {
		pvc.Labels = mergeStringMap(pvc.Labels, agentLabels(agent, tenant))
		pvc.Labels[LabelStorageRetention] = storageRetentionPolicy(agent)
		return r.setRuntimePVCOwnership(agent, &pvc)
	})
	if err != nil {
		return err
	}
	return r.ensureRetainedPersistentVolume(ctx, agent, &pvc)
}

// setRuntimePVCOwnership is called on creation AND while the agent is alive.
// Removing a Retain PVC's controller reference only inside its deletion finalizer
// races Kubernetes garbage collection. Keep these PVCs independent up-front.
// Delete PVCs stay controller-owned and are reclaimed through the normal path.
func (r *AgentIdentityReconciler) setRuntimePVCOwnership(agent *fabricv1alpha1.AgentIdentity, pvc *corev1.PersistentVolumeClaim) error {
	if storageRetentionPolicy(agent) == StorageRetentionDelete {
		return controllerutil.SetControllerReference(agent, pvc, r.Scheme)
	}
	kept := make([]metav1.OwnerReference, 0, len(pvc.OwnerReferences))
	for _, owner := range pvc.OwnerReferences {
		if owner.Kind == "AgentIdentity" && owner.Name == agent.Name && owner.UID == agent.UID {
			continue
		}
		kept = append(kept, owner)
	}
	pvc.OwnerReferences = kept
	return nil
}

func (r *AgentIdentityReconciler) ensureRetainedPersistentVolume(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, pvc *corev1.PersistentVolumeClaim) error {
	if storageRetentionPolicy(agent) != StorageRetentionRetain || pvc.Spec.VolumeName == "" {
		return nil
	}

	reader := client.Reader(r.Client)
	if r.APIReader != nil {
		reader = r.APIReader
	}

	var pv corev1.PersistentVolume
	if err := reader.Get(ctx, types.NamespacedName{Name: pvc.Spec.VolumeName}, &pv); err != nil {
		return fmt.Errorf("read bound PV %q for retained PVC %s/%s: %w", pvc.Spec.VolumeName, pvc.Namespace, pvc.Name, err)
	}
	if pv.Spec.ClaimRef == nil ||
		pv.Spec.ClaimRef.Namespace != pvc.Namespace ||
		pv.Spec.ClaimRef.Name != pvc.Name ||
		(pvc.UID != "" && pv.Spec.ClaimRef.UID != "" && pv.Spec.ClaimRef.UID != pvc.UID) {
		return fmt.Errorf("bound PV %q claimRef does not match retained PVC %s/%s", pv.Name, pvc.Namespace, pvc.Name)
	}
	if pv.Spec.PersistentVolumeReclaimPolicy == corev1.PersistentVolumeReclaimRetain {
		return nil
	}

	before := pv.DeepCopy()
	pv.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimRetain
	if err := r.Patch(ctx, &pv, client.MergeFrom(before)); err != nil {
		return fmt.Errorf("promote bound PV %q reclaim policy to Retain for PVC %s/%s: %w", pv.Name, pvc.Namespace, pvc.Name, err)
	}
	return nil
}

func (r *AgentIdentityReconciler) ensureDeployment(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.AgentRuntimeProfile, namespace, modelAccessSecretUID, modelAccessRevision string, policies ...effectiveToolsetPolicy) error {
	var toolPolicy effectiveToolsetPolicy
	var err error
	if len(policies) > 0 {
		toolPolicy = policies[0]
	} else {
		toolPolicy, err = resolveToolsetPolicy(agent, profile)
		if err != nil {
			return err
		}
	}
	return r.ensureDeploymentRuntime(ctx, agent, tenant, profile, namespace, modelAccessSecretUID, modelAccessRevision, toolPolicy, integrationResolution{})
}

func (r *AgentIdentityReconciler) ensureDeploymentWithIntegrations(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.AgentRuntimeProfile, namespace, modelAccessSecretUID, modelAccessRevision string, toolPolicy effectiveToolsetPolicy, integrations integrationResolution, functional ...effectiveFunctionalProfile) error {
	return r.ensureDeploymentRuntime(ctx, agent, tenant, profile, namespace, modelAccessSecretUID, modelAccessRevision, toolPolicy, integrations, functional...)
}

func (r *AgentIdentityReconciler) ensureDeploymentRuntime(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.AgentRuntimeProfile, namespace, modelAccessSecretUID, modelAccessRevision string, toolPolicy effectiveToolsetPolicy, integrations integrationResolution, functional ...effectiveFunctionalProfile) error {
	var role effectiveFunctionalProfile
	if len(functional) > 0 {
		role = functional[0]
	}
	workspaceVolumes, workspaceMounts, err := resolvedWorkspaceVolumes(agent, tenant)
	if err != nil {
		return err
	}
	integrationManifest, err := renderRuntimeIntegrationManifest(integrations.Effective)
	if err != nil {
		return err
	}
	humanAccess, err := resolveHumanAccess(agent, tenant)
	if err != nil {
		return err
	}
	modelBaseURL := defaultString(toolPolicy.ModelBaseURL, defaultAIGatewayURL+"/v1")
	modelDefault := defaultString(toolPolicy.ModelDefault, defaultAIGatewayModel)
	runtimeDataMount := corev1.VolumeMount{Name: "data", MountPath: "/opt/data"}
	bootstrapDataMount := runtimeDataMount
	bootstrapHome := "/opt/data"
	if agent.Spec.Runtime.Storage.AdoptLegacyProfile {
		if storageRetentionPolicy(agent) != StorageRetentionRetain {
			return fmt.Errorf("legacy Hermes profile adoption requires retained runtime storage")
		}
		legacySubPath := fmt.Sprintf("profiles/%s", agent.Spec.AgentKey)
		runtimeDataMount.SubPath = legacySubPath
		bootstrapDataMount = corev1.VolumeMount{Name: "data", MountPath: "/mnt/txo-data"}
		bootstrapHome = fmt.Sprintf("/mnt/txo-data/%s", legacySubPath)
	}
	managedPolicyMount := corev1.VolumeMount{Name: managedPolicyVolumeName, MountPath: managedPolicyMountPath, ReadOnly: true}
	hermesMounts := append([]corev1.VolumeMount{runtimeDataMount, managedPolicyMount}, workspaceMounts...)
	volumes := []corev1.Volume{
		{Name: "data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: runtimePVCName(agent.Spec.AgentKey)}}},
		{Name: managedPolicyVolumeName, VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: managedToolsetPolicyName(agent.Spec.AgentKey)},
		}}},
	}
	volumes = append(volumes, workspaceVolumes...)
	for _, integration := range integrations.Effective {
		volumes = append(volumes, corev1.Volume{
			Name: integration.VolumeName,
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
				SecretName: integration.CredentialName,
				Items: []corev1.KeyToPath{{Key: integrationCredentialKey, Path: integrationCredentialKey}},
			}},
		})
		hermesMounts = append(hermesMounts, corev1.VolumeMount{
			Name:      integration.VolumeName,
			MountPath: fmt.Sprintf("%s/%s", integrationCredentialMountRoot, integration.ConnectionName),
			ReadOnly:  true,
		})
	}

	name := runtimeName(agent.Spec.AgentKey)
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		labels := agentLabels(agent, tenant)
		deployment.Labels = mergeStringMap(deployment.Labels, labels)
		deployment.Annotations = mergeStringMap(deployment.Annotations, map[string]string{AnnotationRuntimeState: "gateway"})
		replicas := int32(1)
		revisionHistory := int32(2)
		deployment.Spec.Replicas = &replicas
		deployment.Spec.RevisionHistoryLimit = &revisionHistory
		deployment.Spec.Strategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
		deployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{LabelName: "hermes-agent", LabelInstance: agent.Spec.AgentKey}}

		podLabels := copyStringMap(labels)
		podLabels["vixens.io/sizing.hermes"] = defaultString(profile.Spec.SizingLabel, "V-small")
		podAnnotations := map[string]string{
			AnnotationModelAccessRevision:       modelAccessRevision,
			AnnotationModelAccessSecretUID:      modelAccessSecretUID,
			AnnotationToolsetPolicyRevision:     toolPolicy.Revision,
			AnnotationIntegrationPolicyRevision: integrations.Revision,
		}
		if role.Enabled {
			podAnnotations[AnnotationFunctionalProfileRevision] = role.Revision
		}
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
					Command: []string{"/bin/sh", "-lc", `legacy_profile="/opt/data/profiles/${AGENT_NAME}"
if [ "${TXO_LEGACY_PROFILE_ADOPTION}" = "true" ]; then
  if [ ! -d "${HERMES_HOME}" ]; then
    echo "requested legacy Hermes profile ${AGENT_NAME} is absent from retained runtime storage" >&2
    exit 78
  fi
elif [ -d "${legacy_profile}" ]; then
  if [ "${TXO_RUNTIME_STORAGE_RETENTION}" = "Delete" ]; then
    rm -rf -- "${legacy_profile}"
  else
    echo "refusing to discard retained Hermes profile ${legacy_profile}; explicit migration is required" >&2
    exit 78
  fi
fi

# The init container runs as root for retained-PVC migration checks, but the
# Hermes gateway itself runs as the image's hermes user. Keep only the
# Hermes-owned mutable config/backup surface writable by that runtime user.
install -d -o hermes -g hermes -m 0750 "${HERMES_HOME}/backups" "${HERMES_HOME}/backups/config"
if [ -e "${HERMES_HOME}/config.yaml" ]; then
  chown hermes:hermes "${HERMES_HOME}/config.yaml"
fi
/command/s6-setuidgid hermes /opt/hermes/.venv/bin/hermes config set skills.external_dirs '["/workspace/skills"]'`},
					Env: []corev1.EnvVar{
						{Name: "HERMES_HOME", Value: bootstrapHome},
						{Name: "HERMES_MANAGED_DIR", Value: managedPolicyMountPath},
						{Name: "HERMES_DISABLE_LAZY_INSTALLS", Value: "1"},
						{Name: "AGENT_NAME", Value: agent.Spec.AgentKey},
						{Name: "TXO_RUNTIME_STORAGE_RETENTION", Value: storageRetentionPolicy(agent)},
						{Name: "TXO_LEGACY_PROFILE_ADOPTION", Value: fmt.Sprintf("%t", agent.Spec.Runtime.Storage.AdoptLegacyProfile)},
						{Name: "TXO_TOOLSET_POLICY_REVISION", Value: toolPolicy.Revision},
					},
					VolumeMounts: []corev1.VolumeMount{bootstrapDataMount, managedPolicyMount},
				}},
				Containers: []corev1.Container{{
					Name:            "hermes",
					Image:           profile.Spec.Image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Args:            []string{"gateway", "run", "--replace"},
					Env: []corev1.EnvVar{
						{Name: "HERMES_HOME", Value: "/opt/data"},
						{Name: "HERMES_MANAGED_DIR", Value: managedPolicyMountPath},
						{Name: "HERMES_DISABLE_LAZY_INSTALLS", Value: "1"},
						{Name: "TXO_TOOLSET_POLICY_REVISION", Value: toolPolicy.Revision},
						// Hermes TUI/Desktop intentionally add client-only toolsets after config
						// resolution. This operator pin replaces that fold-in with the exact
						// platform-approved set for interactive UI sessions.
						{Name: "HERMES_TUI_TOOLSETS", Value: strings.Join(toolPolicy.Enabled, ",")},
						{Name: "TERMINAL_ENV", Value: "local"},
						{Name: "TXO_AGENT_ID", Value: agent.Name},
						{Name: "TXO_AGENT_KEY", Value: agent.Spec.AgentKey},
						{Name: "TXO_TENANT_ID", Value: tenant.Spec.TenantID},
						{Name: "TXO_TENANT_NAME", Value: tenant.Name},
						{Name: "TXO_MEMORY_BANK_ID", Value: resolvedBankID(agent)},
						{Name: "TXO_LLM_AUTH_MODE", Value: "gateway"},
						{Name: integrationManifestEnv, Value: integrationManifest},
						{Name: "OPENAI_BASE_URL", Value: modelBaseURL},
						{Name: "HERMES_MODEL", Value: modelDefault},
						{
							Name: "OPENAI_API_KEY",
							ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: modelAccessSecretName(agent.Spec.AgentKey)},
								Key:                  modelAccessSecretKey,
							}},
						},
					},
					Resources:      profile.Spec.Resources,
					VolumeMounts:   hermesMounts,
					StartupProbe:   &corev1.Probe{ProbeHandler: corev1.ProbeHandler{Exec: &corev1.ExecAction{Command: probeCommand}}, PeriodSeconds: 5, TimeoutSeconds: 3, FailureThreshold: 30},
					ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{Exec: &corev1.ExecAction{Command: probeCommand}}, PeriodSeconds: 10, TimeoutSeconds: 3, FailureThreshold: 3},
					LivenessProbe:  &corev1.Probe{ProbeHandler: corev1.ProbeHandler{Exec: &corev1.ExecAction{Command: probeCommand}}, PeriodSeconds: 20, TimeoutSeconds: 3, FailureThreshold: 3},
				}},
				Volumes: volumes,
			},
		}
		if role.Enabled {
			// The certified TXO Hermes runtime appends this approved role every gateway
			// turn, independently of personal and per-channel prompts. The
			// private SOUL.md and local skills are never overwritten.
			deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env,
				corev1.EnvVar{Name: "TXO_FUNCTIONAL_SYSTEM_PROMPT", Value: role.Instructions})
		}
		if humanAccess.Enabled {
			container := &deployment.Spec.Template.Spec.Containers[0]
			container.Ports = append(container.Ports, corev1.ContainerPort{
				Name:          "dashboard",
				ContainerPort: hermesDashboardPort,
				Protocol:      corev1.ProtocolTCP,
			})
			container.Env = append(container.Env,
				corev1.EnvVar{Name: "HERMES_DASHBOARD", Value: "1"},
				corev1.EnvVar{Name: "HERMES_DASHBOARD_HOST", Value: "0.0.0.0"},
				corev1.EnvVar{Name: "HERMES_DASHBOARD_PORT", Value: fmt.Sprintf("%d", hermesDashboardPort)},
				corev1.EnvVar{Name: "HERMES_DASHBOARD_PUBLIC_URL", Value: humanAccess.PublicURL},
				corev1.EnvVar{Name: "HERMES_DASHBOARD_OIDC_ISSUER", Value: humanAccess.OIDCIssuer},
				corev1.EnvVar{Name: "HERMES_DASHBOARD_OIDC_CLIENT_ID", Value: humanAccess.OIDCClientID},
				corev1.EnvVar{Name: "HERMES_DASHBOARD_OIDC_SCOPES", Value: humanAccess.OIDCScopes},
				corev1.EnvVar{Name: "HERMES_DASHBOARD_FILES_ROOT", Value: humanAccess.FilesRoot},
			)
		}
		if err := configureHermesHindsight(agent, tenant, profile, deployment); err != nil {
			return err
		}
		return controllerutil.SetControllerReference(agent, deployment, r.Scheme)
	})
	return err
}

type runtimeIntegrationManifestEntry struct {
	Binding        string   `json:"binding"`
	Connection     string   `json:"connection"`
	Protocol       string   `json:"protocol"`
	Endpoint       string   `json:"endpoint"`
	Authentication string   `json:"authentication"`
	Operations     []string `json:"operations"`
	Scopes         []string `json:"scopes,omitempty"`
	CredentialPath string   `json:"credentialPath"`
	Revision       string   `json:"revision"`
}

func renderRuntimeIntegrationManifest(integrations []runtimeIntegration) (string, error) {
	manifest := make([]runtimeIntegrationManifestEntry, 0, len(integrations))
	for _, integration := range integrations {
		manifest = append(manifest, runtimeIntegrationManifestEntry{
			Binding:        integration.BindingName,
			Connection:     integration.ConnectionName,
			Protocol:       integration.Protocol,
			Endpoint:       integration.Endpoint,
			Authentication: integration.Authentication,
			Operations:     append([]string(nil), integration.Operations...),
			Scopes:         append([]string(nil), integration.Scopes...),
			CredentialPath: integration.CredentialPath,
			Revision:       integration.Revision,
		})
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("render runtime integration manifest: %w", err)
	}
	return string(encoded), nil
}

func (r *AgentIdentityReconciler) ensureEgressPolicy(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, namespace string, resolutions ...integrationResolution) error {
	var integrations integrationResolution
	if len(resolutions) > 0 {
		integrations = resolutions[0]
	}
	name := runtimeName(agent.Spec.AgentKey) + "-egress"
	np := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, np, func() error {
		np.Labels = mergeStringMap(np.Labels, agentLabels(agent, tenant))
		np.Spec = networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{LabelName: "hermes-agent", LabelInstance: agent.Spec.AgentKey}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To:    []networkingv1.NetworkPolicyPeer{{NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "kube-system"}}}},
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolUDP), Port: intOrStringPtr(53)}, {Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(53)}},
				},
				{
					To: []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
						LabelManaged:    "true",
						LabelTenantName: tenant.Name,
						"app.kubernetes.io/component": "tenant-ai-gateway",
					}}}},
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(4000)}},
				},
			},
		}
		if tenant.Spec.Memory.Hindsight != nil {
			np.Spec.Egress = append(np.Spec.Egress, networkingv1.NetworkPolicyEgressRule{
				To: []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{LabelName: "hindsight"}}}},
				Ports: []networkingv1.NetworkPolicyPort{{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(8888)}},
			})
		}
		if len(integrations.Effective) > 0 {
			// v0 POC only: an effective Fabric integration binding admits broad HTTP(S)
			// egress. This is deliberately not a destination sandbox. The durable
			// production direction is policy-derived least-privilege egress/brokering.
			np.Spec.Egress = append(np.Spec.Egress, networkingv1.NetworkPolicyEgressRule{
				// Empty To means every destination. This is intentionally broad for
				// v0 POC compatibility and is not the production egress contract.
				Ports: []networkingv1.NetworkPolicyPort{
					{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(80)},
					{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(443)},
					{Protocol: protocolPtr(corev1.ProtocolTCP), Port: intOrStringPtr(8080)},
				},
			})
		}
		return controllerutil.SetControllerReference(agent, np, r.Scheme)
	})
	return err
}
