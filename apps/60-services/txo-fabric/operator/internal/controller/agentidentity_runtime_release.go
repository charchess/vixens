package controller

import (
	"context"
	"fmt"
	"regexp"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

var releaseImageDigest = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:/-]*@sha256:[a-f0-9]{64}$`)

// resolveRuntimeRelease constructs a local effective runtime profile. It NEVER
// writes the copied image/dependencies into AgentRuntimeProfile: a profile is
// just a pointer plus platform-owned resource/security policy.
func (r *AgentIdentityReconciler) resolveRuntimeRelease(ctx context.Context, profile *fabricv1alpha1.AgentRuntimeProfile) (*fabricv1alpha1.AgentRuntimeProfile, string, error) {
	effective := profile.DeepCopy()
	ref := profile.Spec.ReleaseRef
	if ref == "" {
		if profile.Spec.Image == "" {
			return nil, "", fmt.Errorf("AgentRuntimeProfile %q has neither releaseRef nor inline image", profile.Name)
		}
		return effective, "", nil // backwards-compatible legacy inline profile
	}
	if profile.Spec.Image != "" || profile.Spec.Bootstrap.HindsightPluginImage != "" {
		return nil, "", fmt.Errorf("AgentRuntimeProfile %q mixes releaseRef with legacy image/bootstrap fields", profile.Name)
	}
	var release fabricv1alpha1.HermesRuntimeRelease
	if err := r.Get(ctx, types.NamespacedName{Name: ref}, &release); err != nil {
		return nil, "", fmt.Errorf("HermesRuntimeRelease %q for profile %q cannot be resolved: %w", ref, profile.Name, err)
	}
	if !releaseImageDigest.MatchString(release.Spec.Image) {
		return nil, "", fmt.Errorf("HermesRuntimeRelease %q must pin an OCI sha256 engine image", release.Name)
	}
	if release.Spec.HindsightPluginImage != "" && !hindsightBundleDigest.MatchString(release.Spec.HindsightPluginImage) {
		return nil, "", fmt.Errorf("HermesRuntimeRelease %q must pin an OCI sha256 Hindsight plugin image", release.Name)
	}
	effective.Spec.Image = release.Spec.Image
	effective.Spec.Bootstrap.HindsightPluginImage = release.Spec.HindsightPluginImage
	return effective, ref, nil
}
