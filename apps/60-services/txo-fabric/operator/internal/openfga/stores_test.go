package openfga

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
)

const (
	testStoreA = "01H0H015178Y2V4CX10C2KGHF4"
	testStoreB = "01H0H015178Y2V4CX10C2KGHF5"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func response(code int, body interface{}) *http.Response {
	out, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: code,
		Body: io.NopCloser(strings.NewReader(string(out))),
		Header: make(http.Header),
	}
}

func testClient(t *testing.T, fn roundTripFunc) *Client {
	t.Helper()
	c, err := NewClient("fake-but-long-enough-preshared-key", fn)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestTenantStoreNameStableAcrossUserFacingRename(t *testing.T) {
	a, err := TenantStoreName("TEN00001")
	if err != nil {
		t.Fatal(err)
	}
	b, err := TenantStoreName("TEN00002")
	if err != nil {
		t.Fatal(err)
	}
	if a != "txo-fabric-tenant-ten00001" || b != "txo-fabric-tenant-ten00002" || a == b {
		t.Fatalf("tenant store names are not isolated: %q / %q", a, b)
	}
	for _, bad := range []string{"hairem", "TEN", "ten00001", "TEN0000", "TEN00001/foo", "TEN00001\n"} {
		if _, err := TenantStoreName(bad); err == nil {
			t.Errorf("accepted invalid tenant identity %q", bad)
		}
	}
}

func TestEnsureTenantStoreIdempotentAndScoped(t *testing.T) {
	var mu sync.Mutex
	stores := map[string]string{}
	creates := 0
	c := testClient(t, func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		if req.URL.Host != "txo-fabric-openfga.txo-fabric-system.svc:8080" ||
			req.URL.Scheme != "http" ||
			req.Header.Get("Authorization") != "Bearer fake-but-long-enough-preshared-key" {
			t.Errorf("untrusted OpenFGA endpoint or missing service credential")
			return response(403, nil), nil
		}
		switch {
		case req.Method == "GET" && req.URL.Path == "/stores":
			name := req.URL.Query().Get("name")
			items := []Store{}
			if id := stores[name]; id != "" {
				items = append(items, Store{ID: id, Name: name})
			}
			return response(200, listStoresResponse{Stores: items}), nil
		case req.Method == "POST" && req.URL.Path == "/stores":
			var arg struct { Name string `json:"name"` }
			if err := json.NewDecoder(req.Body).Decode(&arg); err != nil {
				t.Error(err)
				return response(400, nil), nil
			}
			creates++
			id := testStoreA
			if arg.Name == "txo-fabric-tenant-ten00002" { id = testStoreB }
			stores[arg.Name] = id
			return response(201, Store{ID: id, Name: arg.Name}), nil
		}
		return response(404, nil), nil
	})
	ctx := context.Background()
	first, err := c.EnsureTenantStore(ctx, "TEN00001")
	if err != nil || first.ID != testStoreA { t.Fatalf("first tenant: %#v %v", first, err) }
	again, err := c.EnsureTenantStore(ctx, "TEN00001")
	if err != nil || again != first { t.Fatalf("duplicate reconciliation: %#v %v", again, err) }
	second, err := c.EnsureTenantStore(ctx, "TEN00002")
	if err != nil || second.ID != testStoreB || second.ID == first.ID {
		t.Fatalf("tenant isolation: %#v %v", second, err)
	}
	if creates != 2 { t.Fatalf("created %d stores, expected exactly two", creates) }
	if got, err := c.ResolveTenantStore(ctx, "TEN00001"); err != nil || got != first {
		t.Fatalf("trusted resolver: %#v %v", got, err)
	}
	if _, err := c.ResolveTenantStore(ctx, "TEN00003"); err == nil {
		t.Fatal("missing tenant store must deny, not create")
	}
	if creates != 2 { t.Fatal("read-only resolver wrote a store") }
}

func TestEnsureTenantStoreRejectsDuplicateAndMalformedStores(t *testing.T) {
	name := "txo-fabric-tenant-ten00001"
	for _, tt := range []struct {
		name string
		list []Store
	}{
		{"duplicate", []Store{{ID:testStoreA,Name:name},{ID:testStoreB,Name:name}}},
		{"malformed-id", []Store{{ID:"not-an-ulid",Name:name}}},
	} {
		t.Run(tt.name,func(t *testing.T) {
			post := false
			c := testClient(t,func(req *http.Request)(*http.Response,error){
				if req.Method=="POST" { post=true;return response(201,Store{}),nil }
				return response(200,listStoresResponse{Stores:tt.list}),nil
			})
			if _,err:=c.EnsureTenantStore(context.Background(),"TEN00001");err==nil {
				t.Fatal("ambiguous tenant store was accepted")
			}
			if post { t.Fatal("ambiguous store triggered unintended creation") }
		})
	}
}

func TestTenantStoreDiscoveryPaginationAndCursorLoop(t *testing.T) {
	name := "txo-fabric-tenant-ten00001"
	t.Run("second-page", func(t *testing.T) {
		c := testClient(t,func(r *http.Request)(*http.Response,error){
			if r.Method != "GET" { t.Fatal("unexpected POST") }
			if r.URL.Query().Get("continuation_token")=="" {
				return response(200,listStoresResponse{Stores:[]Store{},ContinuationToken:"next"}),nil
			}
			return response(200,listStoresResponse{Stores:[]Store{{ID:testStoreA,Name:name}}}),nil
		})
		s,err:=c.ResolveTenantStore(context.Background(),"TEN00001")
		if err!=nil||s.ID!=testStoreA { t.Fatalf("%#v %v",s,err) }
	})
	t.Run("duplicate-across-pages",func(t *testing.T){
		c:=testClient(t,func(r *http.Request)(*http.Response,error){
			if r.URL.Query().Get("continuation_token")=="" {
				return response(200,listStoresResponse{Stores:[]Store{{ID:testStoreA,Name:name}},ContinuationToken:"next"}),nil
			}
			return response(200,listStoresResponse{Stores:[]Store{{ID:testStoreB,Name:name}}}),nil
		})
		if _,err:=c.ResolveTenantStore(context.Background(),"TEN00001");err==nil {t.Fatal("duplicate not detected")}
	})
	t.Run("cursor-loop",func(t *testing.T){
		c:=testClient(t,func(*http.Request)(*http.Response,error){
			return response(200,listStoresResponse{ContinuationToken:"loop"}),nil
		})
		if _,err:=c.ResolveTenantStore(context.Background(),"TEN00001");err==nil {t.Fatal("cursor loop accepted")}
	})
}

func TestTenantStoreFailuresDoNotCreateOrGrant(t *testing.T) {
	for _, status := range []int{401,403,429,500,503} {
		t.Run(http.StatusText(status),func(t *testing.T){
			c:=testClient(t,func(*http.Request)(*http.Response,error){return response(status,map[string]string{"token":"never-log"}),nil})
			if _,err:=c.EnsureTenantStore(context.Background(),"TEN00001");err==nil {
				t.Fatal("upstream authorization/outage was not denied")
			} else if strings.Contains(err.Error(),"never-log") { t.Fatal("upstream body leaked in error") }
		})
	}
	c:=testClient(t,func(*http.Request)(*http.Response,error){return nil,errors.New("disconnected")})
	if _,err:=c.ResolveTenantStore(context.Background(),"TEN00001");err==nil {t.Fatal("outage allowed")}
	if _,err:=NewClient("short",nil);err==nil {t.Fatal("weak credential accepted")}
	if _,err:=NewClient("long-enough-credential\r\nInjected: yes",nil);err==nil {t.Fatal("header injection accepted")}
}

func TestCreateResponseMustMatchCanonicalTenantAndBeDiscoverable(t *testing.T) {
	for _, created := range []Store{
		{ID:testStoreA,Name:"foreign-tenant"},
		{ID:"invalid",Name:"txo-fabric-tenant-ten00001"},
	} {
		c:=testClient(t,func(r *http.Request)(*http.Response,error){
			if r.Method=="POST" {return response(201,created),nil}
			return response(200,listStoresResponse{}),nil
		})
		if _,err:=c.EnsureTenantStore(context.Background(),"TEN00001");err==nil {
			t.Fatal("invalid store creation accepted")
		}
	}
	c:=testClient(t,func(r *http.Request)(*http.Response,error){
		if r.Method=="POST" {return response(201,Store{ID:testStoreA,Name:"txo-fabric-tenant-ten00001"}),nil}
		return response(200,listStoresResponse{}),nil
	})
	if _,err:=c.EnsureTenantStore(context.Background(),"TEN00001");err==nil {
		t.Fatal("non-observable creation accepted")
	}
}

func TestStoreFilterCannotBeControlledFromRequest(t *testing.T) {
	c:=testClient(t,func(r *http.Request)(*http.Response,error){
		v,_:=url.ParseQuery(r.URL.RawQuery)
		if v.Get("name")!="txo-fabric-tenant-ten00001" {
			t.Fatalf("unexpected store name %q",v.Get("name"))
		}
		return response(200,listStoresResponse{Stores:[]Store{{ID:testStoreB,Name:"txo-fabric-tenant-ten00002"}}}),nil
	})
	if _,err:=c.ResolveTenantStore(context.Background(),"TEN00001");err==nil {
		t.Fatal("tenant B store was returned to tenant A")
	}
}
