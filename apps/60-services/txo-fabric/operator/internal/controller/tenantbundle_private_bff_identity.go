package controller

import (
	"context"
	"errors"
	"strings"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var errPrivateIdentityUnavailable = errors.New("private Fabric identity unavailable")

// ResolvePrivateBFFIdentity is READ-ONLY and accepts no issuer, historical
// subject, Fabric ID, model/store, grant or authentication flag. Its caller
// MUST first authenticate the BFF ServiceAccount and consume the signed
// assertion's nonce in shared durable storage (privatebridge.Handler).
//
// The requested tenant slug is NOT an authorization grant. Every value used
// for resolution is loaded from the canonical cluster-scoped TenantBundle,
// its immutable UID-bound retained ledger and authoritative IAM source.
func (r *TenantBundleReconciler) ResolvePrivateBFFIdentity(
	ctx context.Context, tenantKey, uuid string, now time.Time,
) (string, error) {
	if r == nil || r.Client == nil || now.IsZero() || len(tenantKey)>63 {
		return "", errPrivateIdentityUnavailable
	}
	var bundle fabricv1alpha1.TenantBundle
	id := types.NamespacedName{Name:tenantKey}
	reader := r.APIReader
	if reader == nil { reader = r.Client }
	if err := reader.Get(ctx,id,&bundle);err!=nil {
		return "",errPrivateIdentityUnavailable
	}
	if bundle.Name!=tenantKey || !bundle.DeletionTimestamp.IsZero() ||
		bundle.Spec.HumanAccess==nil || bundle.Spec.HumanAccess.Web==nil ||
		bundle.Status.ObservedGeneration!=bundle.Generation {
		return "",errPrivateIdentityUnavailable
	}
	if !privateAuthorizationConditionFresh(bundle.Status.Conditions,bundle.Generation,now) {
		return "",errPrivateIdentityUnavailable
	}
	// Every ledger read checks retained tenant UID, store/model and model
	// fingerprint. An unavailable OpenFGA model verifier is a DENIAL.
	registry,err:=r.readOpenFGAHumanIdentityRegistry(ctx,&bundle)
	if err!=nil {return "",errPrivateIdentityUnavailable}
	fabricID,err:=registry.ResolveAuthentikUserUUID(uuid)
	if err!=nil {return "",errPrivateIdentityUnavailable}

	pins,err:=r.readOpenFGAIAMGroupPins(ctx,&bundle)
	if err!=nil {return "",errPrivateIdentityUnavailable}
	names,err:=checkedIAMGroupNames(&bundle)
	if err!=nil || len(pins)!=len(names) {
		return "",errPrivateIdentityUnavailable
	}
	iam,err:=r.authentikMembershipSource(ctx)
	if err!=nil {return "",errPrivateIdentityUnavailable}
	accounts,ok:=iam.(authoritativeIAMAccountReader)
	if !ok {return "",errPrivateIdentityUnavailable}
	user,err:=accounts.SnapshotUserAccount(ctx,uuid)
	if err!=nil || user.UUID!=uuid || !user.Active {return "",errPrivateIdentityUnavailable}
	authorized:=false
	for _,name:=range names {
		membership,readErr:=iam.SnapshotGroupMembership(ctx,name)
		if readErr!=nil || membership.Name!=name ||
			pins[name]=="" || pins[name]!=membership.AuthentikGroupUUID {
			return "",errPrivateIdentityUnavailable
		}
		ids,resolveErr:=registry.ResolveGroupMembership(membership)
		if resolveErr!=nil {return "",errPrivateIdentityUnavailable}
		for _,id:=range ids {if id==fabricID {authorized=true}}
	}
	if !authorized {return "",errPrivateIdentityUnavailable}
	// Re-check account, ledger and current readiness before returning: a
	// deactivation/ledger change during the request must not create a grant.
	after,err:=accounts.SnapshotUserAccount(ctx,uuid)
	if err!=nil || after.UUID!=uuid || !after.Active {
		return "",errPrivateIdentityUnavailable
	}
	again,err:=r.readOpenFGAHumanIdentityRegistry(ctx,&bundle)
	if err!=nil {return "",errPrivateIdentityUnavailable}
	confirmed,err:=again.ResolveAuthentikUserUUID(uuid)
	if err!=nil || confirmed!=fabricID {return "",errPrivateIdentityUnavailable}
	var current fabricv1alpha1.TenantBundle
	if err:=reader.Get(ctx,id,&current);err!=nil ||
		current.UID!=bundle.UID || current.Generation!=bundle.Generation ||
		current.Status.ObservedGeneration!=current.Generation ||
		!privateAuthorizationConditionFresh(current.Status.Conditions,current.Generation,r.authorizationClock()) {
		return "",errPrivateIdentityUnavailable
	}
	return fabricID,nil
}

// A persisted True is not an unbounded authorization grant. The lease can
// expire independently of status updates; never use LastTransitionTime as TTL.
func privateAuthorizationConditionFresh(conditions []metav1.Condition, generation int64, now time.Time) bool {
	if generation<=0 || now.IsZero() {return false}
	for _,c:=range conditions {
		if c.Type!="OpenFGAAuthorizationReady" {continue}
		if c.Status!=metav1.ConditionTrue || c.ObservedGeneration!=generation ||
			c.Reason!="TenantMembershipReconciled" ||
			!strings.HasPrefix(c.Message,"validUntil=") {
			return false
		}
		after:=strings.TrimPrefix(c.Message,"validUntil=")
		parts:=strings.SplitN(after,";",2)
		if len(parts)!=2 {return false}
		return AuthorizationLeaseFresh(strings.TrimSpace(parts[0]),now)
	}
	return false
}
