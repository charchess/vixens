package privatebridge

import (
	"context"
	"errors"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"k8s.io/client-go/kubernetes"
)

const NonceNamespace = "txo-fabric-system"

// KubernetesTokenReviewer uses the real authentication.k8s.io TokenReview.
// It never trusts an unauthenticated bearer claim or the request's X-User.
type KubernetesTokenReviewer struct { Client kubernetes.Interface }

func (k KubernetesTokenReviewer) Review(ctx context.Context, bearer, audience string) (Workload,error) {
	if k.Client==nil || bearer=="" || audience=="" {return Workload{},errDenied}
	reply,err:=k.Client.AuthenticationV1().TokenReviews().Create(ctx,
		&authenticationv1.TokenReview{Spec:authenticationv1.TokenReviewSpec{
			Token:bearer,Audiences:[]string{audience},
		}},metav1.CreateOptions{})
	if err!=nil || reply.Status.Error!="" || !reply.Status.Authenticated {
		return Workload{},errDenied
	}
	return Workload{
		Authenticated:true,
		Username:reply.Status.User.Username,
		Audiences:reply.Status.Audiences,
	},nil
}

// KubernetesNonceStore persists a one-time JTI in the operator's namespace.
// K8s Lease Create is an atomic CAS across operator/BFF replicas. On Already
// Exists, API outage or uncertainty we DENY; never Delete/reuse a nonce.
// Records are metadata-only and include no token, UUID, subject or raw JTI.
// Garbage collection may remove a record only past the maximal issuance and
// skew horizon via a separately reviewed janitor, not inside authorization.
type KubernetesNonceStore struct { Client client.Client }

func (s KubernetesNonceStore) Consume(ctx context.Context, opaqueKey string, expires time.Time) error {
	if s.Client==nil || len(opaqueKey)!=56 || expires.IsZero() ||
		len(opaqueKey)<8 || opaqueKey[:8]!="bff-jti-" {return errDenied}
	for _,char:=range opaqueKey[8:] {
		if (char<'0'||char>'9') && (char<'a'||char>'f') {return errDenied}
	}
	seconds:=int32(MaxAssertionAge.Seconds()+ClockSkew.Seconds()+60)
	holder:="consumed"
	record:=&coordinationv1.Lease{
		ObjectMeta:metav1.ObjectMeta{
			Namespace:NonceNamespace,
			Name:opaqueKey,
			Labels:map[string]string{
				"app.kubernetes.io/part-of":"txo-fabric",
				"app.kubernetes.io/component":"bff-replay-proof",
			},
			Annotations:map[string]string{
				"fabric.truxonline.io/nonce-valid-until":expires.UTC().Format(time.RFC3339),
			},
		},
		Spec:coordinationv1.LeaseSpec{
			HolderIdentity:&holder,
			LeaseDurationSeconds:&seconds,
		},
	}
	if err:=s.Client.Create(ctx,record);err!=nil {return errors.New("private identity replay protection unavailable")}
	// A successful API acknowledgement is the durable shared CAS proof.
	// Never treat a timed-out create as success (its retry must also deny).
	return nil
}

// StableSnapshot is used only for evidence in tests; a nonce is never
// retrieved to decide whether a repeated assertion can be accepted.
func (s KubernetesNonceStore) StableSnapshot(ctx context.Context, key string) bool {
	if s.Client==nil {return false}
	var record coordinationv1.Lease
	return s.Client.Get(ctx,types.NamespacedName{Namespace:NonceNamespace,Name:key},&record)==nil
}
