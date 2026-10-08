package controller

import (
	"context"
	"reflect"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
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

func TestFunctionalProfileNativeSkillRenderingAndWithdrawal(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	tenant.Name = "indiba"
	agent := functionalTestAgent("indiba-sam", tenant.Name)
	profile := testRuntimeProfile()
	scheme := testScheme(t)
	retained := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
		Name: "hermes-" + agent.Spec.AgentKey + "-data", Namespace: "tenant-indiba",
		UID: types.UID("retained-pvc-uid"),
	}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent, retained).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	role := effectiveFunctionalProfile{Enabled: true, ProfileName: "indiba-sales", Instructions: "Approved sales baseline", Revision: "aabbcc001122"}
	ns := "tenant-indiba"
	reconcile := func(f ...effectiveFunctionalProfile) {
		t.Helper()
		if len(f) > 0 && f[0].Enabled {
			if err := r.ensureFunctionalSkillConfigMap(ctx, agent, tenant, ns, f[0]); err != nil { t.Fatal(err) }
		}
		if err := r.ensureDeploymentWithIntegrations(ctx, agent, tenant, profile, ns,
			"same-key-uid", "same-key-revision", effectiveToolsetPolicy{Revision: "tools-v1", Enabled: []string{"skills"}},
			integrationResolution{}, f...); err != nil { t.Fatal(err) }
		if len(f) == 0 || !f[0].Enabled {
			if err := r.deleteFunctionalSkillConfigMap(ctx, agent, ns); err != nil { t.Fatal(err) }
		}
	}
	var dep appsv1.Deployment
	read := func() {
		t.Helper()
		if err := c.Get(ctx, types.NamespacedName{Name: "hermes-" + agent.Spec.AgentKey, Namespace: ns}, &dep); err != nil { t.Fatal(err) }
	}
	var cm corev1.ConfigMap
	readCM := func() {
		t.Helper()
		if err := c.Get(ctx, types.NamespacedName{Name: functionalSkillConfigMapName(agent), Namespace: ns}, &cm); err != nil { t.Fatal(err) }
	}
	reconcile(role)
	read()
	readCM()
	if cm.Data["SKILL.md"] != renderFunctionalSkill(role) || !strings.HasPrefix(cm.Data["SKILL.md"], "---\nname: txo-role-") {
		t.Fatalf("native Hermes SKILL.md missing from configmap: %#v", cm.Data)
	}
	if dep.Spec.Template.Annotations[AnnotationFunctionalProfileRevision] != role.Revision { t.Fatal("role revision missing from pod template") }
	if dep.Spec.Template.Spec.Volumes[0].Name != "data" { t.Fatal("private PVC changed") }
	main := dep.Spec.Template.Spec.Containers[0]
	if envValue(main.Env, "HERMES_HOME") != "/opt/data" { t.Fatal("private SOUL.md home changed") }
	if envVar(main.Env, "TXO_FUNCTIONAL_SYSTEM_PROMPT") != nil || envVar(main.Env, "HERMES_EPHEMERAL_SYSTEM_PROMPT") != nil {
		t.Fatal("native reference must never inject a global prompt override")
	}
	var roleMount *corev1.VolumeMount
	for i := range main.VolumeMounts {
		if main.VolumeMounts[i].Name == functionalSkillVolumeName { roleMount = &main.VolumeMounts[i] }
	}
	if roleMount == nil || !roleMount.ReadOnly || roleMount.MountPath != "/workspace/skills/functional/" + functionalSkillName(role.ProfileName) {
		t.Fatalf("functional skill not mounted read-only under native external root: %#v", roleMount)
	}
	var roleVolume *corev1.Volume
	for i := range dep.Spec.Template.Spec.Volumes {
		if dep.Spec.Template.Spec.Volumes[i].Name == functionalSkillVolumeName { roleVolume = &dep.Spec.Template.Spec.Volumes[i] }
	}
	if roleVolume == nil || roleVolume.ConfigMap == nil || roleVolume.ConfigMap.Name != functionalSkillConfigMapName(agent) {
		t.Fatalf("role ConfigMap volume absent: %#v", roleVolume)
	}
	initialRV, initialCMRV := dep.ResourceVersion, cm.ResourceVersion
	reconcile(role)
	read()
	readCM()
	if dep.ResourceVersion != initialRV || cm.ResourceVersion != initialCMRV {
		t.Fatalf("no-op role reconcile mutated runtime/configmap: dep %s->%s cm %s->%s", initialRV, dep.ResourceVersion, initialCMRV, cm.ResourceVersion)
	}
	role.Revision = "ddeeff334455"
	role.Instructions = "Updated sales baseline"
	reconcile(role)
	read()
	readCM()
	if !strings.Contains(cm.Data["SKILL.md"], "Updated sales baseline") || dep.Spec.Template.Annotations[AnnotationFunctionalProfileRevision] != role.Revision {
		t.Fatal("role update failed to propagate native skill and revision")
	}
	if cm.ResourceVersion == initialCMRV || dep.ResourceVersion == initialRV {
		t.Fatal("role update must replace ConfigMap and rollout, but no-op must not")
	}
	reconcile()
	read()
	if dep.Spec.Template.Annotations[AnnotationFunctionalProfileRevision] != "" { t.Fatal("unbound agent retains role revision") }
	for _, v := range dep.Spec.Template.Spec.Volumes {
		if v.Name == functionalSkillVolumeName { t.Fatal("unbound agent still mounts managed role") }
	}
	if err := c.Get(ctx, types.NamespacedName{Name: functionalSkillConfigMapName(agent), Namespace: ns}, &cm); !apierrors.IsNotFound(err) {
		t.Fatalf("unbound agent retains stale role ConfigMap: %v", err)
	}
	reconcile(role)
	read()
	if err := r.withdrawInvalidFunctionalRuntime(ctx, agent, tenant, ns); err != nil { t.Fatal(err) }
	if err := c.Get(ctx, types.NamespacedName{Name: dep.Name, Namespace: ns}, &dep); !apierrors.IsNotFound(err) {
		t.Fatalf("withdrawal did not stop obsolete runtime: %v", err)
	}
	var pvc corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Name: retained.Name, Namespace: ns}, &pvc); err != nil || pvc.UID != retained.UID {
		t.Fatalf("withdrawal must preserve private PVC: err=%v UID=%s", err, pvc.UID)
	}
}

func TestInvalidFunctionalProfileReconcileDeniesPreviouslyRunningAgent(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	tenant.Name = "indiba"
	profile := testRuntimeProfile()
	agent := functionalTestAgent("indiba-sam", "indiba")
	agent.Finalizers = []string{AgentFinalizer}
	agent.Spec.Functional.ProfileRef = "missing-approved-sales"
	agent.Status.Runtime.HumanEndpoint = "https://stale.invalid"
	ns := &corev1.Namespace{ObjectMeta:metav1.ObjectMeta{Name:"tenant-indiba"}}
	dep := &appsv1.Deployment{ObjectMeta:metav1.ObjectMeta{
		Name:runtimeName(agent.Spec.AgentKey),Namespace:ns.Name,
		Labels:map[string]string{LabelInstance:agent.Spec.AgentKey,LabelTenantName:"indiba"},
	}}
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.AgentIdentity{}).
		WithObjects(tenant, profile, agent, ns, dep).Build()
	r := &AgentIdentityReconciler{Client:c,Scheme:scheme}
	if _,err:=r.Reconcile(ctx, ctrl.Request{NamespacedName:types.NamespacedName{Name:agent.Name}});err!=nil {t.Fatal(err)}
	var current fabricv1alpha1.AgentIdentity
	if err:=c.Get(ctx,types.NamespacedName{Name:agent.Name},&current);err!=nil {t.Fatal(err)}
	condition:=apiMeta.FindStatusCondition(current.Status.Conditions,"FunctionalConfigurationReady")
	if condition==nil || condition.Status!=metav1.ConditionFalse || current.Status.Phase!="Degraded" {
		t.Fatalf("missing functional profile must fail closed: phase=%s conditions=%#v", current.Status.Phase, current.Status.Conditions)
	}
	if current.Status.Runtime.HumanEndpoint!="" {
		t.Fatalf("stale human endpoint remains advertised: %s",current.Status.Runtime.HumanEndpoint)
	}
	var stopped appsv1.Deployment
	if err:=c.Get(ctx,types.NamespacedName{Name:dep.Name,Namespace:ns.Name},&stopped);!apierrors.IsNotFound(err) {
		t.Fatalf("previous Hermes runtime still serving invalid role: %v",err)
	}
}

func TestFunctionalSkillAccessRequiresAuthorizedToolset(t *testing.T) {
	role := effectiveFunctionalProfile{Enabled: true, ProfileName: "indiba-sales"}
	if err := requireFunctionalSkillsAccess(role, effectiveToolsetPolicy{}); err == nil {
		t.Fatal("functional role must not be reported accessible when skills are disabled")
	}
	if err := requireFunctionalSkillsAccess(effectiveFunctionalProfile{}, effectiveToolsetPolicy{}); err != nil {
		t.Fatalf("unbound agent should be unchanged: %v", err)
	}
	if err := requireFunctionalSkillsAccess(role, effectiveToolsetPolicy{Enabled: []string{"skills"}}); err != nil {
		t.Fatalf("authorized skills toolset must support reference role: %v", err)
	}
}

func TestFunctionalSkillIsPrivateToAgentAndCannotDeleteForeignConfigMap(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	tenant.Name = "indiba"
	scheme := testScheme(t)
	agent := functionalTestAgent("indiba-sam", "indiba")
	other := functionalTestAgent("indiba-alex", "indiba")
	other.Spec.AgentKey = "usr000001-agt00002"
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent, other).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	role := effectiveFunctionalProfile{Enabled: true, ProfileName: "indiba-sales", Instructions: "Shared Indiba role", Revision: "rev"}
	for _, agent := range []*fabricv1alpha1.AgentIdentity{agent, other} {
		if err := r.ensureFunctionalSkillConfigMap(ctx, agent, tenant, "tenant-indiba", role); err != nil { t.Fatal(err) }
	}
	var a, b corev1.ConfigMap
	if err := c.Get(ctx, types.NamespacedName{Name: functionalSkillConfigMapName(agent), Namespace: "tenant-indiba"}, &a); err != nil { t.Fatal(err) }
	if err := c.Get(ctx, types.NamespacedName{Name: functionalSkillConfigMapName(other), Namespace: "tenant-indiba"}, &b); err != nil { t.Fatal(err) }
	if a.Name == b.Name || a.Data["SKILL.md"] != b.Data["SKILL.md"] {
		t.Fatal("same role must create distinct agent-owned ConfigMaps with identical approved content")
	}
	a.Labels[LabelInstance] = "someone-else"
	if err := c.Update(ctx, &a); err != nil { t.Fatal(err) }
	if err := r.deleteFunctionalSkillConfigMap(ctx, agent, "tenant-indiba"); err == nil {
		t.Fatal("must refuse deletion of a hijacked or foreign ConfigMap")
	}
	if err := r.deleteFunctionalSkillConfigMap(ctx, other, "tenant-indiba"); err != nil { t.Fatal(err) }
}
