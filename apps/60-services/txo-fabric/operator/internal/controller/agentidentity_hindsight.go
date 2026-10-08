package controller

import (
	"fmt"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	hermesHindsightAPIURL      = "http://hindsight:8888"
	hermesHindsightSecretName = "hindsight-runtime"
	hermesHindsightSecretKey  = "HINDSIGHT_API_TENANT_API_KEY"
)

const (
	hindsightExtensionVolume = "hindsight-extension"
	hindsightExtensionImageRoot = "/bundle"
	hindsightExtensionInitMount = "/extensions"
	hindsightExtensionDepsRoot = "/opt/txo-hindsight/python"
	hindsightExtensionPluginRoot = "/opt/data/plugins/hindsight"
)

// The OCI bundle carries ONLY the pinned Hindsight plugin and Python deps.
// It is never a replacement for the immutable upstream Hermes application image.
func configureHermesHindsight(agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.AgentRuntimeProfile, deployment *appsv1.Deployment) error {
	if tenant.Spec.Memory.Hindsight == nil {
		return nil
	}
	if len(deployment.Spec.Template.Spec.InitContainers) == 0 || len(deployment.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("Hermes deployment template is missing bootstrap or runtime container")
	}

	bankID := resolvedBankID(agent)
	if profile.Spec.Bootstrap.HindsightPluginImage != "" {
		if err := configureOfficialHermesHindsightPlugin(profile.Spec.Bootstrap.HindsightPluginImage, deployment); err != nil {
			return err
		}
	}
	bootstrap := &deployment.Spec.Template.Spec.InitContainers[0]
	if len(bootstrap.Command) < 3 {
		return fmt.Errorf("Hermes bootstrap container command is incomplete")
	}
	bootstrap.Env = append(bootstrap.Env, corev1.EnvVar{Name: "HINDSIGHT_BANK_ID", Value: bankID})
	bootstrap.Command[2] += `
/opt/hermes/.venv/bin/hermes config set memory.provider hindsight
/opt/hermes/.venv/bin/python - <<'PY'
import json
import os
from pathlib import Path

hermes_home = Path(os.environ["HERMES_HOME"])
config_path = hermes_home / "hindsight" / "config.json"
config_path.parent.mkdir(parents=True, exist_ok=True)
managed = {
    "mode": "local_external",
    "api_url": "http://hindsight:8888",
    "bank_id": os.environ["HINDSIGHT_BANK_ID"],
    "memory_mode": "context",
    # Deliberate empty string: the pinned provider maps []/missing to its
    # observation-only default, while an empty string disables the type filter.
    "recall_types": "",
    "auto_retain": True,
    "auto_recall": True,
    "retain_indicator": False,
    "recall_indicator": False,
}
# Preserve unrelated personal fields; TXO controls only the explicitly
# documented memory transport/bank switches. Reject malformed existing data.
if config_path.is_file():
    config = json.loads(config_path.read_text(encoding="utf-8"))
    if not isinstance(config, dict):
        raise ValueError("Hindsight config must be a JSON object")
else:
    config = {}
config.update(managed)
rendered = json.dumps(config, sort_keys=True, separators=(",", ":")) + "\n"
if not config_path.is_file() or config_path.read_text(encoding="utf-8") != rendered:
    import tempfile
    fd, tmp = tempfile.mkstemp(prefix=".config-", dir=str(config_path.parent))
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as output:
            output.write(rendered)
            output.flush()
            os.fsync(output.fileno())
        os.chmod(tmp, 0o644)
        os.replace(tmp, config_path)
    finally:
        if os.path.exists(tmp):
            os.unlink(tmp)
PY`

	hermes := &deployment.Spec.Template.Spec.Containers[0]
	hermes.Env = append(hermes.Env, corev1.EnvVar{
		Name: "HINDSIGHT_API_KEY",
		ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: hermesHindsightSecretName},
			Key:                  hermesHindsightSecretKey,
		}},
	})
	return nil
}


// Running the official Hermes image requires no fork and no writable plugin
// code on its private PVC. Install an independently published OCI bundle into
// an emptyDir on EACH pod creation, then expose the exact plugin path Hermes
// already discovers and Python dependencies in a dedicated read-only path.
// HERMES_HOME/SOUL.md remains untouched and user-owned.
func configureOfficialHermesHindsightPlugin(image string, deployment *appsv1.Deployment) error {
	if image == "" || len(deployment.Spec.Template.Spec.InitContainers) == 0 || len(deployment.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("Hindsight plugin bundle image or Hermes containers missing")
	}
	pod := &deployment.Spec.Template.Spec
	pod.Volumes = append(pod.Volumes, corev1.Volume{
		Name: hindsightExtensionVolume,
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	})
	pod.InitContainers = append([]corev1.Container{{
		Name: "prepare-hindsight-extension",
		Image: image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{"/bin/sh", "-ec"},
		Args: []string{`set -eu
test -s /bundle/hindsight/plugin.yaml
test -s /bundle/hindsight/__init__.py
test -d /bundle/python/hindsight_client
test -d /bundle/python/aiohttp_retry
mkdir -p /extensions/hindsight /extensions/python
cp -a /bundle/hindsight/. /extensions/hindsight/
cp -a /bundle/python/. /extensions/python/
test -s /extensions/hindsight/plugin.yaml
test -s /extensions/hindsight/__init__.py
`},
		VolumeMounts: []corev1.VolumeMount{{
			Name: hindsightExtensionVolume,
			MountPath: hindsightExtensionInitMount,
		}},
	}}, pod.InitContainers...)

	// Bootstrap runs after the plugin content exists and validates the real
	// external dependencies via the same PYTHONPATH as the s6 gateway.
	bootstrap := &pod.InitContainers[1]
	bootstrap.VolumeMounts = append(bootstrap.VolumeMounts, corev1.VolumeMount{
		Name: hindsightExtensionVolume, MountPath: "/txo-extension", ReadOnly: true,
	})
	bootstrap.Command[2] += `
test -s /txo-extension/hindsight/plugin.yaml
PYTHONPATH="/txo-extension/python" /opt/hermes/.venv/bin/python -c 'import hindsight_client, aiohttp_retry'
`
	runtime := &pod.Containers[0]
	runtime.VolumeMounts = append(runtime.VolumeMounts,
		corev1.VolumeMount{
			Name: hindsightExtensionVolume, MountPath: hindsightExtensionPluginRoot,
			SubPath: "hindsight", ReadOnly: true,
		},
		corev1.VolumeMount{
			Name: hindsightExtensionVolume, MountPath: hindsightExtensionDepsRoot,
			SubPath: "python", ReadOnly: true,
		},
	)
	// Deliberately never pip-install into /opt/hermes/.venv or HERMES_HOME.
	// Only this vetted two-wheel bundle is prepended to Python's search path.
	runtime.Env = append(runtime.Env, corev1.EnvVar{
		Name: "PYTHONPATH", Value: hindsightExtensionDepsRoot,
	})
	return nil
}
