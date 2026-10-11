package privatebridge

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	authv1 "k8s.io/api/authentication/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestKubernetesTokenReviewerRequiresAuthenticatedExactAudience(t *testing.T) {
	kube:=k8sfake.NewSimpleClientset()
	count:=0
	kube.PrependReactor("create","tokenreviews",func(action ktesting.Action)(bool,runtime.Object,error){
		count++
		req,ok:=action.(ktesting.CreateAction).GetObject().(*authv1.TokenReview)
		if !ok || req.Spec.Token!="bound-bff-token" ||
			!exactlyOne(req.Spec.Audiences,testAudience) {
			t.Fatal("unbound or overbroad Kubernetes TokenReview")
		}
		return true,&authv1.TokenReview{Status:authv1.TokenReviewStatus{
			Authenticated:true,
			User:authv1.UserInfo{Username:testSA},
			Audiences:[]string{testAudience},
		}},nil
	})
	review,err:= (KubernetesTokenReviewer{Client:kube}).Review(context.Background(),"bound-bff-token",testAudience)
	if err!=nil || !review.Authenticated || review.Username!=testSA ||
		!exactlyOne(review.Audiences,testAudience) || count!=1 {
		t.Fatal("TokenReview result not enforced")
	}
	kube.PrependReactor("create","tokenreviews",func(ktesting.Action)(bool,runtime.Object,error){
		return true,&authv1.TokenReview{Status:authv1.TokenReviewStatus{Authenticated:false}},nil
	})
	if _,err:=(KubernetesTokenReviewer{Client:kube}).Review(context.Background(),"bound-bff-token",testAudience);err==nil {
		t.Fatal("Kubernetes unauthenticated review accepted")
	}
	kube.PrependReactor("create","tokenreviews",func(ktesting.Action)(bool,runtime.Object,error){
		return true,nil,errors.New("TokenReview endpoint offline")
	})
	if _,err:=(KubernetesTokenReviewer{Client:kube}).Review(context.Background(),"bound-bff-token",testAudience);err==nil {
		t.Fatal("Kubernetes TokenReview outage accepted")
	}
}

func TestKubernetesNonceLeasePersistsAndRejectsReplayedCreate(t *testing.T) {
	scheme:=runtime.NewScheme()
	if err:=coordinationv1.AddToScheme(scheme);err!=nil {t.Fatal(err)}
	c:=fake.NewClientBuilder().WithScheme(scheme).Build()
	store:=KubernetesNonceStore{Client:c}
	key:="bff-jti-"+strings.Repeat("a",48)
	expires:=time.Date(2026,10,11,20,1,0,0,time.UTC)
	if err:=store.Consume(context.Background(),key,expires);err!=nil {t.Fatal(err)}
	if !store.StableSnapshot(context.Background(),key) {t.Fatal("nonce not shared in durable API state")}
	if err:=store.Consume(context.Background(),key,expires);err==nil {
		t.Fatal("reused nonce reaccepted despite existing lease")
	}
	for _,bad:=range []string{"","bff-jti-"+strings.Repeat("g",48),"another-"+strings.Repeat("a",48)} {
		if err:=store.Consume(context.Background(),bad,expires);err==nil {t.Fatal("non-DNS-safe or unexpected lease key accepted")}
	}
	var lease coordinationv1.Lease
	if err:=c.Get(context.Background(),types.NamespacedName{
		Name:key,Namespace:NonceNamespace,
	},&lease);err!=nil {t.Fatal(err)}
	if lease.Spec.HolderIdentity==nil || *lease.Spec.HolderIdentity!="consumed" ||
		lease.Annotations["fabric.truxonline.io/nonce-valid-until"]!=expires.Format(time.RFC3339) {
		t.Fatal("nonce lease not durable / wrong non-secret retention metadata")
	}
}
