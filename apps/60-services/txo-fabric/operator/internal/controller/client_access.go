package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Get keeps platform-owned CloudNativePG Cluster reads out of the controller
// cache. TXO Fabric deliberately has only `get` RBAC on clusters; using the
// cached client for this dependency could start an informer and implicitly
// require list/watch permissions that the operator must not have.
//
// All Fabric-owned and watched resources continue through the normal cached
// client. Tests that construct the reconciler without a manager naturally fall
// back to that client as well.
func (r *TenantBundleReconciler) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if unstructuredObject, ok := obj.(*unstructured.Unstructured); ok && unstructuredObject.GroupVersionKind() == cnpgClusterGVK && r.APIReader != nil {
		return r.APIReader.Get(ctx, key, obj, opts...)
	}
	return r.Client.Get(ctx, key, obj, opts...)
}
