package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	AnnotationToolsetPolicyRevision = "fabric.truxonline.io/toolset-policy-revision"
	managedPolicyConfigKey          = "config.yaml"
	managedPolicyVolumeName         = "managed-policy"
	managedPolicyMountPath          = "/etc/hermes"
)

var managedHermesPlatforms = []string{
	"cli",
	"tui",
	"desktop",
	"acp",
	"telegram",
	"discord",
	"slack",
	"whatsapp",
	"whatsapp_cloud",
	"signal",
	"bluebubbles",
	"email",
	"homeassistant",
	"mattermost",
	"matrix",
	"dingtalk",
	"feishu",
	"wecom",
	"wecom_callback",
	"weixin",
	"qqbot",
	"yuanbao",
	"webhook",
	"api_server",
	"cron",
}

type effectiveToolsetPolicy struct {
	Revision string
	Enabled  []string
	Denied   []string
	Config   string
}

func managedToolsetPolicyName(agentKey string) string {
	return runtimeName(agentKey) + "-managed-policy"
}

func resolveToolsetPolicy(agent *fabricv1alpha1.AgentIdentity, profile *fabricv1alpha1.AgentRuntimeProfile) (effectiveToolsetPolicy, error) {
	entries := profile.Spec.Capabilities.Toolsets
	if len(entries) == 0 {
		return effectiveToolsetPolicy{}, fmt.Errorf("AgentRuntimeProfile %q does not declare capabilities.toolsets", profile.Name)
	}

	states := make(map[string]string, len(entries))
	normalizedEntries := append([]fabricv1alpha1.RuntimeToolsetPolicy(nil), entries...)
	sort.Slice(normalizedEntries, func(i, j int) bool {
		if normalizedEntries[i].Name == normalizedEntries[j].Name {
			return normalizedEntries[i].State < normalizedEntries[j].State
		}
		return normalizedEntries[i].Name < normalizedEntries[j].Name
	})

	for _, entry := range normalizedEntries {
		if entry.Name == "no_mcp" {
			return effectiveToolsetPolicy{}, fmt.Errorf("toolset name %q is reserved by the Fabric Hermes policy bridge", entry.Name)
		}
		if _, exists := states[entry.Name]; exists {
			return effectiveToolsetPolicy{}, fmt.Errorf("AgentRuntimeProfile %q declares toolset %q more than once", profile.Name, entry.Name)
		}
		switch entry.State {
		case fabricv1alpha1.ToolsetPolicyOn, fabricv1alpha1.ToolsetPolicyOff, fabricv1alpha1.ToolsetPolicyAllowedOff:
			states[entry.Name] = entry.State
		default:
			return effectiveToolsetPolicy{}, fmt.Errorf("AgentRuntimeProfile %q declares unsupported state %q for toolset %q", profile.Name, entry.State, entry.Name)
		}
	}

	requested := append([]string(nil), agent.Spec.Runtime.Capabilities.EnableToolsets...)
	sort.Strings(requested)
	requestedSet := make(map[string]struct{}, len(requested))
	for _, name := range requested {
		state, exists := states[name]
		if !exists {
			return effectiveToolsetPolicy{}, fmt.Errorf("AgentIdentity %q requests undeclared toolset %q", agent.Name, name)
		}
		if state != fabricv1alpha1.ToolsetPolicyAllowedOff {
			return effectiveToolsetPolicy{}, fmt.Errorf("AgentIdentity %q may enable toolset %q only when the profile state is AllowedOff (state=%s)", agent.Name, name, state)
		}
		requestedSet[name] = struct{}{}
	}

	enabled := make([]string, 0, len(entries))
	denied := make([]string, 0, len(entries))
	for _, entry := range normalizedEntries {
		switch entry.State {
		case fabricv1alpha1.ToolsetPolicyOn:
			enabled = append(enabled, entry.Name)
		case fabricv1alpha1.ToolsetPolicyAllowedOff:
			if _, ok := requestedSet[entry.Name]; ok {
				enabled = append(enabled, entry.Name)
			} else {
				denied = append(denied, entry.Name)
			}
		case fabricv1alpha1.ToolsetPolicyOff:
			denied = append(denied, entry.Name)
		}
	}
	sort.Strings(enabled)
	sort.Strings(denied)

	platformSelection := append(append([]string(nil), enabled...), "no_mcp")
	platforms := make(map[string][]string, len(managedHermesPlatforms))
	knownPluginToolsets := make(map[string][]string, len(managedHermesPlatforms))
	knownPolicyNames := make([]string, 0, len(normalizedEntries))
	for _, entry := range normalizedEntries {
		knownPolicyNames = append(knownPolicyNames, entry.Name)
	}
	for _, platform := range managedHermesPlatforms {
		platforms[platform] = append([]string(nil), platformSelection...)
		// Hermes plugin toolsets are otherwise auto-enabled on first discovery unless
		// the platform has previously seen them. Treat every profile-declared name as
		// known so an Off/AllowedOff plugin toolset cannot auto-admit itself.
		knownPluginToolsets[platform] = append([]string(nil), knownPolicyNames...)
	}

	managedConfig := map[string]any{
		"platform_toolsets":     platforms,
		"known_plugin_toolsets": knownPluginToolsets,
		"agent": map[string]any{
			"disabled_toolsets": denied,
		},
		"security": map[string]any{
			"allow_lazy_installs": false,
		},
		"plugins": map[string]any{
			"enabled": []string{},
		},
	}
	configJSON, err := json.MarshalIndent(managedConfig, "", "  ")
	if err != nil {
		return effectiveToolsetPolicy{}, fmt.Errorf("render managed Hermes policy: %w", err)
	}
	configJSON = append(configJSON, '\n')

	revisionInput := struct {
		ProfileName string                                `json:"profileName"`
		Image       string                                `json:"image"`
		Entries     []fabricv1alpha1.RuntimeToolsetPolicy `json:"entries"`
		Requested   []string                              `json:"requested"`
	}{
		ProfileName: profile.Name,
		Image:       profile.Spec.Image,
		Entries:     normalizedEntries,
		Requested:   requested,
	}
	revisionJSON, err := json.Marshal(revisionInput)
	if err != nil {
		return effectiveToolsetPolicy{}, fmt.Errorf("hash managed Hermes policy: %w", err)
	}
	sum := sha256.Sum256(revisionJSON)
	revision := hex.EncodeToString(sum[:8])

	return effectiveToolsetPolicy{
		Revision: revision,
		Enabled:  enabled,
		Denied:   denied,
		Config:   string(configJSON),
	}, nil
}

func (r *AgentIdentityReconciler) ensureManagedToolsetPolicy(
	ctx context.Context,
	agent *fabricv1alpha1.AgentIdentity,
	tenant *fabricv1alpha1.TenantBundle,
	namespace string,
	policy effectiveToolsetPolicy,
) error {
	name := managedToolsetPolicyName(agent.Spec.AgentKey)
	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		configMap.Labels = mergeStringMap(configMap.Labels, agentLabels(agent, tenant))
		configMap.Annotations = mergeStringMap(configMap.Annotations, map[string]string{
			AnnotationToolsetPolicyRevision: policy.Revision,
		})
		configMap.Data = map[string]string{managedPolicyConfigKey: policy.Config}
		return controllerutil.SetControllerReference(agent, configMap, r.Scheme)
	})
	return err
}
