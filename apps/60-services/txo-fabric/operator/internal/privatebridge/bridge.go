// Package privatebridge provides the narrow BFF-to-Fabric control-plane
// identity resolution endpoint. It never enrolls users or authorizes chat.
package privatebridge

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	Path = "/internal/v1/identity/resolve"
	MaxAssertionAge = 30 * time.Second
	ClockSkew = 5 * time.Second
	MaxWorkloadTokenAge = time.Hour
)

// Contract is fixed by the platform, never supplied by the browser.
type Contract struct {
	Audience string
	KubernetesIssuer string
	BFFServiceAccount string // system:serviceaccount:<namespace>:<name>
	AssertionKey []byte
	Now func() time.Time
}

type Workload struct {
	Authenticated bool
	Username string
	Audiences []string
}

// TokenReviewer must execute a Kubernetes TokenReview with exactly the
// requested audience. UserInfo is trusted only from a successful TokenReview.
type TokenReviewer interface {
	Review(context.Context, string, string) (Workload, error)
}

// NonceStore must create a shared durable record using atomic create-only
// semantics. An existing record, even if expired, MUST NOT be reusable.
type NonceStore interface {
	Consume(context.Context, string, time.Time) error
}

// Resolver uses only operator-owned tenant/catalog/store/model/identity
// bindings and a live, bounded authorization condition. Never a client ID.
type Resolver interface {
	Resolve(context.Context, string, string, time.Time) (string, error)
}

type Handler struct {
	Contract Contract
	Tokens TokenReviewer
	Nonces NonceStore
	Identities Resolver
}

type Request struct {
	TenantKey string `json:"tenantKey"`
	AuthentikUserUUID string `json:"authentikUserUUID"`
}

type Assertion struct {
	Method string `json:"method"`
	Path string `json:"path"`
	TenantKey string `json:"tenantKey"`
	AuthentikUserUUID string `json:"authentikUserUUID"`
	ServiceAccount string `json:"serviceAccount"`
	Audience string `json:"audience"`
	IssuedAt int64 `json:"iat"`
	ExpiresAt int64 `json:"exp"`
	JTI string `json:"jti"`
}

type Resolution struct {
	TenantKey string `json:"tenantKey"`
	FabricUserID string `json:"fabricUserId"`
}

var (
	tenantKeyPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	uuidPattern = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[1-8][a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)
	jtiPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{22,128}$`)
	errDenied = errors.New("private Fabric identity request denied")
)

func (h *Handler) clock() time.Time {
	if h.Contract.Now != nil { return h.Contract.Now().UTC() }
	return time.Now().UTC()
}

// SignAssertion defines deterministic HMAC-SHA256 wire encoding for a BFF
// client implementation. Neither assertion payload nor key is logged.
func SignAssertion(key []byte, a Assertion) (string, error) {
	if len(key) < 32 { return "", errDenied }
	data, err := json.Marshal(a)
	if err != nil { return "", errDenied }
	payload := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(payload))
	return payload+"."+base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func verifyAssertion(key []byte, raw string) (Assertion, error) {
	var a Assertion
	if len(key)<32 || len(raw)>4096 { return a, errDenied }
	p := strings.Split(raw, ".")
	if len(p)!=2 || len(p[1])!=43 { return a, errDenied }
	provided, err := base64.RawURLEncoding.DecodeString(p[1])
	if err!=nil || len(provided)!=sha256.Size { return a, errDenied }
	mac:=hmac.New(sha256.New,key)
	_,_ = mac.Write([]byte(p[0]))
	if !hmac.Equal(provided,mac.Sum(nil)) { return a, errDenied }
	payload,err:=base64.RawURLEncoding.DecodeString(p[0])
	if err!=nil || len(payload)>2048 { return a,errDenied }
	d:=json.NewDecoder(strings.NewReader(string(payload)))
	d.DisallowUnknownFields()
	if d.Decode(&a)!=nil || d.Decode(new(any))!=io.EOF { return Assertion{},errDenied }
	return a,nil
}

func exactlyOne(value []string, expected string) bool {
	return len(value)==1 && value[0]==expected && expected!=""
}

// TokenReview authenticates the signed JWT, but does not return its issuer
// in UserInfo. Inspect its bound issuer/subject/audience claims ONLY AFTER
// a successful TokenReview, never as a substitute for that API verdict.
func tokenClaimsMatch(jwt, issuer, audience, subject string, now time.Time) bool {
	p:=strings.Split(jwt, ".")
	if len(p)!=3 || len(jwt)>16384 { return false }
	raw,err:=base64.RawURLEncoding.DecodeString(p[1])
	if err!=nil || len(raw)>4096 {return false}
	var claims struct {
		Issuer string `json:"iss"`
		Subject string `json:"sub"`
		Audience json.RawMessage `json:"aud"`
		IssuedAt int64 `json:"iat"`
		ExpiresAt int64 `json:"exp"`
		NotBefore int64 `json:"nbf"`
	}
	if json.Unmarshal(raw,&claims)!=nil || claims.Issuer!=issuer || claims.Subject!=subject {return false}
	issued:=time.Unix(claims.IssuedAt,0)
	expires:=time.Unix(claims.ExpiresAt,0)
	if claims.IssuedAt<=0 || claims.ExpiresAt<=0 || !expires.After(issued) ||
		expires.Sub(issued)>MaxWorkloadTokenAge ||
		now.Before(issued.Add(-ClockSkew)) || !now.Before(expires) ||
		(claims.NotBefore!=0 && now.Before(time.Unix(claims.NotBefore,0).Add(-ClockSkew))) {
		return false
	}
	var single string
	if json.Unmarshal(claims.Audience,&single)==nil {return single==audience}
	var many []string
	return json.Unmarshal(claims.Audience,&many)==nil && exactlyOne(many,audience)
}

func validAssertion(a Assertion, q Request, c Contract, now time.Time) bool {
	if a.Method!=http.MethodPost || a.Path!=Path ||
		a.TenantKey!=q.TenantKey || a.AuthentikUserUUID!=q.AuthentikUserUUID ||
		a.ServiceAccount!=c.BFFServiceAccount || a.Audience!=c.Audience ||
		!tenantKeyPattern.MatchString(a.TenantKey) || !uuidPattern.MatchString(a.AuthentikUserUUID) ||
		!jtiPattern.MatchString(a.JTI) {return false}
	issued:=time.Unix(a.IssuedAt,0)
	expires:=time.Unix(a.ExpiresAt,0)
	return a.IssuedAt>0 && a.ExpiresAt>0 && expires.After(issued) &&
		expires.Sub(issued)<=MaxAssertionAge &&
		!now.Before(issued.Add(-ClockSkew)) &&
		now.Before(expires) && !now.After(issued.Add(MaxAssertionAge+ClockSkew))
}

func scopedNonceKey(c Contract, a Assertion) string {
	// No raw user identifier or assertion in durable Kubernetes metadata.
	scope:=c.KubernetesIssuer+"\x00"+c.Audience+"\x00"+c.BFFServiceAccount+
		"\x00"+a.TenantKey+"\x00"+a.JTI
	sum:=sha256.Sum256([]byte(scope))
	return "bff-jti-"+hex.EncodeToString(sum[:24])
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	deny:=func(){ w.Header().Set("Cache-Control","no-store"); http.Error(w,"identity unavailable",http.StatusForbidden) }
	if h==nil || r==nil || h.Tokens==nil || h.Nonces==nil || h.Identities==nil ||
		h.Contract.Audience=="" || h.Contract.KubernetesIssuer=="" ||
		!strings.HasPrefix(h.Contract.BFFServiceAccount,"system:serviceaccount:") ||
		len(h.Contract.AssertionKey)<32 || r.Method!=http.MethodPost ||
		r.URL.Path!=Path || r.URL.RawQuery!="" { deny();return }
	if ct:=r.Header.Get("Content-Type");ct!="application/json" { deny();return }
	rawToken:=r.Header.Get("Authorization")
	if len(rawToken)>16391 || !strings.HasPrefix(rawToken,"Bearer ") ||
		strings.TrimSpace(rawToken[7:])!=rawToken[7:] ||
		rawToken[7:]=="" {deny();return}
	var q Request
	decoder:=json.NewDecoder(http.MaxBytesReader(w,r.Body,4096))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&q)!=nil || decoder.Decode(new(any))!=io.EOF ||
		!tenantKeyPattern.MatchString(q.TenantKey) || !uuidPattern.MatchString(q.AuthentikUserUUID) { deny();return }
	a,err:=verifyAssertion(h.Contract.AssertionKey,r.Header.Get("X-Fabric-Assertion"))
	now:=h.clock()
	if err!=nil || !validAssertion(a,q,h.Contract,now) { deny();return }
	review,err:=h.Tokens.Review(r.Context(),rawToken[7:],h.Contract.Audience)
	if err!=nil || !review.Authenticated || review.Username!=h.Contract.BFFServiceAccount ||
		!exactlyOne(review.Audiences,h.Contract.Audience) ||
		!tokenClaimsMatch(rawToken[7:],h.Contract.KubernetesIssuer,
			h.Contract.Audience,h.Contract.BFFServiceAccount,now) { deny();return }
	// CAS create in shared Kubernetes storage: a concurrent replica or a
	// repeated request cannot return identity a second time. Consumed even
	// when downstream readiness fails; clients must mint a NEW assertion.
	if h.Nonces.Consume(r.Context(),scopedNonceKey(h.Contract,a),
		time.Unix(a.ExpiresAt,0).Add(ClockSkew))!=nil {deny();return}
	userID,err:=h.Identities.Resolve(r.Context(),q.TenantKey,q.AuthentikUserUUID,now)
	if err!=nil || userID=="" {deny();return}
	w.Header().Set("Cache-Control","no-store")
	w.Header().Set("Content-Type","application/json")
	_ = json.NewEncoder(w).Encode(Resolution{TenantKey:q.TenantKey,FabricUserID:userID})
}
