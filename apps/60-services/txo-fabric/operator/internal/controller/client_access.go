package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Get keeps read-only platform dependencies and generated tenant credentials
// out of the controller cache. TXO Fabric deliberately has only `get` RBAC on
// CloudNativePG Clusters and does not need cluster-wide list/watch access to
// Secrets; using the cached client for either kind could start an informer and
// implicitly require permissions the operator must not have.
//
// All Fabric-owned and watched resources continue through the normal cached
// client. Tests that construct the reconciler without a manager naturally fall
// back to that client as well.
func (r *TenantBundleReconciler) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if r.APIReader != nil {
		if _, ok := obj.(*corev1.Secret); ok {
			return r.APIReader.Get(ctx, key, obj, opts...)
		}
		if unstructuredObject, ok := obj.(*unstructured.Unstructured); ok && unstructuredObject.GroupVersionKind() == cnpgClusterGVK {
			return r.APIReader.Get(ctx, key, obj, opts...)
		}
	}
	return r.Client.Get(ctx, key, obj, opts...)
}

// AgentIdentity model-access and v0 integration credentials are intentionally
// uncached. A cached Secret Get would lazily start a cluster-wide Secret informer
// and turn a narrow `get` permission into an implicit `list/watch` requirement.
func (r *AgentIdentityReconciler) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if r.APIReader != nil {
		if _, ok := obj.(*corev1.Secret); ok {
			return r.APIReader.Get(ctx, key, obj, opts...)
		}
	}
	return r.Client.Get(ctx, key, obj, opts...)
}
