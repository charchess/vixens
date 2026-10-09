package openfga

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
)

// A Tuple is a direct, unconditioned relationship in the approved Fabric
// authorization model. No caller-facing API accepts these values: only
// Fabric's privileged reconciliation process may supply desired state.
type Tuple struct {
	User string `json:"user"`
	Relation string `json:"relation"`
	Object string `json:"object"`
	// OpenFGA conditional relationships are unsupported by this v0.1 sync.
	// A pointer keeps Tuple comparable for exact-set reconciliation while
	// ensuring an unexpected JSON condition is observable and rejected.
	Condition *json.RawMessage `json:"condition,omitempty"`
}

// TupleScope identifies exactly one Fabric-controlled object/relation.
// A complete, verified authoritative snapshot is REQUIRED before passing
// desired tuples; a partial IAM response is never an empty desired set.
type TupleScope struct {
	Object string
	Relation string
}

type readTuple struct {
	Key Tuple `json:"key"`
	Condition json.RawMessage `json:"condition,omitempty"`
}

type readTuplesResponse struct {
	Tuples []readTuple `json:"tuples"`
	ContinuationToken string `json:"continuation_token"`
}

type tupleReadRequest struct {
	TupleKey map[string]string `json:"tuple_key"`
	PageSize int `json:"page_size"`
	ContinuationToken string `json:"continuation_token,omitempty"`
}

type tupleWriteBatch struct {
	TupleKeys []Tuple `json:"tuple_keys"`
	OnDuplicate string `json:"on_duplicate,omitempty"`
	OnMissing string `json:"on_missing,omitempty"`
}

type tupleWriteRequest struct {
	Writes *tupleWriteBatch `json:"writes,omitempty"`
	Deletes *tupleWriteBatch `json:"deletes,omitempty"`
	AuthorizationModelID string `json:"authorization_model_id"`
}

const maxTupleBatch = 100

var tupleIdentityPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)

func parseTupleObject(value string) (string, string, bool) {
	for _, objectType := range []string{"group", "agent"} {
		prefix := objectType+":"
		if len(value)>len(prefix) && value[:len(prefix)]==prefix &&
			tupleIdentityPattern.MatchString(value[len(prefix):]) {
			return objectType, value[len(prefix):], true
		}
	}
	return "", "", false
}

func validateTupleScope(scope TupleScope) error {
	objectType, _, ok := parseTupleObject(scope.Object)
	if !ok {
		return errors.New("invalid Fabric authorization tuple object")
	}
	switch objectType {
	case "group":
		if scope.Relation == "member" || scope.Relation == "tenant" { return nil }
	case "agent":
		if scope.Relation == "chatter" || scope.Relation == "manager" ||
			scope.Relation == "owner" || scope.Relation == "tenant" { return nil }
	}
	return errors.New("unsupported Fabric tuple relation for managed scope")
}

func validateTupleUser(scope TupleScope, user string) bool {
	for _, prefix := range []string{"user:", "group:", "tenant:"} {
		if len(user) <= len(prefix) || user[:len(prefix)] != prefix { continue }
		name := user[len(prefix):]
		switch {
		case prefix=="user:":
			return tupleIdentityPattern.MatchString(name) && scope.Relation!="tenant"
		case prefix=="tenant:":
			return tupleIdentityPattern.MatchString(name) && (scope.Relation=="tenant" || scope.Relation=="owner")
		case prefix=="group:" && (scope.Relation=="chatter" || scope.Relation=="manager"):
			const member="#member"
			return len(name)>len(member) && name[len(name)-len(member):]==member &&
				tupleIdentityPattern.MatchString(name[:len(name)-len(member)])
		case prefix=="group:" && scope.Relation=="owner":
			return tupleIdentityPattern.MatchString(name)
		}
	}
	return false
}

func validateScopedTuple(scope TupleScope, tuple Tuple) error {
	if tuple.Condition != nil || tuple.Object != scope.Object || tuple.Relation != scope.Relation ||
		!validateTupleUser(scope, tuple.User) {
		return errors.New("invalid, out-of-scope or untrusted Fabric authorization tuple")
	}
	return nil
}

func scopedTuplesMatch(scope TupleScope, tuples []Tuple, desired map[Tuple]bool) error {
	actual := make(map[Tuple]bool,len(tuples))
	for _, tuple := range tuples {
		if err := validateScopedTuple(scope,tuple); err != nil { return err }
		if actual[tuple] { return errors.New("duplicate tuple returned by OpenFGA") }
		actual[tuple]=true
	}
	if len(actual)!=len(desired) { return errors.New("Fabric OpenFGA tuple convergence not confirmed") }
	for tuple := range desired {
		if !actual[tuple] { return errors.New("Fabric OpenFGA tuple convergence not confirmed") }
	}
	return nil
}

func sortTuples(tuples []Tuple) {
	sort.Slice(tuples,func(i,j int)bool {
		if tuples[i].Object!=tuples[j].Object {return tuples[i].Object<tuples[j].Object}
		if tuples[i].Relation!=tuples[j].Relation {return tuples[i].Relation<tuples[j].Relation}
		return tuples[i].User<tuples[j].User
	})
}

// readTupleScope paginates only a SINGLE object/relation. If the upstream
// server returns tuples outside this scope, a looped cursor or malformed
// data, nothing is trusted and no writes are attempted.
func (c *Client) readTupleScope(ctx context.Context, storeID string, scope TupleScope) ([]Tuple,error) {
	var collected []Tuple
	token := ""
	seen := make(map[string]bool)
	for page:=0;page<maxPages;page++ {
		body,_:=json.Marshal(tupleReadRequest{
			TupleKey:map[string]string{"object":scope.Object,"relation":scope.Relation},
			PageSize:100,ContinuationToken:token,
		})
		raw,err:=c.request(ctx,http.MethodPost,"/stores/"+storeID+"/read",body)
		if err!=nil {return nil,err}
		var resp readTuplesResponse
		if err:=json.Unmarshal(raw,&resp);err!=nil {return nil,errors.New("invalid Fabric OpenFGA tuple read response")}
		for _,record:=range resp.Tuples {
			if len(record.Condition)>0 && string(record.Condition)!="null" {
				return nil,errors.New("unexpected conditioned Fabric authorization tuple")
			}
			if err:=validateScopedTuple(scope,record.Key);err!=nil {return nil,err}
			collected=append(collected,record.Key)
		}
		if resp.ContinuationToken=="" {return collected,nil}
		if seen[resp.ContinuationToken] {return nil,errors.New("OpenFGA tuple pagination repeated a cursor")}
		seen[resp.ContinuationToken]=true
		token=resp.ContinuationToken
	}
	return nil,errors.New("OpenFGA tuple discovery exceeded pagination bound")
}

// ReconcileTupleScope converges one owned relation from a COMPLETE and
// authoritative snapshot, including removal of stale members. It is an
// internal primitive, NOT wired to the TenantBundle until Authentik identity
// mapping, source freshness and revocation policies are complete.
//
// For safety, this restricts the permitted object/relation vocabulary; a
// browser may never call it, and the calling controller must validate tenant
// ownership and hold the authoritative relationship writer lease.
func (c *Client) ReconcileTupleScope(ctx context.Context, storeID, modelID string, scope TupleScope, desired []Tuple) error {
	if !storeIDPattern.MatchString(storeID) || !storeIDPattern.MatchString(modelID) {
		return errors.New("invalid Fabric store or pinned authorization model identity")
	}
	if err:=validateTupleScope(scope);err!=nil{return err}
	expected:=make(map[Tuple]bool,len(desired))
	for _,tuple:=range desired {
		if err:=validateScopedTuple(scope,tuple);err!=nil{return err}
		if expected[tuple] {return errors.New("duplicate tuple in authoritative Fabric snapshot")}
		expected[tuple]=true
	}

	if err:=c.ValidateModelBinding(ctx,storeID,modelID);err!=nil {
		return fmt.Errorf("Fabric OpenFGA model not trusted for tuple reconciliation: %w",err)
	}
	observed,err:=c.readTupleScope(ctx,storeID,scope)
	if err!=nil{return fmt.Errorf("Fabric OpenFGA tuple inventory failed: %w",err)}
	present:=make(map[Tuple]bool,len(observed))
	for _,tuple:=range observed {
		if present[tuple] {return errors.New("duplicate Fabric tuple in upstream inventory")}
		present[tuple]=true
	}
	var deletes,writes []Tuple
	for tuple:=range present {if !expected[tuple] {deletes=append(deletes,tuple)}}
	for tuple:=range expected {if !present[tuple] {writes=append(writes,tuple)}}
	sortTuples(deletes)
	sortTuples(writes)

	// Revoke before adding any new grants. The upstream endpoint supports
	// idempotent on_missing/on_duplicate options. A partial batch failure is
	// NOT treated as success: the next reconcile re-reads actual state.
	for start:=0;start<len(deletes);start+=maxTupleBatch {
		end:=start+maxTupleBatch
		if end>len(deletes) {end=len(deletes)}
		req:=tupleWriteRequest{AuthorizationModelID:modelID,Deletes:&tupleWriteBatch{
			TupleKeys:deletes[start:end],OnMissing:"ignore",
		}}
		if err:=c.writeTupleBatch(ctx,storeID,req);err!=nil {return err}
	}
	for start:=0;start<len(writes);start+=maxTupleBatch {
		end:=start+maxTupleBatch
		if end>len(writes) {end=len(writes)}
		req:=tupleWriteRequest{AuthorizationModelID:modelID,Writes:&tupleWriteBatch{
			TupleKeys:writes[start:end],OnDuplicate:"ignore",
		}}
		if err:=c.writeTupleBatch(ctx,storeID,req);err!=nil {return err}
	}

	// A successful Write response is insufficient proof of revocation. After
	// all changes, verify exact snapshot and still-approved pinned model.
	actual,err:=c.readTupleScope(ctx,storeID,scope)
	if err!=nil {return err}
	if err:=scopedTuplesMatch(scope,actual,expected);err!=nil{return err}
	if err:=c.ValidateModelBinding(ctx,storeID,modelID);err!=nil{return err}
	return nil
}

func (c *Client) writeTupleBatch(ctx context.Context, storeID string, req tupleWriteRequest) error {
	body,err:=json.Marshal(req)
	if err!=nil {return errors.New("invalid Fabric tuple write request")}
	if _,err:=c.request(ctx,http.MethodPost,"/stores/"+storeID+"/write",body);err!=nil {
		return fmt.Errorf("Fabric OpenFGA tuple transaction failed: %w",err)
	}
	return nil
}
