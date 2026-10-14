package controller

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/openfga"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const agentChatBindingsKey = "agentChatBindings"

type retainedChatAgent struct {
	Name string `json:"name"`
	UID string `json:"uid"`
}

type desiredChatAgent struct {
	identity retainedChatAgent
	groups []string
}

func exactChatGroupKeys(agent *fabricv1alpha1.AgentIdentity, allowed map[string]bool) ([]string, error) {
	seen := map[string]bool{}
	var keys []string
	if len(agent.Spec.Access.ChatIAMGroups) > 32 {
		return nil, errors.New("too many declared Fabric agent chat IAM groups")
	}
	for _, key := range agent.Spec.Access.ChatIAMGroups {
		if !allowed[key] || seen[key] || key == "" {
			return nil, errors.New("unapproved or duplicate agent chat IAM group")
		}
		seen[key] = true
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}

// reconcileAgentChatGrants is only invoked AFTER the same TenantBundle
// reconcile verified fresh Authentik group#member and tenant#member tuples.
// A workspace username/group and AgentIdentity displayName are never grant
// subjects. New users enter solely through the immutable issuer-pinned ledger.
func (r *TenantBundleReconciler) reconcileAgentChatGrants(
	ctx context.Context, bundle *fabricv1alpha1.TenantBundle,
	writer scopedIAMTupleWriter, memberLease string,
) error {
	if r == nil || r.Client == nil || bundle == nil ||
		!bundle.DeletionTimestamp.IsZero() || writer == nil ||
		!AuthorizationLeaseFresh(memberLease, r.authorizationClock()) {
		return errors.New("agent chat reconciliation requires fresh tenant membership")
	}
	approved, err := r.readOpenFGAAuthorizationBinding(ctx, bundle)
	if err != nil { return err }
	pins, err := r.readOpenFGAIAMGroupPins(ctx, bundle)
	if err != nil { return err }
	// Resolve the ledger again: an untrusted issuer, changed UID or mismatched
	// store/model cannot authorize even one agent relation.
	if _, err := r.readOpenFGAHumanIdentityRegistry(ctx, bundle); err != nil { return err }
	keys := make(map[string]bool)
	for _, key := range bundle.Spec.HumanAccess.Web.IAMGroups {
		name := authentikGroupName(bundle.Name, key)
		if keys[key] || pins[name] == "" { return errors.New("invalid pinned tenant IAM group") }
		keys[key] = true
	}
	if len(pins) != len(keys) { return errors.New("untrusted tenant IAM group inventory") }

	var cm corev1.ConfigMap
	name, err := openFGABindingName(bundle.Spec.TenantID)
	if err != nil { return err }
	ref := types.NamespacedName{Namespace:openFGAPlatformNamespace, Name:name}
	if err := r.Get(ctx, ref, &cm); err != nil { return err }
	if cm.Data["tenantUID"] != string(bundle.UID) ||
		cm.Data["tenantID"] != bundle.Spec.TenantID ||
		cm.Data["tenantName"] != bundle.Name ||
		cm.Data["storeID"] != approved.Store.ID ||
		cm.Data["modelID"] != approved.ModelID ||
		cm.Data["modelFingerprint"] != approved.ModelFingerprint {
		return errors.New("untrusted retained agent chat tenant/store/model binding")
	}
	var previous []retainedChatAgent
	if encoded, present := cm.Data[agentChatBindingsKey]; present {
		if len(encoded) == 0 || len(encoded) > 65536 ||
			json.Unmarshal([]byte(encoded), &previous) != nil || previous == nil {
			return errors.New("invalid retained Fabric agent grant inventory")
		}
	}
	prev := map[string]retainedChatAgent{}
	for _, entry := range previous {
		if entry.Name == "" || entry.UID == "" || len(entry.Name) > 253 ||
			strings.ContainsAny(entry.Name, "/: ") || prev[entry.Name].UID != "" {
			return errors.New("ambiguous retained Fabric agent grant ownership")
		}
		prev[entry.Name] = entry
	}
	var objects fabricv1alpha1.AgentIdentityList
	reader := r.APIReader
	if reader == nil { reader = r.Client }
	if err := reader.List(ctx, &objects); err != nil { return err }
	desired := map[string]desiredChatAgent{}
	seenAgentKeys := map[string]bool{}
	for i := range objects.Items {
		agent := &objects.Items[i]
		if agent.Spec.TenantRef.Name != bundle.Name { continue }
		if agent.Name == "" || agent.UID == "" || agent.Spec.AgentKey == "" ||
			len(agent.Name) > 122 || strings.ContainsAny(agent.Name,"/: ") ||
			seenAgentKeys[agent.Spec.AgentKey] {
			return errors.New("ambiguous current Fabric agent identity")
		}
		seenAgentKeys[agent.Spec.AgentKey] = true
		groups, err := exactChatGroupKeys(agent, keys)
		if err != nil { return err }
		// No implied permission for humanAccess.enabled, workspace userRef
		// or even a Ready agent without explicit chatIAMGroups.
		if len(groups) == 0 { continue }
		desired[agent.Name] = desiredChatAgent{
			identity: retainedChatAgent{Name:agent.Name, UID:string(agent.UID)},
		}
		if agent.DeletionTimestamp.IsZero() && agent.Spec.HumanAccess.Enabled &&
			agent.Status.Phase == "Ready" && agent.Status.ObservedGeneration == agent.Generation &&
			agent.Generation > 0 {
			desired[agent.Name] = desiredChatAgent{
				identity: retainedChatAgent{Name:agent.Name, UID:string(agent.UID)},
				groups:groups,
			}
		}
	}

	// Persist the previous authoritative inventory so we can revoke grants
	// for deleted AgentIdentities; K8s cannot list a deleted object later.
	names := make([]string, 0, len(prev)+len(desired))
	scopes := map[string]bool{}
	for name := range prev { scopes[name] = true }
	for name := range desired { scopes[name] = true }
	for name := range scopes { names = append(names, name) }
	sort.Strings(names)
	// Record the union BEFORE any FGA write. If a later write, read-back
	// or CAS fails, subsequent reconciles still know the exact scopes to
	// revoke even when an AgentIdentity has since disappeared.
	pending := make([]retainedChatAgent, 0, len(names))
	for _, agentName := range names {
		item := prev[agentName]
		if item.UID == "" { item = desired[agentName].identity }
		pending = append(pending, item)
	}
	pendingJSON, err := json.Marshal(pending)
	if err != nil { return err }
	if cm.Data[agentChatBindingsKey] != string(pendingJSON) {
		cm.Data[agentChatBindingsKey] = string(pendingJSON)
		if err := r.Update(ctx,&cm); err != nil { return err }
	}
	uidDrift := false
	for _, agentName := range names {
		next := desired[agentName]
		before := prev[agentName]
		// Never rebind a persisted agent name to a recreated UID.
		// Withdraw old grants, but require a reviewed lifecycle transition
		// (absence and successful cleanup) before admitting its new identity.
		changedUID := before.UID != "" && next.identity.UID != "" &&
			before.UID != next.identity.UID
		if changedUID { uidDrift = true; next.groups = nil }
		object := "agent:" + agentName
		tenantScope := openfga.TupleScope{Object:object, Relation:"tenant"}
		chatterScope := openfga.TupleScope{Object:object, Relation:"chatter"}
		var tenantTuples, chatterTuples []openfga.Tuple
		if len(next.groups) > 0 && !changedUID {
			tenantTuples = []openfga.Tuple{{
				User:"tenant:"+bundle.Spec.TenantID, Relation:"tenant", Object:object,
			}}
			for _, key := range next.groups {
				chatterTuples = append(chatterTuples, openfga.Tuple{
					User:"group:"+authentikGroupName(bundle.Name,key)+"#member",
					Relation:"chatter", Object:object,
				})
			}
		}
		// OpenFGA writes are not a transaction across two relations.
		// For a new grant, bind the tenant FIRST, then the chatter permission:
		// a failure between writes must not leave a positive can_chat check.
		// For removal, revoke chatter FIRST, then tenant association.
		// Consumers must additionally require a current True Fabric
		// authorization lease: a partial write never publishes that lease.
		reconcile := func(scope openfga.TupleScope, tuples []openfga.Tuple) error {
			return writer.ReconcileTupleScope(ctx, approved.Store.ID, approved.ModelID,
				scope, tuples)
		}
		if len(chatterTuples) != 0 {
			if err := reconcile(tenantScope, tenantTuples); err != nil { return err }
			if err := reconcile(chatterScope, chatterTuples); err != nil { return err }
		} else {
			if err := reconcile(chatterScope, nil); err != nil { return err }
			if err := reconcile(tenantScope, nil); err != nil { return err }
		}
	}
	if uidDrift { return errors.New("retained Fabric agent UID changed without revoked adoption") }

	// Check that the catalog and pin did not change during external writes.
	var latest fabricv1alpha1.AgentIdentityList
	if err := reader.List(ctx, &latest); err != nil { return err }
	if !sameChatAgentSnapshot(objects.Items, latest.Items, bundle.Name) {
		return errors.New("Fabric agent lifecycle changed during grant reconciliation")
	}
	if confirmed, err := r.readOpenFGAAuthorizationBinding(ctx,bundle); err != nil || confirmed != approved {
		return errors.New("Fabric tenant model/store changed during agent grants")
	}
	if !AuthorizationLeaseFresh(memberLease,r.authorizationClock()) {
		return errors.New("tenant membership freshness expired during agent grant sync")
	}
	next := make([]retainedChatAgent, 0, len(desired))
	for name := range desired { next = append(next, desired[name].identity) }
	sort.Slice(next,func(i,j int)bool{return next[i].Name<next[j].Name})
	encoded, err := json.Marshal(next)
	if err != nil { return err }
	if cm.Data[agentChatBindingsKey] != string(encoded) {
		cm.Data[agentChatBindingsKey] = string(encoded)
		if err := r.Update(ctx,&cm); err != nil { return err }
	}
	return nil
}

// A complete second list makes AgentIdentity removal/UID/generation changes
// observable before publishing a positive tenant authorization lease.
func sameChatAgentSnapshot(a,b []fabricv1alpha1.AgentIdentity, tenant string) bool {
	selectAgents := func(input []fabricv1alpha1.AgentIdentity) map[string]fabricv1alpha1.AgentIdentity {
		out := map[string]fabricv1alpha1.AgentIdentity{}
		for _, agent := range input {
			if agent.Spec.TenantRef.Name == tenant { out[agent.Name] = agent }
		}
		return out
	}
	old,next := selectAgents(a),selectAgents(b)
	if len(old)!=len(next) { return false }
	for name, agent := range old {
		current,ok:=next[name]
		if !ok || agent.UID!=current.UID ||
			agent.Generation!=current.Generation ||
			!reflect.DeepEqual(agent.Spec,current.Spec) ||
			!reflect.DeepEqual(agent.DeletionTimestamp,current.DeletionTimestamp) ||
			agent.Status.Phase!=current.Status.Phase ||
			agent.Status.ObservedGeneration!=current.Status.ObservedGeneration {
			return false
		}
	}
	return true
}
