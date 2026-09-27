package v1alpha1

import runtime "k8s.io/apimachinery/pkg/runtime"

// Temporary bootstrap methods: the operator CI runs tests before controller-gen.
// These methods are removed once controller-gen has produced the authoritative
// zz_generated.deepcopy.go for the new profile CRDs.

func (in *PostgreSQLProfile) DeepCopyInto(out *PostgreSQLProfile) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	if in.Spec.ClusterRef != nil {
		out.Spec.ClusterRef = &NamespacedObjectReference{}
		*out.Spec.ClusterRef = *in.Spec.ClusterRef
	}
	if in.Spec.RequiredExtensions != nil {
		out.Spec.RequiredExtensions = append([]string(nil), in.Spec.RequiredExtensions...)
	}
}

func (in *PostgreSQLProfile) DeepCopy() *PostgreSQLProfile {
	if in == nil {
		return nil
	}
	out := new(PostgreSQLProfile)
	in.DeepCopyInto(out)
	return out
}

func (in *PostgreSQLProfile) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *PostgreSQLProfileList) DeepCopyInto(out *PostgreSQLProfileList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]PostgreSQLProfile, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (in *PostgreSQLProfileList) DeepCopy() *PostgreSQLProfileList {
	if in == nil {
		return nil
	}
	out := new(PostgreSQLProfileList)
	in.DeepCopyInto(out)
	return out
}

func (in *PostgreSQLProfileList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *HindsightProfile) DeepCopyInto(out *HindsightProfile) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Spec.Resources.DeepCopyInto(&out.Spec.Resources)
}

func (in *HindsightProfile) DeepCopy() *HindsightProfile {
	if in == nil {
		return nil
	}
	out := new(HindsightProfile)
	in.DeepCopyInto(out)
	return out
}

func (in *HindsightProfile) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *HindsightProfileList) DeepCopyInto(out *HindsightProfileList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]HindsightProfile, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (in *HindsightProfileList) DeepCopy() *HindsightProfileList {
	if in == nil {
		return nil
	}
	out := new(HindsightProfileList)
	in.DeepCopyInto(out)
	return out
}

func (in *HindsightProfileList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
