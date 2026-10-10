package controller

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "strings"

    fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
    "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/authentik"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/types"
)

// trustedHumanOIDCProof MUST be constructed only after a real server-side
// signed Authentik ID-token verification and a single-use OAuth nonce/state
// check. It is deliberately private to Fabric, never a CRD, browser JSON or
// an untrusted HTTP header. Future BFF→Fabric transport MUST be authenticated
// and authoritatively bind its tenant; that transport is NOT implemented here.
type trustedHumanOIDCProof struct {
    issuer string
    subject string
    authentikUUID string
}

type authoritativeHumanAccountReader interface {
    SnapshotUserAccount(context.Context, string) (authentik.UserAccountSnapshot, error)
}

func allocateOpaqueFabricUserID() (string,error) {
    var random [16]byte
    if _,err:=rand.Read(random[:]);err!=nil {return "",errors.New("cannot allocate Fabric user identity")}
    return "usr"+hex.EncodeToString(random[:]),nil
}

func decodeRetainedIdentityBindings(raw string) ([]authentik.IdentityBinding,error) {
    if len(raw)==0 || len(raw)>maxIdentityBindingJSONBytes {
        return nil,errors.New("missing or oversized retained Fabric identity ledger")
    }
    var bindings []authentik.IdentityBinding
    dec:=json.NewDecoder(strings.NewReader(raw))
    dec.DisallowUnknownFields()
    if err:=dec.Decode(&bindings);err!=nil || bindings==nil {
        return nil,errors.New("malformed retained Fabric identity ledger")
    }
    var extra any
    if dec.Decode(&extra)!=io.EOF {
        return nil,errors.New("trailing data in Fabric identity ledger")
    }
    return bindings,nil
}

// enrollHumanFromVerifiedIdentity is a STAGED privileged write primitive;
// it is not wired into an HTTP endpoint or the TenantBundle reconcile loop.
//
// The signed Authentik claim provides (issuer, opaque sub, immutable UUID).
// Fabric proves the UUID is still an ACTIVE Authentik user before accepting
// this association, and rechecks after writing. It does not infer membership,
// issue a chat grant, create an Authentik account, or change the OIDC sub_mode.
//
// The retained store+model+tenant-UID association is verified against OpenFGA
// before every write, and the ConfigMap Update is resourceVersion/CAS guarded.
// Concurrent updates return Conflict, never blind overwrite: the trusted
// enrollment caller must retry from a fresh read and revalidate its proof.
// User IDs are generated from cryptographic randomness, not from login, sub,
// UUID, workspace paths or browser values. An existing UUID/subject cannot be
// silently rebound (account replacement requires a reviewed migration).
func (r *TenantBundleReconciler) enrollHumanFromVerifiedIdentity(
    ctx context.Context, bundle *fabricv1alpha1.TenantBundle,
    proof trustedHumanOIDCProof, source authoritativeHumanAccountReader,
) (string,error) {
    if source==nil || bundle==nil || !bundle.DeletionTimestamp.IsZero() ||
        bundle.Spec.HumanAccess==nil || bundle.Spec.HumanAccess.Web==nil {
        return "",errors.New("trusted Fabric human enrollment dependencies unavailable")
    }
    if err:=validateTenantIAM(bundle);err!=nil {return "",err}
    issuer:=bundle.Spec.HumanAccess.Web.OIDC.Issuer
    // Do not normalize opaque subjects or infer uuid=sub. Validate identity
    // domains using the same strict registry contract as authorization reads.
    claim:=authentik.IdentityBinding{
        TenantID: bundle.Spec.TenantID, Issuer: proof.issuer,
        OIDCSubject: proof.subject, AuthentikUserUUID: proof.authentikUUID,
        FabricUserID: "usr-validation-only",
    }
    if proof.issuer!=issuer {
        return "",errors.New("OIDC issuer does not match Fabric tenant")
    }
    if _,err:=authentik.NewIdentityRegistry(bundle.Name,bundle.Spec.TenantID,issuer,
        bundle.Spec.HumanAccess.Web.IAMGroups,[]authentik.IdentityBinding{claim});err!=nil {
        return "",errors.New("invalid signed Authentik identity claim")
    }

    account,err:=source.SnapshotUserAccount(ctx,proof.authentikUUID)
    if err!=nil {return "",fmt.Errorf("trusted Authentik account unavailable: %w",err)}
    if account.UUID!=proof.authentikUUID || !account.Active {
        return "",errors.New("AuthentiK account inactive or inconsistent")
    }

    // Validate the retained tenantUID/store/model against its *live* model
    // before accessing an identity ledger; source-only ledger presence is
    // never evidence that a human is authorized to chat.
    if _,err:=r.readOpenFGAHumanIdentityRegistry(ctx,bundle);err!=nil {
        return "",fmt.Errorf("Fabric human enrollment ledger untrusted: %w",err)
    }
    approved,err:=r.readOpenFGAAuthorizationBinding(ctx,bundle)
    if err!=nil {return "",err}
    name,err:=openFGABindingName(bundle.Spec.TenantID)
    if err!=nil {return "",err}
    var cm corev1.ConfigMap
    if err:=r.Get(ctx,types.NamespacedName{Namespace:openFGAPlatformNamespace,Name:name},&cm);err!=nil {
        return "",err
    }
    if cm.Data["tenantID"]!=bundle.Spec.TenantID ||
        cm.Data["tenantName"]!=bundle.Name ||
        cm.Data["tenantUID"]!=string(bundle.UID) ||
        cm.Data["storeID"]!=approved.Store.ID ||
        cm.Data["modelID"]!=approved.ModelID ||
        cm.Data["modelFingerprint"]!=approved.ModelFingerprint ||
        cm.Data[identityIssuerKey]!=issuer ||
        cm.Data[identitySchemaKey]!="v1" {
        return "",errors.New("retained Fabric identity ledger changed ownership before enrollment")
    }
    existing,err:=decodeRetainedIdentityBindings(cm.Data[identityBindingsKey])
    if err!=nil {return "",err}
    if _,err:=authentik.NewIdentityRegistry(bundle.Name,bundle.Spec.TenantID,issuer,
        bundle.Spec.HumanAccess.Web.IAMGroups,existing);err!=nil {
        return "",errors.New("retained Fabric identity mappings inconsistent")
    }
    for _,previous:=range existing {
        if previous.OIDCSubject==proof.subject || previous.AuthentikUserUUID==proof.authentikUUID {
            if previous.OIDCSubject!=proof.subject || previous.AuthentikUserUUID!=proof.authentikUUID {
                return "",errors.New("identity replacement requires explicit Fabric migration")
            }
            // Idempotent: a repeated trusted login does not modify the ledger
            // or rotate a stable Fabric user ID.
            return previous.FabricUserID,nil
        }
    }
    if len(existing)>=10000 {return "",errors.New("Fabric human identity ledger is full")}
    seenIDs:=map[string]bool{}
    for _,old:=range existing {seenIDs[old.FabricUserID]=true}
    var allocated string
    for attempt:=0;attempt<8;attempt++ {
        allocated,err=allocateOpaqueFabricUserID()
        if err!=nil {return "",err}
        if !seenIDs[allocated] {break}
        allocated=""
    }
    if allocated=="" {return "",errors.New("Fabric user ID allocation collision")}
    claim.FabricUserID=allocated
    next:=append(existing,claim)
    if _,err:=authentik.NewIdentityRegistry(bundle.Name,bundle.Spec.TenantID,issuer,
        bundle.Spec.HumanAccess.Web.IAMGroups,next);err!=nil {
        return "",errors.New("new Fabric identity binding invalid")
    }
    raw,err:=json.Marshal(next)
    if err!=nil || len(raw)>maxIdentityBindingJSONBytes {
        return "",errors.New("Fabric human identity ledger capacity exceeded")
    }
    // Kubernetes performs optimistic concurrency control via resourceVersion.
    cm.Data[identityBindingsKey]=string(raw)
    if err:=r.Update(ctx,&cm);err!=nil {
        return "",fmt.Errorf("Fabric human identity enrollment CAS write failed: %w",err)
    }
    fresh,err:=r.readOpenFGAHumanIdentityRegistry(ctx,bundle)
    if err!=nil {return "",fmt.Errorf("Fabric human identity enrollment unconfirmed: %w",err)}
    confirmed,err:=fresh.ResolveOIDCSubject(issuer,proof.subject)
    if err!=nil || confirmed!=allocated {
        return "",errors.New("Fabric human enrollment binding changed after write")
    }
    after,err:=source.SnapshotUserAccount(ctx,proof.authentikUUID)
    if err!=nil || after.UUID!=proof.authentikUUID || !after.Active {
        // A persisted association alone is never a grant. The future BFF
        // must always recheck live active status + fresh FGA authorization.
        return "",errors.New("Authentik user disabled or changed during enrollment")
    }
    return allocated,nil
}
