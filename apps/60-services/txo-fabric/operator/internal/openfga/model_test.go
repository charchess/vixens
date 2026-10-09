package openfga

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

const sampleModelID = "01H0H015178Y2V4CX10C2KGHF6"

func modelWithID(t *testing.T, id string) map[string]any {
	t.Helper()
	want, err := desiredModel()
	if err != nil { t.Fatal(err) }
	out := map[string]any{"id": id}
	for k, v := range want { out[k] = v }
	return out
}

func TestCompiledModelValidAndStable(t *testing.T) {
	model, err := desiredModel()
	if err != nil { t.Fatal(err) }
	if model["schema_version"] != "1.1" { t.Fatal("wrong schema") }
	types := model["type_definitions"].([]any)
	if len(types) != 7 { t.Fatalf("expected seven Fabric types, got %d", len(types)) }
	fp1, err := ModelFingerprint()
	if err != nil || len(fp1) != 64 { t.Fatalf("invalid fingerprint: %q %v", fp1, err) }
	fp2, _ := ModelFingerprint()
	if fp2 != fp1 { t.Fatal("authorization model fingerprint is non-deterministic") }

	found := map[string]bool{}
	for _, item := range types {
		x := item.(map[string]any)
		found[x["type"].(string)] = true
	}
	for _, typ := range []string{"user","platform","tenant","group","agent","workspace","provider_account"} {
		if !found[typ] { t.Fatalf("missing Fabric resource type %q", typ) }
	}
}

func TestEnsureAuthorizationModelPublishesOnlyToEmptyStore(t *testing.T) {
	ctx := context.Background()
	created := false
	var latest []json.RawMessage
	posts := 0
	c := testClient(t,func(r *http.Request)(*http.Response,error) {
		if r.Header.Get("Authorization") == "" {
			t.Fatal("OpenFGA request without platform credential")
		}
		if r.URL.Path != "/stores/"+testStoreA+"/authorization-models" { t.Fatalf("unsafe model endpoint: %s",r.URL.Path) }
		if r.Method == http.MethodGet {
			return response(http.StatusOK,authorizationModelResponse{AuthorizationModels:latest}),nil
		}
		if r.Method != http.MethodPost { t.Fatalf("unexpected HTTP method: %s",r.Method) }
		posts++
		var payload map[string]any
		if err:=json.NewDecoder(r.Body).Decode(&payload);err!=nil {t.Fatal(err)}
		model,err:=desiredModel()
		if err!=nil || !modelMatchesWithoutID(payload,model) {
			t.Fatal("published model differs from source-controlled Fabric model")
		}
		created=true
		blob,_:=json.Marshal(modelWithID(t,sampleModelID))
		latest=[]json.RawMessage{blob}
		return response(http.StatusCreated,authorizationModelWriteResponse{AuthorizationModelID:sampleModelID}),nil
	})
	id,err:=c.EnsureAuthorizationModel(ctx,testStoreA)
	if err!=nil || id!=sampleModelID || !created || posts!=1 {t.Fatalf("first publication: id=%s err=%v posts=%d",id,err,posts)}
	id,err=c.EnsureAuthorizationModel(ctx,testStoreA)
	if err!=nil || id!=sampleModelID || posts!=1 {t.Fatalf("idempotent adoption: id=%s err=%v posts=%d",id,err,posts)}
	if err:=c.ValidateModelBinding(ctx,testStoreA,id);err!=nil {t.Fatal(err)}
}

func modelMatchesWithoutID(a,b map[string]any) bool {
	j,_:=json.Marshal(a)
	id,ok:=modelMatchesDesired(append([]byte(`{"id":"`+sampleModelID+`",`), j[1:]...),b)
	return ok && id==sampleModelID
}

func TestUnknownModelNeverReplacedOrRolledBack(t *testing.T) {
	ctx:=context.Background()
	known:=modelWithID(t,sampleModelID)
	untrusted:=modelWithID(t,testStoreB)
	untrusted["schema_version"]="1.2"
	unknown,_:=json.Marshal(untrusted)
	old,_:=json.Marshal(known)
	posts:=0
	c:=testClient(t,func(r *http.Request)(*http.Response,error){
		if r.Method==http.MethodPost {posts++;return response(201,nil),nil}
		return response(200,authorizationModelResponse{AuthorizationModels:[]json.RawMessage{unknown,old}}),nil
	})
	if _,err:=c.EnsureAuthorizationModel(ctx,testStoreA);err==nil {t.Fatal("accepted newer unknown model")}
	if err:=c.ValidateModelBinding(ctx,testStoreA,sampleModelID);err==nil {t.Fatal("old model accepted over unexpected latest")}
	if posts!=0 {t.Fatal("silently wrote an authorization-model migration")}
}

func TestUnauthorizedAndUnreachableModelServiceAreFailClosed(t *testing.T) {
	for _,code:=range []int{401,403,429,503} {
		c:=testClient(t,func(*http.Request)(*http.Response,error){
			return response(code,map[string]string{"secret":"must-not-log"}),nil
		})
		if _,err:=c.EnsureAuthorizationModel(context.Background(),testStoreA);err==nil || strings.Contains(err.Error(),"must-not-log"){
			t.Fatalf("upstream error %d leaked a credential or was ignored: %v",code,err)
		}
	}
	c:=testClient(t,func(*http.Request)(*http.Response,error){
		return nil,errors.New("socket down")
	})
	if _,err:=c.EnsureAuthorizationModel(context.Background(),testStoreA);err==nil {t.Fatal("network outage ignored")}
	for _,bad:=range []string{"TEN00001","not-a-store","",testStoreA+"?foo=bar"} {
		if _,err:=c.EnsureAuthorizationModel(context.Background(),bad);err==nil {t.Fatalf("accepted store ID %q",bad)}
	}
}

func TestModelPublicationDoesNotTrustMalformedResponse(t *testing.T) {
	for _,bad:=range []any{
		map[string]string{"authorization_model_id":"not-ulid"},
		map[string]string{"missing":"id"},
	} {
		c:=testClient(t,func(r *http.Request)(*http.Response,error){
			if r.Method==http.MethodGet {return response(200,authorizationModelResponse{}),nil}
			return response(201,bad),nil
		})
		if _,err:=c.EnsureAuthorizationModel(context.Background(),testStoreA);err==nil {t.Fatal("malformed publication accepted")}
	}
	c:=testClient(t,func(r *http.Request)(*http.Response,error){
		if r.Method==http.MethodGet {return response(200,authorizationModelResponse{}),nil}
		return response(201,authorizationModelWriteResponse{AuthorizationModelID:sampleModelID}),nil
	})
	if _,err:=c.EnsureAuthorizationModel(context.Background(),testStoreA);err==nil {t.Fatal("unconfirmed publication accepted")}
}

func TestModelListPaginatesAndDeniesAmbiguousCursor(t *testing.T) {
	desired,_:=desiredModel()
	blob,_:=json.Marshal(modelWithID(t,sampleModelID))
	c:=testClient(t,func(r *http.Request)(*http.Response,error){
		if r.URL.Query().Get("continuation_token")=="" {
			return response(200,authorizationModelResponse{ContinuationToken:"next"}),nil
		}
		return response(200,authorizationModelResponse{AuthorizationModels:[]json.RawMessage{blob}}),nil
	})
	models,err:=c.currentModels(context.Background(),testStoreA)
	if err!=nil || len(models)!=1 {t.Fatalf("pagination: %d models %v",len(models),err)}
	_,matches:=modelMatchesDesired(models[0],desired)
	if !matches {t.Fatal("API model not equivalent to checked-in model")}
	bad:=testClient(t,func(*http.Request)(*http.Response,error){
		return response(200,authorizationModelResponse{ContinuationToken:"loop"}),nil
	})
	if _,err:=bad.currentModels(context.Background(),testStoreA);err==nil {t.Fatal("repeated cursor accepted")}
}
