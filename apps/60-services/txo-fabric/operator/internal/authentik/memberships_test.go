package authentik

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request)(*http.Response,error)
func (f roundTripFunc) RoundTrip(r *http.Request)(*http.Response,error){return f(r)}

const (
	groupName = "txo-fabric-indiba-sales"
	groupUUID = "550e8400-e29b-41d4-a716-446655440000"
	userOne = "550e8400-e29b-41d4-a716-446655440001"
	userTwo = "550e8400-e29b-41d4-a716-446655440002"
)
func apiResponse(status int, payload any)*http.Response {
	raw,_:=json.Marshal(payload)
	return &http.Response{StatusCode:status,Body:io.NopCloser(strings.NewReader(string(raw))),Header:make(http.Header)}
}
func apiPage(current,total,next,count int, values ...any) map[string]any {
	if values==nil {values=[]any{}}
	return map[string]any{
		"pagination":map[string]any{"current":current,"total_pages":total,"next":next,"count":count},
		"results":values,
	}
}
func groupObj()map[string]any{return map[string]any{"name":groupName,"pk":groupUUID}}
func userObj(pk int,uuid string,active bool)map[string]any{
	return map[string]any{"pk":pk,"uuid":uuid,"is_active":active,"groups":[]string{groupUUID},"username":"mutable-human-login"}
}
func clientWith(t *testing.T, fn roundTripFunc)*Client {
	t.Helper()
	c,err:=NewClient("fake-authentik-api-credential",fn)
	if err!=nil {t.Fatal(err)}
	return c
}

func TestSnapshotGroupMembershipCompleteActiveMembersOnly(t *testing.T) {
	var groups,users []int
	client:=clientWith(t,func(req *http.Request)(*http.Response,error) {
		if req.URL.Host!="authentik.auth.svc:9000" || req.URL.Scheme!="http" ||
			req.Header.Get("Authorization")!="Bearer fake-authentik-api-credential" ||
			req.Method!="GET" {
			t.Fatal("untrusted Authentik HTTP request")
		}
		if req.URL.Query().Get("page_size")!="100" {t.Fatal("missing bounded pagination")}
		page:=req.URL.Query().Get("page")
		switch req.URL.Path {
		case "/api/v3/core/groups/":
			if req.URL.Query().Get("name")!=groupName ||
				req.URL.Query().Get("include_users")!="false" || page!="1" {
				t.Fatalf("unexpected group lookup %s",req.URL)
			}
			groups=append(groups,1)
			return apiResponse(200,apiPage(1,1,0,1,groupObj())),nil
		case "/api/v3/core/users/":
			if req.URL.Query().Get("groups_by_pk")!=groupUUID ||
				req.URL.Query().Get("include_groups")!="true" {
				t.Fatalf("users not scoped to group %s",req.URL)
			}
			switch page {
			case "1":
				users=append(users,1)
				return apiResponse(200,apiPage(1,2,2,2,userObj(1,userOne,true))),nil
			case "2":
				users=append(users,2)
				return apiResponse(200,apiPage(2,2,0,2,userObj(2,userTwo,false))),nil
			}
		}
		t.Fatalf("untrusted Authentik endpoint %s",req.URL)
		return nil,errors.New("unexpected request")
	})
	out,err:=client.SnapshotGroupMembership(context.Background(),groupName)
	if err!=nil {t.Fatal(err)}
	if out.Name!=groupName || out.AuthentikGroupUUID!=groupUUID ||
		!reflect.DeepEqual(out.ActiveAuthentikUserUUIDs,[]string{userOne}) {
		t.Fatalf("incomplete or unsafe group snapshot: %#v",out)
	}
	if !reflect.DeepEqual(groups,[]int{1,1}) || !reflect.DeepEqual(users,[]int{1,2,1,2}) {
		t.Fatalf("membership was not read twice and paginated completely: %v %v",groups,users)
	}
}

func TestSnapshotGroupMembershipChangedBetweenReadsDenies(t *testing.T) {
	usersCalls:=0
	c:=clientWith(t,func(req *http.Request)(*http.Response,error) {
		if req.URL.Path=="/api/v3/core/groups/" {
			return apiResponse(200,apiPage(1,1,0,1,groupObj())),nil
		}
		usersCalls++
		if usersCalls==1 {return apiResponse(200,apiPage(1,1,0,1,userObj(1,userOne,true))),nil}
		return apiResponse(200,apiPage(1,1,0,1,userObj(2,userTwo,true))),nil
	})
	if _,err:=c.SnapshotGroupMembership(context.Background(),groupName);err==nil {
		t.Fatal("accepted a changing membership set")
	}
}

func TestSnapshotGroupMembershipEmptyGroupIsNotMissingGroup(t *testing.T) {
	c:=clientWith(t,func(req *http.Request)(*http.Response,error) {
		if req.URL.Path=="/api/v3/core/groups/" {
			return apiResponse(200,apiPage(1,1,0,1,groupObj())),nil
		}
		// Django Paginator may represent empty results as one empty page.
		return apiResponse(200,apiPage(1,1,0,0)),nil
	})
	out,err:=c.SnapshotGroupMembership(context.Background(),groupName)
	if err!=nil || out.AuthentikGroupUUID!=groupUUID || len(out.ActiveAuthentikUserUUIDs)!=0 {
		t.Fatalf("empty but valid group rejected: %#v %v",out,err)
	}
}

func TestSnapshotGroupMembershipNeverTreatsBrokenDirectoryAsEmpty(t *testing.T) {
	for _,tc:=range []struct{
		name string
		group func(*http.Request)(*http.Response,error)
		user func(*http.Request)(*http.Response,error)
	}{
		{"missing-group",func(*http.Request)(*http.Response,error){return apiResponse(200,apiPage(1,1,0,0)),nil},nil},
		{"duplicate-group",func(*http.Request)(*http.Response,error){return apiResponse(200,apiPage(1,1,0,2,groupObj(),groupObj())),nil},nil},
		{"bad-group-uuid",func(*http.Request)(*http.Response,error){return apiResponse(200,apiPage(1,1,0,1,map[string]any{"name":groupName,"pk":"tampered"})),nil},nil},
		{"unauthorized",func(*http.Request)(*http.Response,error){return apiResponse(403,map[string]any{"secret":"never-log"}),nil},nil},
		{"network-error",func(*http.Request)(*http.Response,error){return nil,errors.New("network outage")},nil},
		{"page-skipped",nil,func(*http.Request)(*http.Response,error){return apiResponse(200,apiPage(1,2,0,2,userObj(1,userOne,true))),nil}},
		{"wrong-user-group",nil,func(*http.Request)(*http.Response,error){
			u:=userObj(1,userOne,true);u["groups"]=[]string{"550e8400-e29b-41d4-a716-446655449999"}
			return apiResponse(200,apiPage(1,1,0,1,u)),nil
		}},
		{"missing-membership",nil,func(*http.Request)(*http.Response,error){
			u:=userObj(1,userOne,true);delete(u,"groups")
			return apiResponse(200,apiPage(1,1,0,1,u)),nil
		}},
		{"duplicate-user",nil,func(*http.Request)(*http.Response,error){
			return apiResponse(200,apiPage(1,1,0,2,userObj(1,userOne,true),userObj(1,userOne,true))),nil
		}},
		{"missing-is-active",nil,func(*http.Request)(*http.Response,error){
			u:=userObj(1,userOne,true);delete(u,"is_active")
			return apiResponse(200,apiPage(1,1,0,1,u)),nil
		}},
	}{
		t.Run(tc.name,func(t *testing.T) {
			c:=clientWith(t,func(req *http.Request)(*http.Response,error) {
				if req.URL.Path=="/api/v3/core/groups/" {
					if tc.group!=nil {return tc.group(req)}
					return apiResponse(200,apiPage(1,1,0,1,groupObj())),nil
				}
				if tc.user!=nil {return tc.user(req)}
				return apiResponse(200,apiPage(1,1,0,1,userObj(1,userOne,true))),nil
			})
			if _,err:=c.SnapshotGroupMembership(context.Background(),groupName);err==nil {
				t.Fatal("accepted unavailable, partial or corrupted human IAM state")
			}
		})
	}
}

func TestAuthentikCredentialAndGroupNameAreNeverClientControlled(t *testing.T) {
	for _,token:=range []string{"","password","evil\nHeader: x","   fake-authentik-api-credential"} {
		if _,err:=NewClient(token,nil);err==nil {t.Fatalf("accepted malformed token %q",token)}
	}
	calls:=0
	c:=clientWith(t,func(*http.Request)(*http.Response,error){calls++;return apiResponse(200,nil),nil})
	for _,name:=range []string{"admin","txo-fabric-indiba-sales/../groups",
		"txo-fabric-Indiba-sales","txo-fabric-","txo-fabric-indiba-sales?admin=true"} {
		if _,err:=c.SnapshotGroupMembership(context.Background(),name);err==nil {
			t.Fatalf("accepted tenant-provided group name %q",name)
		}
	}
	if calls!=0 {t.Fatal("untrusted group name triggered privileged Authentik API request")}
}
