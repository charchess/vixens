package controller

import (
	"context"
	"reflect"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func testHermesRelease(name, imageHex, pluginHex string) *fabricv1alpha1.HermesRuntimeRelease {
	release := &fabricv1alpha1.HermesRuntimeRelease{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: fabricv1alpha1.HermesRuntimeReleaseSpec{
			Image: "nousresearch/hermes-agent@sha256:" + strings.Repeat(imageHex, 64),
		},
	}
	if pluginHex != "" {
		release.Spec.HindsightPluginImage = "ghcr.io/charchess/txo-hermes-hindsight-plugin@sha256:" + strings.Repeat(pluginHex, 64)
	}
	return release
}

func TestResolveHermesRuntimeReleaseAndLegacyProfile(t *testing.T) {
	ctx := context.Background()
	release := testHermesRelease("hermes-sep24", "a", "b")
	profile := testRuntimeProfile()
	profile.Spec.ReleaseRef = release.Name
	profile.Spec.Image = ""
	profile.Spec.Bootstrap.HindsightPluginImage = ""
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(release).Build()
	r := &AgentIdentityReconciler{Client: c}
	effective, name, err := r.resolveRuntimeRelease(ctx, profile)
	if err != nil { t.Fatal(err) }
	if name != release.Name || effective.Spec.Image != release.Spec.Image ||
		effective.Spec.Bootstrap.HindsightPluginImage != release.Spec.HindsightPluginImage {
		t.Fatalf("unresolved release: name=%s effective=%#v", name, effective.Spec)
	}
	if profile.Spec.Image != "" || profile.Spec.Bootstrap.HindsightPluginImage != "" {
		t.Fatal("release resolver mutated the source runtime profile")
	}
	if effective.Spec.Storage.StorageClassName != profile.Spec.Storage.StorageClassName ||
		!reflect.DeepEqual(effective.Spec.Capabilities, profile.Spec.Capabilities) ||
		!reflect.DeepEqual(effective.Spec.Resources, profile.Spec.Resources) {
		t.Fatal("release selection changed platform-owned storage, limits or capabilities")
	}

	legacy := testRuntimeProfile()
	inline, legacyRelease, err := r.resolveRuntimeRelease(ctx, legacy)
	if err != nil || legacyRelease != "" || inline.Spec.Image != legacy.Spec.Image {
		t.Fatalf("legacy inline profile is not backwards compatible: release=%q err=%v", legacyRelease, err)
	}
	if inline == legacy { t.Fatal("resolver must return a defensive copy") }
}

func TestResolveRuntimeReleaseFailsClosed(t *testing.T) {
	ctx := context.Background()
	valid := testHermesRelease("valid-release", "a", "b")
	invalidImage := testHermesRelease("bad-image", "a", "")
	invalidImage.Spec.Image = "nousresearch/hermes-agent:v2026.9.24" // mutable tag cannot be a release
	invalidPlugin := testHermesRelease("bad-plugin", "a", "")
	invalidPlugin.Spec.HindsightPluginImage = "ghcr.io/charchess/hindsight:latest"
	r := &AgentIdentityReconciler{Client: fake.NewClientBuilder().WithScheme(testScheme(t)).
		WithObjects(valid, invalidImage, invalidPlugin).Build()}
	for _, tc := range []struct{name,ref,image,plugin,errorPart string}{
		{"no-mode", "", "", "", "neither releaseRef nor inline image"},
		{"missing", "not-here", "", "", "cannot be resolved"},
		{"mixed-image", valid.Name, "nousresearch/hermes-agent:v2026.9.24", "", "mixes releaseRef"},
		{"mixed-plugin", valid.Name, "", "ghcr.io/charchess/plugin@sha256:" + strings.Repeat("b", 64), "mixes releaseRef"},
		{"tagged-image", invalidImage.Name, "", "", "must pin an OCI sha256 engine image"},
		{"tagged-plugin", invalidPlugin.Name, "", "", "must pin an OCI sha256 Hindsight plugin image"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile := testRuntimeProfile()
			profile.Spec.ReleaseRef = tc.ref
			profile.Spec.Image = tc.image
			profile.Spec.Bootstrap.HindsightPluginImage = tc.plugin
			_, _, err := r.resolveRuntimeRelease(ctx, profile)
			if err == nil || !strings.Contains(err.Error(), tc.errorPart) {
				t.Fatalf("wanted %q, got %v", tc.errorPart, err)
			}
		})
	}
}

func TestRuntimeReleaseChangeFanoutOnlyToAttachedProfiles(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	sept := testHermesRelease("hermes-sep24", "a", "b")
	oct := testHermesRelease("hermes-oct02", "c", "d")
	profileDefault := testRuntimeProfile()
	profileDefault.Spec.ReleaseRef = sept.Name
	profileDefault.Spec.Image = ""
	profileCanary := profileDefault.DeepCopy()
	profileCanary.Name = "hermes-upgrade-canary"
	profileCanary.Spec.ReleaseRef = oct.Name
	profileLegacy := profileDefault.DeepCopy()
	profileLegacy.Name = "legacy-inline"
	profileLegacy.Spec.ReleaseRef = ""
	profileLegacy.Spec.Image = "ghcr.io/charchess/txo-fabric-hermes:old"
	agents := []*fabricv1alpha1.AgentIdentity{
		{ObjectMeta: metav1.ObjectMeta{Name: "hairem-default"}, Spec: fabricv1alpha1.AgentIdentitySpec{TenantRef: fabricv1alpha1.ObjectReference{Name:"hairem"}, AgentKey:"a1"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "indiba-default"}, Spec: fabricv1alpha1.AgentIdentitySpec{TenantRef: fabricv1alpha1.ObjectReference{Name:"indiba"}, AgentKey:"a2", Runtime: fabricv1alpha1.AgentRuntimeBinding{ProfileRef:"hermes-default"}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "hairem-canary"}, Spec: fabricv1alpha1.AgentIdentitySpec{TenantRef: fabricv1alpha1.ObjectReference{Name:"hairem"}, AgentKey:"a3", Runtime: fabricv1alpha1.AgentRuntimeBinding{ProfileRef:"hermes-upgrade-canary"}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "indiba-legacy"}, Spec: fabricv1alpha1.AgentIdentitySpec{TenantRef: fabricv1alpha1.ObjectReference{Name:"indiba"}, AgentKey:"a4", Runtime: fabricv1alpha1.AgentRuntimeBinding{ProfileRef:"legacy-inline"}}},
	}
	objects := []client.Object{sept,oct,profileDefault,profileCanary,profileLegacy}
	for _, a := range agents { objects = append(objects, a) }
	r := &AgentIdentityReconciler{Client:fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()}
	if got := requestNames(r.requestsForRuntimeRelease(ctx,sept)); !reflect.DeepEqual(got, []string{"hairem-default","indiba-default"}) {
		t.Fatalf("default release fan-out=%v",got)
	}
	if got := requestNames(r.requestsForRuntimeRelease(ctx,oct)); !reflect.DeepEqual(got, []string{"hairem-canary"}) {
		t.Fatalf("canary release fan-out=%v",got)
	}
	if got := r.requestsForRuntimeRelease(ctx, testHermesRelease("unreferenced", "d", "")); len(got) != 0 {
		t.Fatalf("unreferenced release caused rollouts: %#v", got)
	}
	// Immutable new release is introduced separately; changing only the
	// profile pointer routes new events to the same original agent set.
	profileDefault.Spec.ReleaseRef = oct.Name
	if err := r.Update(ctx,profileDefault); err != nil { t.Fatal(err) }
	if got := requestNames(r.requestsForRuntimeRelease(ctx,oct)); !reflect.DeepEqual(got, []string{"hairem-canary","hairem-default","indiba-default"}) {
		t.Fatalf("profile pointer move was not observed: %v",got)
	}
}

func TestInvalidRuntimeReleaseWithdrawsServingPodWithoutDeletingPVC(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	tenant.Name = "indiba"
	profile := testRuntimeProfile()
	profile.Spec.ReleaseRef = "missing-immutable-release"
	profile.Spec.Image = ""
	agent := testAgentIdentity()
	agent.Name = "indiba-canary"
	agent.Spec.TenantRef.Name = "indiba"
	agent.Finalizers = []string{AgentFinalizer}
	agent.Status.Runtime.HumanEndpoint = "https://stale.invalid"
	namespace := tenantNamespace(tenant.Name)
	ns := &corev1.Namespace{ObjectMeta:metav1.ObjectMeta{Name:namespace}}
	dep := &appsv1.Deployment{ObjectMeta:metav1.ObjectMeta{
		Name:runtimeName(agent.Spec.AgentKey),Namespace:namespace,
		Labels:map[string]string{LabelInstance:agent.Spec.AgentKey,LabelTenantName:tenant.Name},
	}}
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta:metav1.ObjectMeta{
		Name:runtimePVCName(agent.Spec.AgentKey),Namespace:namespace,UID:types.UID("retained-pvc")},
	}
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.AgentIdentity{}).
		WithObjects(tenant,profile,agent,ns,dep,pvc).Build()
	r := &AgentIdentityReconciler{Client:c,Scheme:scheme}
	if _,err:=r.Reconcile(ctx,ctrl.Request{NamespacedName:types.NamespacedName{Name:agent.Name}});err!=nil {t.Fatal(err)}
	var current fabricv1alpha1.AgentIdentity
	if err:=c.Get(ctx,types.NamespacedName{Name:agent.Name},&current);err!=nil {t.Fatal(err)}
	cond:=apiMeta.FindStatusCondition(current.Status.Conditions,"RuntimeProfileResolved")
	if current.Status.Phase!="Degraded" || cond==nil || cond.Reason!="RuntimeReleaseInvalid" || cond.Status!=metav1.ConditionFalse {
		t.Fatalf("missing release must fail closed: phase=%s conditions=%#v",current.Status.Phase,current.Status.Conditions)
	}
	if current.Status.Runtime.HumanEndpoint!="" {t.Fatal("stale external endpoint still advertised")}
	var remaining appsv1.Deployment
	if err:=c.Get(ctx,types.NamespacedName{Name:dep.Name,Namespace:namespace},&remaining);!apierrors.IsNotFound(err){
		t.Fatalf("obsolete runtime remains serving: %v",err)
	}
	var retained corev1.PersistentVolumeClaim
	if err:=c.Get(ctx,types.NamespacedName{Name:pvc.Name,Namespace:namespace},&retained);err!=nil || retained.UID!=pvc.UID {
		t.Fatalf("private PVC changed: err=%v UID=%s",err,retained.UID)
	}
}

func TestRuntimeReleasePointerChangesOnlyExecutableImage(t *testing.T) {
	ctx:=context.Background()
	tenant:=testTenant()
	tenant.Spec.Memory.Hindsight=nil // image-only test; provider separate tests cover memory
	agent:=testAgentIdentity()
	profile:=testRuntimeProfile()
	profile.Spec.ReleaseRef="hermes-sep24"
	profile.Spec.Image=""
	profile.Spec.Bootstrap.HindsightPluginImage=""
	r1:=testHermesRelease("hermes-sep24","a","")
	r2:=testHermesRelease("hermes-oct02","c","")
	scheme:=testScheme(t)
	c:=fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent,r1,r2).Build()
	r:=&AgentIdentityReconciler{Client:c,Scheme:scheme}
	namespace:=tenantNamespace(tenant.Name)
	for _,ref:=range []string{r1.Name,r1.Name,r2.Name}{
		profile.Spec.ReleaseRef=ref
		effective,_,err:=r.resolveRuntimeRelease(ctx,profile)
		if err!=nil {t.Fatal(err)}
		if err:=r.ensureDeployment(ctx,agent,tenant,effective,namespace,"","");err!=nil {t.Fatal(err)}
		var dep appsv1.Deployment
		if err:=c.Get(ctx,types.NamespacedName{Name:runtimeName(agent.Spec.AgentKey),Namespace:namespace},&dep);err!=nil {t.Fatal(err)}
		expected:=r1.Spec.Image
		if ref==r2.Name {expected=r2.Spec.Image}
		if dep.Spec.Template.Spec.Containers[0].Image!=expected {
			t.Fatalf("pointer=%s resolved deployment image=%s want %s",ref,dep.Spec.Template.Spec.Containers[0].Image,expected)
		}
		if dep.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath!="/opt/data" {
			t.Fatal("pointer change moved the private runtime workspace")
		}
	}
}
