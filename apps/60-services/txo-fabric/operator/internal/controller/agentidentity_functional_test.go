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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func functionalTestAgent(name, tenant string) *fabricv1alpha1.AgentIdentity {
	return &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID(name + "-uid")},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: tenant},
			AgentKey: "usr000001-agt00001",
			DisplayName: name,
			Runtime: fabricv1alpha1.AgentRuntimeBinding{ProfileRef: "hermes-default"},
			Functional: fabricv1alpha1.AgentFunctionalBinding{ProfileRef: "indiba-sales"},
		},
	}
}

func TestFunctionalProfileResolutionAndTenantIsolation(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	tenant.Name = "indiba"
	profile := &fabricv1alpha1.AgentFunctionalProfile{
		ObjectMeta: metav1.ObjectMeta{Name: "indiba-sales"},
		Spec: fabricv1alpha1.AgentFunctionalProfileSpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: "indiba"},
			Instructions: "Share the Indiba sales methods while retaining individual personality.",
		},
	}
	agent := functionalTestAgent("indiba-sam", "indiba")
	objects := []client.Object{profile}
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(objects...).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: testScheme(t)}

	resolved, err := r.resolveFunctionalProfile(ctx, agent, tenant)
	if err != nil || !resolved.Enabled || resolved.ProfileName != profile.Name ||
		resolved.Instructions != profile.Spec.Instructions || resolved.Revision == "" {
		t.Fatalf("resolve authorized profile: %+v, %v", resolved, err)
	}
	again, err := r.resolveFunctionalProfile(ctx, agent, tenant)
	if err != nil || again.Revision != resolved.Revision {
		t.Fatalf("same baseline should have stable revision: %+v %v", again, err)
	}

	foreign := *agent.DeepCopy()
	foreign.Spec.TenantRef.Name = "hairem"
	if _, err := r.resolveFunctionalProfile(ctx, &foreign, &fabricv1alpha1.TenantBundle{ObjectMeta: metav1.ObjectMeta{Name: "hairem"}}); err == nil || !strings.Contains(err.Error(), "not owned") {
		t.Fatalf("cross-tenant functional profile not rejected: %v", err)
	}

	unbound := agent.DeepCopy()
	unbound.Spec.Functional.ProfileRef = ""
	noProfile, err := r.resolveFunctionalProfile(ctx, unbound, tenant)
	if err != nil || noProfile.Enabled || noProfile.Revision != "" {
		t.Fatalf("unbound agent must stay baseline: %+v %v", noProfile, err)
	}

	missing := agent.DeepCopy()
	missing.Spec.Functional.ProfileRef = "missing-sales"
	if _, err := r.resolveFunctionalProfile(ctx, missing, tenant); !apierrors.IsNotFound(err) && (err == nil || !strings.Contains(err.Error(), "unavailable")) {
		t.Fatalf("missing functional profile should fail closed: %v", err)
	}

	profile.Spec.Instructions = "  "
	if err := c.Update(ctx, profile); err != nil {
		t.Fatal(err)
	}
	if _, err := r.resolveFunctionalProfile(ctx, agent, tenant); err == nil || !strings.Contains(err.Error(), "invalid instructions") {
		t.Fatalf("blank instructions should fail closed: %v", err)
	}

	profile.Spec.Instructions = "Updated sales baseline"
	if err := c.Update(ctx, profile); err != nil {
		t.Fatal(err)
	}
	next, err := r.resolveFunctionalProfile(ctx, agent, tenant)
	if err != nil || next.Revision == resolved.Revision {
		t.Fatalf("changed baseline did not produce a new revision: %+v %v", next, err)
	}
}

func TestFunctionalProfileWatchDoesNotCrossTenant(t *testing.T) {
	ctx := context.Background()
	a := functionalTestAgent("indiba-sam", "indiba")
	b := functionalTestAgent("indiba-alex", "indiba")
	b.Spec.AgentKey = "usr000001-agt00002"
	c := functionalTestAgent("hairem-sam", "hairem")
	d := functionalTestAgent("indiba-clover", "indiba")
	d.Spec.Functional.ProfileRef = ""
	store := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(a,b,c,d).Build()
	r := &AgentIdentityReconciler{Client:store,Scheme:testScheme(t)}
	p := &fabricv1alpha1.AgentFunctionalProfile{
		ObjectMeta:metav1.ObjectMeta{Name:"indiba-sales"},
		Spec:fabricv1alpha1.AgentFunctionalProfileSpec{TenantRef:fabricv1alpha1.ObjectReference{Name:"indiba"},Instructions:"Sales"},
	}
	got := requestNames(r.requestsForFunctionalProfile(ctx,p))
	want := []string{"indiba-alex","indiba-sam"}
	if !reflect.DeepEqual(got,want) {t.Fatalf("watch fan-out=%v want=%v",got,want)}
}

func TestFunctionalProfileHermesGatewayRenderingAndWithdrawal(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	tenant.Name = "indiba"
	agent := functionalTestAgent("indiba-sam", tenant.Name)
	profile := testRuntimeProfile()
	scheme := testScheme(t)
	retained := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
		Name: "hermes-"+agent.Spec.AgentKey+"-data", Namespace: "tenant-indiba",
		UID: types.UID("retained-pvc-uid"),
	}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent, retained).Build()
	r := &AgentIdentityReconciler{Client:c,Scheme:scheme}
	role := effectiveFunctionalProfile{
		Enabled:true, ProfileName:"indiba-sales", Instructions:"Approved sales baseline",
		Revision:"aabbcc001122",
	}
	ns := "tenant-indiba"
	reconcile := func(f ...effectiveFunctionalProfile) {
		t.Helper()
		err := r.ensureDeploymentWithIntegrations(ctx,agent,tenant,profile,ns,
			"same-key-uid","same-key-revision",effectiveToolsetPolicy{Revision:"tools-v1"},
			integrationResolution{},f...)
		if err != nil {t.Fatal(err)}
	}
	var deployment appsv1.Deployment
	read := func() {
		t.Helper()
		if err:=c.Get(ctx,types.NamespacedName{Name:"hermes-"+agent.Spec.AgentKey,Namespace:ns},&deployment);err!=nil {t.Fatal(err)}
	}
	reconcile(role)
	read()
	v:=envVar(deployment.Spec.Template.Spec.Containers[0].Env,"HERMES_EPHEMERAL_SYSTEM_PROMPT")
	if v==nil || v.Value!="Approved sales baseline" || v.ValueFrom!=nil {
		t.Fatalf("Hermes role overlay not rendered: %#v",v)
	}
	if deployment.Spec.Template.Annotations[AnnotationFunctionalProfileRevision] != role.Revision {
		t.Fatal("functional revision missing from pod template")
	}
	if deployment.Spec.Template.Spec.Volumes[0].Name!="data" {
		t.Fatal("private runtime PVC mount unexpectedly changed")
	}
	if envValue(deployment.Spec.Template.Spec.Containers[0].Env,"HERMES_HOME")!="/opt/data" {
		t.Fatal("private HERMES_HOME changed")
	}
	firstRV:=deployment.ResourceVersion
	reconcile(role)
	read()
	if deployment.ResourceVersion!=firstRV {
		t.Fatalf("no-op role reconcile caused runtime mutation: %s -> %s",firstRV,deployment.ResourceVersion)
	}
	role.Revision="ddeeff334455"
	role.Instructions="Updated approved baseline"
	reconcile(role)
	read()
	if deployment.Spec.Template.Annotations[AnnotationFunctionalProfileRevision]!="ddeeff334455" ||
		envValue(deployment.Spec.Template.Spec.Containers[0].Env,"HERMES_EPHEMERAL_SYSTEM_PROMPT")!="Updated approved baseline" {
		t.Fatal("role revision did not propagate to Hermes pod template")
	}
	if deployment.ResourceVersion==firstRV {
		t.Fatal("approved role update failed to trigger pod template revision")
	}
	// Removing the reference restores today's baseline without touching private data.
	reconcile()
	read()
	if envVar(deployment.Spec.Template.Spec.Containers[0].Env,"HERMES_EPHEMERAL_SYSTEM_PROMPT")!=nil {
		t.Fatal("unbound agent still has a managed functional overlay")
	}
	if deployment.Spec.Template.Annotations[AnnotationFunctionalProfileRevision]!="" {
		t.Fatal("unbound agent still has role revision annotation")
	}
	reconcile(role)
	read()
	if err:=r.withdrawInvalidFunctionalRuntime(ctx,agent,tenant,ns);err!=nil {t.Fatal(err)}
	if err:=c.Get(ctx,types.NamespacedName{Name:deployment.Name,Namespace:ns},&deployment);!apierrors.IsNotFound(err) {
		t.Fatalf("invalid functional profile did not shut down previous runtime: %v",err)
	}
	var pvc corev1.PersistentVolumeClaim
	if err:=c.Get(ctx,types.NamespacedName{Name:retained.Name,Namespace:ns},&pvc);err!=nil ||
		pvc.UID!=retained.UID {
		t.Fatalf("profile withdrawal must preserve retained private PVC: err=%v UID=%s",err,pvc.UID)
	}
}
