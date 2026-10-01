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

func configureHermesHindsight(agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, deployment *appsv1.Deployment) error {
	if tenant.Spec.Memory.Hindsight == nil {
		return nil
	}
	if len(deployment.Spec.Template.Spec.InitContainers) == 0 || len(deployment.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("Hermes deployment template is missing bootstrap or runtime container")
	}

	bankID := resolvedBankID(agent)
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
config = {
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
config_path.write_text(json.dumps(config, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")
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
