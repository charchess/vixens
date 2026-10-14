package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/authentik"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/openfga"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
)

const hairemChatAgentName = "ten00001-usr000001-agt00002"

func chatAgent(name, tenant, uid, key string, iam []string) *fabricv1alpha1.AgentIdentity {
	return &fabricv1alpha1.AgentIdentity{
		ObjectMeta:metav1.ObjectMeta{
			Name:name, UID:types.UID(uid), Generation:1,
		},
		Spec:fabricv1alpha1.AgentIdentitySpec{
			TenantRef:fabricv1alpha1.ObjectReference{Name:tenant},
			AgentKey:key,
			HumanAccess:fabricv1alpha1.AgentHumanAccessBinding{Enabled:true},
			Access:fabricv1alpha1.AgentAccessSpec{
				UserRef:"charchess", Groups:[]string{"admin"}, ChatIAMGroups:iam,
			},
		},
		Status:fabricv1alpha1.AgentIdentityStatus{
			Phase:"Ready", ObservedGeneration:1,
		},
	}
}

func readyHAIremChat(t *testing.T) (*TenantBundleReconciler,*fabricv1alpha1.TenantBundle,*freshAccountSource,*recordingIAMWriter,string) {
	t.Helper()
	ctx:=context.Background()
	b:=humanIdentityTenant("hairem","TEN00001","uid-hairem","client0")
	r:=preparedHumanIdentityReconciler(t,b)
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err!=nil {t.Fatal(err)}
	installIAMIdentityBindings(t,r,b.Spec.TenantID,authentik.IdentityBinding{
		TenantID:b.Spec.TenantID,Issuer:b.Spec.HumanAccess.Web.OIDC.Issuer,
		OIDCSubject:"legacy-issuer-pinned-subject",
		AuthentikUserUUID:identityLedgerUserUUID,FabricUserID:"usr000001",
	})
	group:=authentikGroupName(b.Name,"client0")
	src:=&freshAccountSource{stagedIAMSource:&stagedIAMSource{
		groups:map[string]authentik.GroupMembership{
			group:makeIAMGroup(group,identityLedgerGroupUUID,identityLedgerUserUUID),
		},calls:map[string]int{},fail:map[string]error{},
	}}
	if err:=r.reconcileOpenFGAIAMGroupPins(ctx,b,src);err!=nil {t.Fatal(err)}
	w:=newIAMWriter()
	lease,reason:=r.reconcileFreshTenantMembers(ctx,b,src,w)
	if reason!="" || !AuthorizationLeaseFresh(lease,time.Now()) {
		t.Fatalf("cannot establish trusted hAIrem membership: %s %q",reason,lease)
	}
	return r,b,src,w,lease
}

func agentTuples(w *recordingIAMWriter, store, name, relation string) []openfga.Tuple {
	return w.current[store+"/agent:"+name+"/"+relation]
}

func TestHAIremDeclaredAgentChatGroupIsExplicitAndPinned(t *testing.T) {
	ctx:=context.Background()
	r,b,_,w,lease:=readyHAIremChat(t)
	a:=chatAgent(hairemChatAgentName,"hairem","immutable-agent-uid", "usr000001-agt00002",[]string{"client0"})
	if err:=r.Create(ctx,a);err!=nil {t.Fatal(err)}
	if err:=r.reconcileAgentChatGrants(ctx,b,w,lease);err!=nil {t.Fatal(err)}
	approved,err:=r.readOpenFGAAuthorizationBinding(ctx,b)
	if err!=nil {t.Fatal(err)}
	expectedChatter:=[]openfga.Tuple{{Object:"agent:"+a.Name,Relation:"chatter",
		User:"group:txo-fabric-hairem-client0#member"}}
	expectedTenant:=[]openfga.Tuple{{Object:"agent:"+a.Name,Relation:"tenant",
		User:"tenant:TEN00001"}}
	if !reflect.DeepEqual(agentTuples(w,approved.Store.ID,a.Name,"chatter"),expectedChatter) ||
		!reflect.DeepEqual(agentTuples(w,approved.Store.ID,a.Name,"tenant"),expectedTenant) {
		t.Fatalf("non-canonical agent chat relations: %v",w.current)
	}
	if len(w.stores)!=1 || !w.stores[approved.Store.ID] ||
		len(w.models)!=1 || !w.models[approved.ModelID] {
		t.Fatal("agent relation used foreign store/model")
	}
	binding:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
	var entries []retainedChatAgent
	if err:=json.Unmarshal([]byte(binding.Data[agentChatBindingsKey]),&entries);err!=nil ||
		!reflect.DeepEqual(entries,[]retainedChatAgent{{Name:a.Name,UID:string(a.UID)}}) {
		t.Fatal("durable agent cleanup inventory lacks immutable UID")
	}
	before:=binding.ResourceVersion
	for pass:=0;pass<2;pass++ {
		if err:=r.reconcileAgentChatGrants(ctx,b,w,lease);err!=nil {t.Fatal(err)}
		if after:=openFGAModelTestBinding(t,r,b.Spec.TenantID);after.ResourceVersion!=before {
			t.Fatal("repeated agent reconcile changed retained state")
		}
	}
	// Workspace userRef is never converted to FGA user:charchess nor a
	// supposed global OIDC sub. Authentik membership is verified separately.
	for _,tuple:=range expectedChatter {
		if strings.Contains(tuple.User,"charchess") || strings.Contains(tuple.User,"usr000001") {
			t.Fatal("mutable username or speculative Fabric ID used for a chat grant")
		}
	}
}

func TestHAIremChatScopesConvergeToEmptyOnDisabledDeletedOrMembershipRemoval(t *testing.T) {
	ctx:=context.Background()
	r,b,source,w,lease:=readyHAIremChat(t)
	a:=chatAgent(hairemChatAgentName,"hairem","uid-chat-v1","usr000001-agt00002",[]string{"client0"})
	if err:=r.Create(ctx,a);err!=nil {t.Fatal(err)}
	if err:=r.reconcileAgentChatGrants(ctx,b,w,lease);err!=nil {t.Fatal(err)}
	approved,err:=r.readOpenFGAAuthorizationBinding(ctx,b)
	if err!=nil {t.Fatal(err)}
	// Fresh Authentik removal empties group#member and tenant#member.
	group:=authentikGroupName(b.Name,"client0")
	source.groups[group]=makeIAMGroup(group,identityLedgerGroupUUID)
	fresh,reason:=r.reconcileFreshTenantMembers(ctx,b,source,w)
	if reason!="" || fresh=="" || len(w.at(approved.Store.ID,group))!=0 ||
		len(tenantMemberTuples(w,approved.Store.ID,b.Spec.TenantID))!=0 {
		t.Fatal("removed Authentik membership retained principal authorization")
	}
	// Agent not Ready -> no requested chat relation even if intent remains.
	var updated fabricv1alpha1.AgentIdentity
	if err:=r.Get(ctx,types.NamespacedName{Name:a.Name},&updated);err!=nil {t.Fatal(err)}
	updated.Status.Phase="Degraded"
	if err:=r.Update(ctx,&updated);err!=nil {t.Fatal(err)}
	if err:=r.reconcileAgentChatGrants(ctx,b,w,fresh);err!=nil {t.Fatal(err)}
	if len(agentTuples(w,approved.Store.ID,a.Name,"chatter"))!=0 ||
		len(agentTuples(w,approved.Store.ID,a.Name,"tenant"))!=0 {
		t.Fatal("degraded agent retained chat grant")
	}
	if err:=r.Delete(ctx,&updated);err!=nil {t.Fatal(err)}
	if err:=r.reconcileAgentChatGrants(ctx,b,w,fresh);err!=nil {t.Fatal(err)}
	record:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
	if record.Data[agentChatBindingsKey]!="[]" {t.Fatal("deleted AgentIdentity retained grant inventory")}
}

func TestHAIremAgentChatCannotInferPermissionOrCrossTenant(t *testing.T) {
	ctx:=context.Background()
	r,b,_,w,lease:=readyHAIremChat(t)
	a:=chatAgent(hairemChatAgentName,"hairem","uid-chat", "usr000001-agt00002",nil)
	foreign:=chatAgent("indiba-existing-agent","indiba","uid-indiba","existing-agent",[]string{"client0"})
	if err:=r.Create(ctx,a);err!=nil {t.Fatal(err)}
	if err:=r.Create(ctx,foreign);err!=nil {t.Fatal(err)}
	beforeCalls := len(w.calls) // already contains the two fresh IAM membership scopes
	if err:=r.reconcileAgentChatGrants(ctx,b,w,lease);err!=nil {t.Fatal(err)}
	binding,err:=r.readOpenFGAAuthorizationBinding(ctx,b)
	if err!=nil {t.Fatal(err)}
	if len(w.calls)!=beforeCalls {t.Fatalf("an unselected agent received chat tuples: %v",w.calls)}
	if len(agentTuples(w,binding.Store.ID,a.Name,"chatter"))!=0 ||
		len(agentTuples(w,binding.Store.ID,foreign.Name,"chatter"))!=0 {
		t.Fatal("workspace userRef or foreign agent became a grant")
	}
	// Changing only the declared local group to an unapproved key must
	// fail before any FGA write or persisted inventory mutation.
	invalid:=a.DeepCopy()
	invalid.Spec.Access.ChatIAMGroups=[]string{"admin"}
	if err:=r.Update(ctx,invalid);err!=nil {t.Fatal(err)}
	priorCalls:=len(w.calls)
	if err:=r.reconcileAgentChatGrants(ctx,b,w,lease);err==nil || len(w.calls)!=priorCalls {
		t.Fatal("workspace group key was treated as Authentik IAM group")
	}
}

func TestHAIremAgentChatFailuresNeverIssuePositiveLease(t *testing.T) {
	ctx:=context.Background()
	r,b,_,w,lease:=readyHAIremChat(t)
	a:=chatAgent(hairemChatAgentName,"hairem","uid-chat", "usr000001-agt00002",[]string{"client0"})
	if err:=r.Create(ctx,a);err!=nil {t.Fatal(err)}
	for _,tc:=range []struct{
		name string
		before func()
		after func()
	}{
		{"stale-lease",func(){},func(){}},
		{"writer-outage",func(){w.failFor="agent:"+a.Name},func(){w.failFor=""}},
		{"model-outage",func(){r.OpenFGAModelClient.(*fakeOpenFGAModelClient).validateErr=errors.New("unavailable")},
			func(){r.OpenFGAModelClient.(*fakeOpenFGAModelClient).validateErr=nil}},
	}{
		t.Run(tc.name,func(t *testing.T){
			tc.before()
			before:=len(w.calls)
			givenLease:=lease
			if tc.name=="stale-lease" {givenLease=time.Now().Add(-time.Second).UTC().Format(time.RFC3339Nano)}
			if err:=r.reconcileAgentChatGrants(ctx,b,w,givenLease);err==nil {
				t.Fatal("accepted incomplete/stale FGA authorization")
			}
			if tc.name!="writer-outage" && len(w.calls)!=before {
				t.Fatal("unsafe writer called after invalid prerequisite")
			}
			tc.after()
		})
	}
}

type failTenantWrite struct { original *recordingIAMWriter; fail bool }
func (f *failTenantWrite) ReconcileTupleScope(ctx context.Context, store,model string, scope openfga.TupleScope, tuples []openfga.Tuple) error {
	if f.fail && scope.Relation=="tenant" && strings.HasPrefix(scope.Object,"agent:") {
		return errors.New("partial FGA backend failure")
	}
	return f.original.ReconcileTupleScope(ctx,store,model,scope,tuples)
}
func TestPartialFGAAgentWritePreservesRevocationInventoryAcrossRestart(t *testing.T) {
	ctx:=context.Background()
	r,b,_,w,lease:=readyHAIremChat(t)
	a:=chatAgent(hairemChatAgentName,"hairem","uid-chat", "usr000001-agt00002",[]string{"client0"})
	if err:=r.Create(ctx,a);err!=nil {t.Fatal(err)}
	flaky:=&failTenantWrite{original:w,fail:true}
	if err:=r.reconcileAgentChatGrants(ctx,b,flaky,lease);err==nil {
		t.Fatal("partial FGA write presented as success")
	}
	record:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
	if !strings.Contains(record.Data[agentChatBindingsKey],hairemChatAgentName) {
		t.Fatal("missing durable cleanup scope after partial write")
	}
	approved,err:=r.readOpenFGAAuthorizationBinding(ctx,b);if err!=nil {t.Fatal(err)}
	if len(agentTuples(w,approved.Store.ID,a.Name,"chatter"))!=1 {
		t.Fatal("test did not produce actual partial write")
	}
	if err:=r.Delete(ctx,a);err!=nil {t.Fatal(err)}
	// A new controller instance uses the same retained Kubernetes objects
	// and store, and revokes the agent even though its CR no longer exists.
	restarted:=&TenantBundleReconciler{Client:r.Client,Scheme:r.Scheme,
		OpenFGAModelClient:r.OpenFGAModelClient}
	if err:=restarted.reconcileAgentChatGrants(ctx,b,w,lease);err!=nil {t.Fatal(err)}
	if len(agentTuples(w,approved.Store.ID,a.Name,"chatter"))!=0 {
		t.Fatal("orphaned grant survived restart and agent deletion")
	}
}

func TestAgentUIDChangeCannotSilentlyAdoptLegacyGrant(t *testing.T) {
	ctx:=context.Background()
	r,b,_,w,lease:=readyHAIremChat(t)
	a:=chatAgent(hairemChatAgentName,"hairem","old-agent-uid", "usr000001-agt00002",[]string{"client0"})
	if err:=r.Create(ctx,a);err!=nil {t.Fatal(err)}
	if err:=r.reconcileAgentChatGrants(ctx,b,w,lease);err!=nil {t.Fatal(err)}
	if err:=r.Delete(ctx,a);err!=nil {t.Fatal(err)}
	replacement:=chatAgent(hairemChatAgentName,"hairem","new-agent-uid","usr000001-agt00002",[]string{"client0"})
	if err:=r.Create(ctx,replacement);err!=nil {t.Fatal(err)}
	approved,err:=r.readOpenFGAAuthorizationBinding(ctx,b);if err!=nil {t.Fatal(err)}
	if err:=r.reconcileAgentChatGrants(ctx,b,w,lease);err==nil {
		t.Fatal("replacement AgentIdentity UID auto-adopted old chat grant")
	}
	if len(agentTuples(w,approved.Store.ID,a.Name,"chatter"))!=0 {
		t.Fatal("old agent chat grant not withdrawn during UID drift")
	}
}

func TestExistingHAIremGitOpsAgentsAreTheOnlyExplicitChatIntent(t *testing.T) {
	raw,err:=os.ReadFile("../../../tenants/hairem/agents.yaml")
	if err!=nil {t.Fatal(err)}
	decoder:=kyaml.NewYAMLOrJSONDecoder(bytes.NewReader(raw),4096)
	seen:=map[string]bool{}
	for {
		var doc struct {
			Kind string `json:"kind"`
			Metadata struct {Name string `json:"name"`} `json:"metadata"`
			Spec struct {
				TenantRef struct {Name string `json:"name"`} `json:"tenantRef"`
				AgentKey string `json:"agentKey"`
				Access struct {
					UserRef string `json:"userRef"`
					Groups []string `json:"groups"`
					ChatIAMGroups []string `json:"chatIAMGroups"`
				} `json:"access"`
			} `json:"spec"`
		}
		err:=decoder.Decode(&doc)
		if err==io.EOF {break}
		if err!=nil {t.Fatal(err)}
		if doc.Kind=="" {continue}
		if doc.Kind!="AgentIdentity" || doc.Spec.TenantRef.Name!="hairem" ||
			seen[doc.Metadata.Name] || doc.Spec.AgentKey=="" ||
			doc.Spec.Access.UserRef!="charchess" ||
			!reflect.DeepEqual(doc.Spec.Access.Groups,[]string{"admin"}) ||
			!reflect.DeepEqual(doc.Spec.Access.ChatIAMGroups,[]string{"client0"}) {
			t.Fatalf("unexpected or implicit tenant chat intent %q",doc.Metadata.Name)
		}
		seen[doc.Metadata.Name]=true
	}
	if len(seen)!=13 || !seen[hairemChatAgentName] {t.Fatal("existing hAIrem AgentIdentity inventory changed")}
}
