package controller

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func stringPtr(v string) *string                     { return &v }
func boolPtr(v bool) *bool                            { return &v }
func int64Ptr(v int64) *int64                        { return &v }
func protocolPtr(v corev1.Protocol) *corev1.Protocol { return &v }
func intOrStringPtr(v int) *intstr.IntOrString {
	value := intstr.FromInt(v)
	return &value
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mergeStringMap(dst, src map[string]string) map[string]string {
	if dst == nil {
		dst = map[string]string{}
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
