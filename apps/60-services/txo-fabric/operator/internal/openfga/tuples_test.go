package openfga

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"io"
	"strings"
	"testing"
)

func TestReconcileTupleScopeRevokesMembersWithoutCrossTenantMutation(t *testing.T) {
	scope:=TupleScope{Object:"group:grp000001",Relation:"member"}
	another:=TupleScope{Object:"group:grp000002",Relation:"member"}
	ed:=Tuple{User:"user:usr000001",Relation:"member",Object:scope.Object}
	paul:=Tuple{User:"user:usr000002",Relation:"member",Object:scope.Object}
	other:=Tuple{User:"user:usr000003",Relation:"member",Object:another.Object}
	state:=map[string]map[Tuple]bool{
		testStoreA:{ed:true,paul:true,other:true},
		testStoreB:{ed:true},
	}
	var actions []string
	var lastWriteModel string
	posts:=0
	client:=testClient(t,func(req *http.Request)(*http.Response,error) {
		if req.URL.Host!="txo-fabric-openfga.txo-fabric-system.svc:8080" ||
			req.Header.Get("Authorization")!="Bearer fake-but-long-enough-preshared-key" {
			t.Fatal("tuple API called outside private Fabric service")
		}
		storeID:=testStoreA
		if strings.Contains(req.URL.Path,testStoreB) {storeID=testStoreB}
		if req.URL.Path=="/stores/"+storeID+"/authorization-models" && req.Method=="GET" {
			blob,_:=json.Marshal(modelWithID(t,sampleModelID))
			return response(200,authorizationModelResponse{AuthorizationModels:[]json.RawMessage{blob}}),nil
		}
		if req.URL.Path=="/stores/"+storeID+"/read" && req.Method=="POST" {
			var body tupleReadRequest
			if err:=json.NewDecoder(req.Body).Decode(&body);err!=nil {t.Fatal(err)}
			if body.PageSize!=100 || body.TupleKey["object"]!=scope.Object || body.TupleKey["relation"]!="member" {
				t.Fatalf("unscoped OpenFGA read: %+v",body)
			}
			var values []Tuple
			for v:=range state[storeID] {
				if v.Object==scope.Object && v.Relation==scope.Relation {values=append(values,v)}
			}
			sortTuples(values)
			var records []readTuple
			for _,key:=range values {records=append(records,readTuple{Key:key})}
			return response(200,readTuplesResponse{Tuples:records}),nil
		}
		if req.URL.Path=="/stores/"+storeID+"/write" && req.Method=="POST" {
			posts++
			var payload tupleWriteRequest
			if err:=json.NewDecoder(req.Body).Decode(&payload);err!=nil {t.Fatal(err)}
			if storeID!=testStoreA {t.Fatal("tuple writer crossed tenant store")}
			lastWriteModel=payload.AuthorizationModelID
			if payload.Deletes!=nil {
				if payload.Deletes.OnMissing!="ignore" {t.Fatal("delete isn't retry safe")}
				for _,key:=range payload.Deletes.TupleKeys {
					if key.Object!=scope.Object {t.Fatal("deleting foreign tuple")}
					actions=append(actions,"delete:"+key.User)
					delete(state[storeID],key)
				}
			}
			if payload.Writes!=nil {
				if payload.Writes.OnDuplicate!="ignore" {t.Fatal("write isn't retry safe")}
				for _,key:=range payload.Writes.TupleKeys {
					if key.Object!=scope.Object {t.Fatal("writing foreign tuple")}
					actions=append(actions,"write:"+key.User)
					state[storeID][key]=true
				}
			}
			return response(200,map[string]any{}),nil
		}
		t.Fatalf("unexpected OpenFGA HTTP %s %s",req.Method,req.URL.Path)
		return nil,errors.New("unexpected HTTP call")
	})
	if err:=client.ReconcileTupleScope(context.Background(),testStoreA,sampleModelID,scope,[]Tuple{ed});err!=nil {t.Fatal(err)}
	if len(state[testStoreA])!=2 || !state[testStoreA][ed] || state[testStoreA][paul] || !state[testStoreA][other] ||
		!state[testStoreB][ed] {t.Fatal("missing membership revocation or cross-tenant contamination")}
	if !reflect.DeepEqual(actions,[]string{"delete:user:usr000002"}) ||
		lastWriteModel!=sampleModelID {t.Fatalf("incorrect revocation actions=%v model=%s",actions,lastWriteModel)}
	if err:=client.ReconcileTupleScope(context.Background(),testStoreA,sampleModelID,scope,[]Tuple{ed});err!=nil {t.Fatal(err)}
	if posts!=1 {t.Fatal("idempotent reconciliation made an unnecessary write")}

	if err:=client.ReconcileTupleScope(context.Background(),testStoreA,sampleModelID,scope,nil);err!=nil {t.Fatal(err)}
	if state[testStoreA][ed] || !state[testStoreA][other] || !state[testStoreB][ed] {t.Fatal("failed full removal or deleted another scope")}
	if posts!=2 {t.Fatalf("expected the re-check and second revocation, posts=%d",posts)}
}

func TestReconcileTupleScopeDeletesBeforeGrantingAndValidatesModel(t *testing.T) {
	scope:=TupleScope{Object:"agent:ten00002-usr000001-agt00001",Relation:"chatter"}
	old:=Tuple{User:"user:usr000002",Relation:"chatter",Object:scope.Object}
	newGrant:=Tuple{User:"group:grp000001#member",Relation:"chatter",Object:scope.Object}
	present:=map[Tuple]bool{old:true}
	var actions []string
	modeled:=0
	client:=testClient(t,func(req *http.Request)(*http.Response,error) {
		if req.URL.Path=="/stores/"+testStoreA+"/authorization-models" {
			modeled++
			blob,_:=json.Marshal(modelWithID(t,sampleModelID))
			return response(200,authorizationModelResponse{AuthorizationModels:[]json.RawMessage{blob}}),nil
		}
		if req.URL.Path=="/stores/"+testStoreA+"/read" {
			var body tupleReadRequest
			if err:=json.NewDecoder(req.Body).Decode(&body);err!=nil {t.Fatal(err)}
			if body.TupleKey["relation"]!=scope.Relation || body.TupleKey["object"]!=scope.Object {t.Fatal("unsafe read")}
			var rows []readTuple
			for key:=range present {rows=append(rows,readTuple{Key:key})}
			return response(200,readTuplesResponse{Tuples:rows}),nil
		}
		if req.URL.Path=="/stores/"+testStoreA+"/write" {
			var batch tupleWriteRequest
			if err:=json.NewDecoder(req.Body).Decode(&batch);err!=nil {t.Fatal(err)}
			if batch.AuthorizationModelID!=sampleModelID {t.Fatal("unpinned model write")}
			if batch.Deletes!=nil {
				for _,key:=range batch.Deletes.TupleKeys {actions=append(actions,"delete");delete(present,key)}
			}
			if batch.Writes!=nil {
				for _,key:=range batch.Writes.TupleKeys {actions=append(actions,"write");present[key]=true}
			}
			return response(200,struct{}{}),nil
		}
		t.Fatalf("unexpected URL: %s",req.URL.Path)
		return nil,nil
	})
	if err:=client.ReconcileTupleScope(context.Background(),testStoreA,sampleModelID,scope,[]Tuple{newGrant});err!=nil {t.Fatal(err)}
	if !reflect.DeepEqual(actions,[]string{"delete","write"}) || modeled!=2 || !present[newGrant] || present[old] {
		t.Fatalf("unexpected replace result: actions=%v modelReads=%d state=%v",actions,modeled,present)
	}
}

func TestTupleScopeRejectsForgedInputsWithoutServerCall(t *testing.T) {
	calls:=0
	client:=testClient(t,func(*http.Request)(*http.Response,error) {
		calls++
		return response(200,map[string]any{}),nil
	})
	scope:=TupleScope{Object:"group:grp000001",Relation:"member"}
	good:=Tuple{Object:scope.Object,Relation:scope.Relation,User:"user:usr000001"}
	cases:=[]struct{
		name string
		storeID string
		modelID string
		scope TupleScope
		desired []Tuple
	}{
		{"foreign-store","TEN00002",sampleModelID,scope,[]Tuple{good}},
		{"unpinned-model",testStoreA,"latest",scope,[]Tuple{good}},
		{"unsafe-object",testStoreA,sampleModelID,TupleScope{Object:"tenant:TEN00002",Relation:"admin"},nil},
		{"unknown-relation",testStoreA,sampleModelID,TupleScope{Object:scope.Object,Relation:"admin"},nil},
		{"cross-object-grant",testStoreA,sampleModelID,scope,[]Tuple{{Object:"group:grp000002",Relation:"member",User:good.User}}},
		{"wrong-relation",testStoreA,sampleModelID,scope,[]Tuple{{Object:scope.Object,Relation:"owner",User:good.User}}},
		{"mutable-username",testStoreA,sampleModelID,scope,[]Tuple{{Object:scope.Object,Relation:"member",User:"edfoley"}}},
		{"invalid-group-membership",testStoreA,sampleModelID,scope,[]Tuple{{Object:scope.Object,Relation:"member",User:"group:grp000002#member"}}},
		{"duplicates",testStoreA,sampleModelID,scope,[]Tuple{good,good}},
	}
	for _,tc:=range cases {
		t.Run(tc.name,func(t *testing.T){
			err:=client.ReconcileTupleScope(context.Background(),tc.storeID,tc.modelID,tc.scope,tc.desired)
			if err==nil {t.Fatal("accepted invalid desired state")}
		})
	}
	if calls!=0 {t.Fatal("invalid or forged tuple intent reached OpenFGA")}
}

func TestTupleReconciliationFailsClosedOnUntrustedReadAndOutage(t *testing.T) {
	scope:=TupleScope{Object:"group:grp000001",Relation:"member"}
	good:=Tuple{Object:scope.Object,Relation:"member",User:"user:usr000001"}
	modelPayload,_:=json.Marshal(modelWithID(t,sampleModelID))
	for _,tc:=range []struct{name string; read func(*http.Request)(*http.Response,error)}{
		{"authorization-error",func(*http.Request)(*http.Response,error){return response(403,nil),nil}},
		{"network-error",func(*http.Request)(*http.Response,error){return nil,errors.New("unreachable")}},
		{"foreign-tuple",func(*http.Request)(*http.Response,error){
			return response(200,readTuplesResponse{Tuples:[]readTuple{{Key:Tuple{Object:"group:other",Relation:"member",User:good.User}}}}),nil
		}},
		{"conditioned-tuple",func(*http.Request)(*http.Response,error){
			condition:=json.RawMessage(`{"name":"bad_condition"}`)
			return response(200,readTuplesResponse{Tuples:[]readTuple{{Key:Tuple{Object:scope.Object,Relation:"member",User:good.User,Condition:&condition}}}}),nil
		}},
		{"pagination-loop",func(*http.Request)(*http.Response,error){
			return response(200,readTuplesResponse{ContinuationToken:"same"}),nil
		}},
		{"invalid-json",func(*http.Request)(*http.Response,error){
			return &http.Response{StatusCode:200,Body:io.NopCloser(strings.NewReader("bad")),Header:make(http.Header)},nil
		}},
	} {
		t.Run(tc.name,func(t *testing.T) {
			writes:=0
			client:=testClient(t,func(req *http.Request)(*http.Response,error){
				if req.URL.Path=="/stores/"+testStoreA+"/authorization-models" {
					return response(200,authorizationModelResponse{AuthorizationModels:[]json.RawMessage{modelPayload}}),nil
				}
				if req.URL.Path=="/stores/"+testStoreA+"/read" {return tc.read(req)}
				if req.URL.Path=="/stores/"+testStoreA+"/write" {writes++;return response(200,nil),nil}
				t.Fatalf("unexpected endpoint %q",req.URL.Path)
				return nil,nil
			})
			if err:=client.ReconcileTupleScope(context.Background(),testStoreA,sampleModelID,scope,[]Tuple{good});err==nil {
				t.Fatal("reconciled tuples without trusted complete upstream snapshot")
			}
			if writes!=0 {t.Fatal("untrusted read triggered a destructive tuple write")}
		})
	}
}

func TestTupleWriteFailureOrMissingReadBackNeverConfirmsRevocation(t *testing.T) {
	scope:=TupleScope{Object:"group:grp000001",Relation:"member"}
	old:=Tuple{Object:scope.Object,Relation:"member",User:"user:usr000001"}
	modelPayload,_:=json.Marshal(modelWithID(t,sampleModelID))
	for _,failWrite:=range []bool{true,false} {
		t.Run(map[bool]string{true:"write-failed",false:"write-not-visible"}[failWrite],func(t *testing.T) {
			writes:=0
			client:=testClient(t,func(req *http.Request)(*http.Response,error){
				switch req.URL.Path {
				case "/stores/"+testStoreA+"/authorization-models":
					return response(200,authorizationModelResponse{AuthorizationModels:[]json.RawMessage{modelPayload}}),nil
				case "/stores/"+testStoreA+"/read":
					return response(200,readTuplesResponse{Tuples:[]readTuple{{Key:old}}}),nil
				case "/stores/"+testStoreA+"/write":
					writes++
					if failWrite {return response(503,nil),nil}
					return response(200,nil),nil
				}
				return nil,errors.New("unexpected call")
			})
			if err:=client.ReconcileTupleScope(context.Background(),testStoreA,sampleModelID,scope,nil);err==nil {
				t.Fatal("confirmed revocation after failed or missing upstream state transition")
			}
			if writes!=1 {t.Fatal("unexpected duplicate writes")}
		})
	}
}
