package authentik

import (
    "context"
    "errors"
    "net/http"
    "testing"
)

func TestAccountSnapshotUsesExactUUIDAndIndependentReads(t *testing.T) {
    reads:=0
    c:=clientWith(t, func(req *http.Request)(*http.Response,error) {
        reads++
        if req.URL.Path!="/api/v3/core/users/" ||
           req.URL.Query().Get("uuid")!=userOne ||
           req.URL.Query().Get("include_groups")!="false" ||
           req.URL.Query().Get("page")!="1" ||
           req.URL.Query().Get("page_size")!="100" ||
           req.Header.Get("Authorization")!="Bearer fake-authentik-api-credential" {
            t.Fatalf("unexpected unscoped Authentik account read %s", req.URL)
        }
        return apiResponse(200,apiPage(1,1,0,1,userObj(17,userOne,true))),nil
    })
    account,err:=c.SnapshotUserAccount(context.Background(),userOne)
    if err!=nil || !account.Active || account.UUID!=userOne || reads!=2 {
        t.Fatalf("account snapshot=%#v reads=%d err=%v",account,reads,err)
    }
}

func TestAccountSnapshotDeniesChangedDisabledOrMissingIdentity(t *testing.T) {
    for _,tc:=range []struct{
        name string
        reply func(int)(*http.Response,error)
        err bool
        active bool
    }{
        {"disabled",func(_ int)(*http.Response,error){return apiResponse(200,apiPage(1,1,0,1,userObj(17,userOne,false))),nil},false,false},
        {"became-disabled",func(n int)(*http.Response,error){return apiResponse(200,apiPage(1,1,0,1,userObj(17,userOne,n==1))),nil},true,false},
        {"missing",func(_ int)(*http.Response,error){return apiResponse(200,apiPage(1,1,0,0)),nil},true,false},
        {"duplicate",func(_ int)(*http.Response,error){return apiResponse(200,apiPage(1,1,0,2,userObj(1,userOne,true),userObj(2,userOne,true))),nil},true,false},
        {"wrong-uuid",func(_ int)(*http.Response,error){return apiResponse(200,apiPage(1,1,0,1,userObj(17,userTwo,true))),nil},true,false},
        {"inactive-unknown",func(_ int)(*http.Response,error){u:=userObj(17,userOne,true);delete(u,"is_active");return apiResponse(200,apiPage(1,1,0,1,u)),nil},true,false},
        {"missing-pk",func(_ int)(*http.Response,error){u:=userObj(17,userOne,true);delete(u,"pk");return apiResponse(200,apiPage(1,1,0,1,u)),nil},true,false},
        {"outage",func(_ int)(*http.Response,error){return nil,errors.New("unavailable")},true,false},
        {"HTTP-403",func(_ int)(*http.Response,error){return apiResponse(403,map[string]any{"error":"secret"}),nil},true,false},
    } {
        t.Run(tc.name,func(t *testing.T){
            reads:=0
            c:=clientWith(t,func(*http.Request)(*http.Response,error){reads++;return tc.reply(reads)})
            got,err:=c.SnapshotUserAccount(context.Background(),userOne)
            if tc.err && err==nil {t.Fatalf("accepted untrusted account: %#v",got)}
            if !tc.err && (err!=nil || got.Active!=tc.active) {t.Fatalf("account=%#v err=%v",got,err)}
        })
    }
}

func TestAccountSnapshotRejectsUntrustedUUIDBeforeNetwork(t *testing.T) {
    calls:=0
    c:=clientWith(t,func(*http.Request)(*http.Response,error){calls++;return nil,errors.New("unreachable")})
    for _,uuid:=range []string{"","alice","550e8400-e29b-41d4-a716-446655440001/../other",
        "550E8400-E29B-41D4-A716-446655440001","550e8400-e29b-41d4-a716-446655440001?admin=1"} {
        if _,err:=c.SnapshotUserAccount(context.Background(),uuid);err==nil {t.Fatalf("accepted %q",uuid)}
    }
    if calls!=0 {t.Fatalf("bad UUID used network: %d",calls)}
}
