package privatebridge

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testUUID="550e8400-e29b-41d4-a716-446655440000"
	testSA="system:serviceaccount:txo-fabric-system:txo-fabric-webui-bff"
	testIssuer="https://kubernetes.default.svc"
	testAudience="txo-fabric-private-identity"
)

var testNow=time.Date(2026,10,11,20,0,0,0,time.UTC)

type tokenReviewerFunc func(context.Context,string,string)(Workload,error)
func (f tokenReviewerFunc) Review(ctx context.Context,token,audience string)(Workload,error) {return f(ctx,token,audience)}

type resolutionFunc func(context.Context,string,string,time.Time)(string,error)
func (f resolutionFunc) Resolve(ctx context.Context,tenant,uuid string,now time.Time)(string,error) {
	return f(ctx,tenant,uuid,now)
}

type memoryNonceStore struct {
	mu sync.Mutex
	used map[string]bool
	calls int
	fail bool
}
func (n *memoryNonceStore) Consume(_ context.Context,key string,_ time.Time) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.calls++
	if n.fail || n.used[key] {return errors.New("replay or outage")}
	n.used[key]=true
	return nil
}

func jwtFor(issuer,subject,audience string) string {
	head:=base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	p,_:=json.Marshal(map[string]any{"iss":issuer,"sub":subject,"aud":audience})
	return head+"."+base64.RawURLEncoding.EncodeToString(p)+".signature"
}

func proofFor(tenant,uuid,jti string) Assertion {
	return Assertion{
		Method:"POST",Path:Path,TenantKey:tenant,AuthentikUserUUID:uuid,
		ServiceAccount:testSA,Audience:testAudience,
		IssuedAt:testNow.Unix(),ExpiresAt:testNow.Add(20*time.Second).Unix(),JTI:jti,
	}
}

func fixture() (*Handler,*memoryNonceStore) {
	nonces:=&memoryNonceStore{used:map[string]bool{}}
	handler:=&Handler{
		Contract:Contract{
			Audience:testAudience,KubernetesIssuer:testIssuer,
			BFFServiceAccount:testSA,AssertionKey:[]byte(strings.Repeat("k",32)),
			Now:func()time.Time{return testNow},
		},
		Tokens:tokenReviewerFunc(func(_ context.Context,token,aud string)(Workload,error){
			if aud!=testAudience {return Workload{},errors.New("wrong audience")}
			return Workload{Authenticated:true,Username:testSA,Audiences:[]string{aud}},nil
		}),
		Nonces:nonces,
		Identities:resolutionFunc(func(_ context.Context,tenant,uuid string,_ time.Time)(string,error){
			if uuid!=testUUID {return "",errors.New("not found")}
			switch tenant {
			case "hairem":return "usr000001",nil
			case "indiba":return "usr000002",nil
			default:return "",errors.New("not found")
			}
		}),
	}
	return handler,nonces
}

func serve(t *testing.T,h *Handler,a Assertion,q Request,token string) *httptest.ResponseRecorder {
	t.Helper()
	raw,err:=json.Marshal(q)
	if err!=nil {t.Fatal(err)}
	proof,err:=SignAssertion(h.Contract.AssertionKey,a)
	if err!=nil {t.Fatal(err)}
	req:=httptest.NewRequest(http.MethodPost,Path,bytes.NewReader(raw))
	req.Header.Set("Content-Type","application/json")
	req.Header.Set("Authorization","Bearer "+token)
	req.Header.Set("X-Fabric-Assertion",proof)
	w:=httptest.NewRecorder()
	h.ServeHTTP(w,req)
	return w
}

func TestExactTenantLookupOnlyOnceAndDifferentFabricUsers(t *testing.T) {
	h,store:=fixture()
	tok:=jwtFor(testIssuer,testSA,testAudience)
	a:=proofFor("hairem",testUUID,"unique-proof-01234567890123456")
	w:=serve(t,h,a,Request{TenantKey:"hairem",AuthentikUserUUID:testUUID},tok)
	if w.Code!=200 || !strings.Contains(w.Body.String(),`"fabricUserId":"usr000001"`) {
		t.Fatalf("denied valid workload assertion: %d %s",w.Code,w.Body.String())
	}
	again:=serve(t,h,a,Request{TenantKey:"hairem",AuthentikUserUUID:testUUID},tok)
	if again.Code!=403 {t.Fatal("valid Kubernetes TokenReview bypassed one-time replay protection")}
	a2:=proofFor("indiba",testUUID,"unique-proof-01234567890123457")
	other:=serve(t,h,a2,Request{TenantKey:"indiba",AuthentikUserUUID:testUUID},tok)
	if other.Code!=200 || !strings.Contains(other.Body.String(),`"fabricUserId":"usr000002"`) ||
		strings.Contains(other.Body.String(),"usr000001") {t.Fatal("cross-tenant identity contamination")}
	if store.calls!=3 {t.Fatalf("missing atomic nonce consumption: %d",store.calls)}
	if w.Header().Get("Cache-Control")!="no-store" {t.Fatal("user binding response is cacheable")}
}

func TestPrivateBridgeRejectsSignatureContextTimeIdentityAndReviewForgery(t *testing.T) {
	normal:=proofFor("hairem",testUUID,"unique-proof-01234567890123456")
	tests:=[]struct{
		name string
		alter func(*Handler,*Assertion,*Request,*string)
	}{
		{"wrong-method",func(_ *Handler,a *Assertion,_ *Request,_ *string){a.Method="GET"}},
		{"wrong-endpoint",func(_ *Handler,a *Assertion,_ *Request,_ *string){a.Path="/healthz"}},
		{"wrong-tenant",func(_ *Handler,a *Assertion,_ *Request,_ *string){a.TenantKey="indiba"}},
		{"wrong-uuid",func(_ *Handler,a *Assertion,_ *Request,_ *string){a.AuthentikUserUUID="550e8400-e29b-41d4-a716-446655440001"}},
		{"wrong-audience",func(_ *Handler,a *Assertion,_ *Request,_ *string){a.Audience="kubernetes.default.svc"}},
		{"wrong-serviceaccount",func(_ *Handler,a *Assertion,_ *Request,_ *string){a.ServiceAccount="system:serviceaccount:txo-fabric-system:other"}},
		{"expired",func(_ *Handler,a *Assertion,_ *Request,_ *string){a.IssuedAt-=60;a.ExpiresAt-=60}},
		{"future",func(_ *Handler,a *Assertion,_ *Request,_ *string){a.IssuedAt+=60;a.ExpiresAt+=60}},
		{"overlong",func(_ *Handler,a *Assertion,_ *Request,_ *string){a.ExpiresAt+=31}},
		{"empty-jti",func(_ *Handler,a *Assertion,_ *Request,_ *string){a.JTI=""}},
		{"wrong-token-issuer",func(_ *Handler,_ *Assertion,_ *Request,t *string){*t=jwtFor("https://foreign.invalid",testSA,testAudience)}},
		{"wrong-token-subject",func(_ *Handler,_ *Assertion,_ *Request,t *string){*t=jwtFor(testIssuer,"system:serviceaccount:evil:bff",testAudience)}},
		{"wrong-token-audience",func(_ *Handler,_ *Assertion,_ *Request,t *string){*t=jwtFor(testIssuer,testSA,"foreign-client")}},
		{"review-unauthenticated",func(h *Handler,_ *Assertion,_ *Request,_ *string){h.Tokens=tokenReviewerFunc(func(context.Context,string,string)(Workload,error){return Workload{},nil})}},
		{"review-unavailable",func(h *Handler,_ *Assertion,_ *Request,_ *string){h.Tokens=tokenReviewerFunc(func(context.Context,string,string)(Workload,error){return Workload{},errors.New("offline")})}},
		{"review-foreign-SA",func(h *Handler,_ *Assertion,_ *Request,_ *string){h.Tokens=tokenReviewerFunc(func(context.Context,string,string)(Workload,error){return Workload{Authenticated:true,Username:"system:serviceaccount:foreign:bff",Audiences:[]string{testAudience}},nil})}},
		{"review-foreign-audience",func(h *Handler,_ *Assertion,_ *Request,_ *string){h.Tokens=tokenReviewerFunc(func(context.Context,string,string)(Workload,error){return Workload{Authenticated:true,Username:testSA,Audiences:[]string{"foreign"}},nil})}},
		{"nonce-store-unavailable",func(h *Handler,_ *Assertion,_ *Request,_ *string){h.Nonces.(*memoryNonceStore).fail=true}},
		{"ledger-or-freshness-unavailable",func(h *Handler,_ *Assertion,_ *Request,_ *string){h.Identities=resolutionFunc(func(context.Context,string,string,time.Time)(string,error){return "",errors.New("OpenFGA or ledger unavailable")})}},
	}
	for _,tc:=range tests{
		t.Run(tc.name,func(t *testing.T){
			h,_:=fixture()
			a:=normal
			q:=Request{TenantKey:"hairem",AuthentikUserUUID:testUUID}
			tok:=jwtFor(testIssuer,testSA,testAudience)
			tc.alter(h,&a,&q,&tok)
			w:=serve(t,h,a,q,tok)
			if w.Code!=403 || w.Body.String()!="identity unavailable\n" {
				t.Fatalf("denial leaks identity or accepted forgery: %d %q",w.Code,w.Body.String())
			}
		})
	}
}

func TestPrivateBridgeTamperedSignatureUnknownFieldsAndMethod(t *testing.T) {
	h,store:=fixture()
	a:=proofFor("hairem",testUUID,"unique-proof-01234567890123456")
	token:=jwtFor(testIssuer,testSA,testAudience)
	q:=Request{TenantKey:"hairem",AuthentikUserUUID:testUUID}
	for _,tc:=range []struct{
		name string
		alter func(*http.Request)
	}{
		{"wrong-http-method",func(r *http.Request){r.Method="GET"}},
		{"wrong-http-path",func(r *http.Request){r.URL.Path="/healthz"}},
		{"query",func(r *http.Request){r.URL.RawQuery="user=admin"}},
		{"browser-token",func(r *http.Request){r.Header.Set("Authorization","Bearer browser-session")}},
		{"spoofed-sig",func(r *http.Request){r.Header.Set("X-Fabric-Assertion","abc.def")}},
		{"unknown-field",func(r *http.Request){r.Body=ioNopCloser(`{"tenantKey":"hairem","authentikUserUUID":"`+testUUID+`","fabricUserId":"usr000001"}`)}},
	}{
		t.Run(tc.name,func(t *testing.T) {
			raw,_:=json.Marshal(q)
			signed,_:=SignAssertion(h.Contract.AssertionKey,a)
			req:=httptest.NewRequest("POST",Path,bytes.NewReader(raw))
			req.Header.Set("Content-Type","application/json")
			req.Header.Set("Authorization","Bearer "+token)
			req.Header.Set("X-Fabric-Assertion",signed)
			tc.alter(req)
			w:=httptest.NewRecorder()
			h.ServeHTTP(w,req)
			if w.Code!=403 || w.Body.String()!="identity unavailable\n" {t.Fatalf("unsafe access/oracle: %d %q",w.Code,w.Body.String())}
		})
	}
	if store.calls!=0 {t.Fatal("invalid requests consumed nonce or invoked resolver")}
}

type bodyReader struct { *strings.Reader }
func (bodyReader) Close() error{return nil}
func ioNopCloser(s string) bodyReader{return bodyReader{strings.NewReader(s)}}

func TestConcurrentReplayExactlyOneSuccess(t *testing.T) {
	h,store:=fixture()
	a:=proofFor("hairem",testUUID,"concurrent-proof-01234567890123")
	q:=Request{TenantKey:"hairem",AuthentikUserUUID:testUUID}
	tok:=jwtFor(testIssuer,testSA,testAudience)
	var accepted atomic.Int32
	var rejected atomic.Int32
	var wg sync.WaitGroup
	for i:=0;i<48;i++{
		wg.Add(1)
		go func(){
			defer wg.Done()
			w:=serve(t,h,a,q,tok)
			if w.Code==200 {accepted.Add(1)} else if w.Code==403 {rejected.Add(1)} else {t.Errorf("unexpected HTTP %d",w.Code)}
		}()
	}
	wg.Wait()
	if accepted.Load()!=1 || rejected.Load()!=47 || store.calls!=48 {
		t.Fatalf("non-atomic replay test: accepted=%d rejected=%d nonce=%d",accepted.Load(),rejected.Load(),store.calls)
	}
}

func TestNonceScopeSeparatesIssuerAudienceTenantAndNeverStoresUUID(t *testing.T) {
	h,_:=fixture()
	a:=proofFor("hairem",testUUID,"unique-proof-01234567890123456")
	base:=scopedNonceKey(h.Contract,a)
	if len(base)!=56 || !strings.HasPrefix(base,"bff-jti-") || strings.Contains(base,testUUID) {
		t.Fatal("nonce metadata not bounded opaque DNS safe key")
	}
	other:=a;other.TenantKey="indiba"
	if scopedNonceKey(h.Contract,other)==base {t.Fatal("tenant nonce collision")}
	c:=h.Contract;c.KubernetesIssuer="other"
	if scopedNonceKey(c,a)==base {t.Fatal("issuer nonce collision")}
	c=h.Contract;c.Audience="other"
	if scopedNonceKey(c,a)==base {t.Fatal("audience nonce collision")}
}
