package openfga

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
)

// compiledAuthorizationModel is generated from the model.fga contract
// (apps/60-services/txo-fabric/openfga/model/model.fga). CI independently
// validates this JSON with the same two-tenant test suite as its DSL source.
//
//go:embed model.json
var compiledAuthorizationModel []byte

type authorizationModelResponse struct {
	AuthorizationModels []json.RawMessage `json:"authorization_models"`
	ContinuationToken string `json:"continuation_token"`
}

type authorizationModelWriteResponse struct {
	AuthorizationModelID string `json:"authorization_model_id"`
}

// ErrUnknownAuthorizationModel means the newest retained model is not the
// source-controlled Fabric model. Callers must not replace it implicitly.
var ErrUnknownAuthorizationModel = errors.New("tenant has an unknown or changed OpenFGA authorization model")

// ModelFingerprint is the content digest of the compiled, ordered Fabric
// authorization model. Do not use a mutable "latest" model for a Check.
func ModelFingerprint() (string, error) {
	payload, err := desiredModel()
	if err != nil { return "", err }
	canonical, err := json.Marshal(payload)
	if err != nil { return "", err }
	hash := sha256.Sum256(canonical)
	return hex.EncodeToString(hash[:]), nil
}

func desiredModel() (map[string]any, error) {
	var model map[string]any
	if err := json.Unmarshal(compiledAuthorizationModel, &model); err != nil {
		return nil, errors.New("invalid compiled Fabric authorization model")
	}
	if model["schema_version"] != "1.1" { return nil, errors.New("unsupported Fabric authorization model schema") }
	defs, ok := model["type_definitions"].([]any)
	if !ok || len(defs) < 3 { return nil, errors.New("compiled Fabric authorization model has no definitions") }
	if len(model) != 2 { return nil, errors.New("unexpected fields in compiled Fabric authorization model") }
	return model, nil
}

func normalizedModelValue(v any) any {
	switch current := v.(type) {
	case map[string]any:
		out := make(map[string]any,len(current))
		for k, value := range current {
			// OpenFGA serializes non-executable tooling metadata on reads;
			// source-controlled policy equivalence must not depend on it.
			if k == "metadata" { continue }
			// The API also materializes an empty conditions map even when
			// the source model declares no conditions. Preserve non-empty
			// conditions because they change authorization semantics.
			if k == "conditions" {
				if conditions, ok := value.(map[string]any); ok && len(conditions) == 0 { continue }
			}
			// OpenFGA JSON/protobuf default serialization differs for
			// omitted zero fields; never normalize away a meaningful
			// userset node such as {"this":{}}.
			if value == nil || value == "" { continue }
			normalized := normalizedModelValue(value)
			if empty, ok := normalized.(map[string]any); ok && len(empty) == 0 && k != "this" { continue }
			out[k] = normalized
		}
		return out
	case []any:
		out := make([]any, len(current))
		for i := range current { out[i] = normalizedModelValue(current[i]) }
		return out
	default:
		return v
	}
}

func modelMatchesDesired(raw json.RawMessage, desired map[string]any) (string, bool) {
	var actual map[string]any
	if err := json.Unmarshal(raw, &actual); err != nil { return "", false }
	id, ok := actual["id"].(string)
	if !ok || !storeIDPattern.MatchString(id) { return "", false }
	delete(actual, "id")
	delete(actual, "created_at")
	return id, reflect.DeepEqual(normalizedModelValue(actual), normalizedModelValue(desired))
}

func (c *Client) currentModels(ctx context.Context, storeID string) ([]json.RawMessage, error) {
	token := ""
	seen := map[string]bool{}
	var models []json.RawMessage
	for page := 0; page < maxPages; page++ {
		path := "/stores/"+storeID+"/authorization-models?page_size=100"
		if token!="" { path+="&continuation_token="+url.QueryEscape(token) }
		resp, err := c.request(ctx,http.MethodGet,path,nil)
		if err!=nil { return nil,err }
		var data authorizationModelResponse
		if err:=json.Unmarshal(resp,&data);err!=nil {return nil,errors.New("invalid Fabric authorization models response")}
		models=append(models,data.AuthorizationModels...)
		if data.ContinuationToken=="" {return models,nil}
		if seen[data.ContinuationToken] {return nil,errors.New("OpenFGA model pagination cursor repeated")}
		seen[data.ContinuationToken]=true
		token=data.ContinuationToken
	}
	return nil,errors.New("OpenFGA model discovery exceeded pagination bound")
}

// EnsureAuthorizationModel ensures that a store's most recent model is exactly
// the approved Fabric model. It publishes only to an EMPTY store. An unknown
// existing model is NOT upgraded, rolled back or overwritten automatically;
// migrations require a separate reviewed mechanism and explicit policy.
func (c *Client) EnsureAuthorizationModel(ctx context.Context, storeID string) (string,error) {
	if !storeIDPattern.MatchString(storeID) { return "",errors.New("invalid tenant OpenFGA store ID") }
	desired,err:=desiredModel(); if err!=nil{return "",err}
	found,err:=c.currentModels(ctx,storeID); if err!=nil{return "",err}
	if len(found)>0 { id,match:=modelMatchesDesired(found[0],desired); if !match{return "",ErrUnknownAuthorizationModel}; return id,nil }
	return c.publishAuthorizationModel(ctx,storeID,desired,nil)
}

// MigrateAuthorizationModel appends only to exactly one valid schema-1.1,
// noncanonical legacy revision. It never deletes or overwrites a revision.
func (c *Client) MigrateAuthorizationModel(ctx context.Context, storeID string) (string,error) {
	if !storeIDPattern.MatchString(storeID) { return "",errors.New("invalid tenant OpenFGA store ID") }
	desired,err:=desiredModel(); if err!=nil{return "",err}
	found,err:=c.currentModels(ctx,storeID); if err!=nil{return "",err}
	if len(found)!=1 || !isBoundedLegacyModel(found[0]) { return "",errors.New("tenant OpenFGA model is not an approved bounded legacy migration state") }
	if _,match:=modelMatchesDesired(found[0],desired);match { return "",errors.New("tenant OpenFGA model migration is not required") }
	return c.publishAuthorizationModel(ctx,storeID,desired,found[0])
}

func isBoundedLegacyModel(raw json.RawMessage) bool {
	var model map[string]any
	if err:=json.Unmarshal(raw,&model);err!=nil{return false}
	id,ok:=model["id"].(string)
	if !ok || !storeIDPattern.MatchString(id) || model["schema_version"]!="1.1" {return false}
	if defs,ok:=model["type_definitions"].([]any);!ok || len(defs)==0{return false}
	delete(model,"id"); delete(model,"created_at")
	if conditions, present := model["conditions"]; present {
		empty, ok := conditions.(map[string]any)
		if !ok || len(empty) != 0 { return false }
		delete(model, "conditions")
	}
	return len(model)==2
}

func (c *Client) publishAuthorizationModel(ctx context.Context, storeID string, desired map[string]any, legacy json.RawMessage) (string,error) {
	payload,err:=json.Marshal(desired); if err!=nil{return "",errors.New("invalid Fabric authorization model JSON")}
	resp,err:=c.request(ctx,http.MethodPost,"/stores/"+storeID+"/authorization-models",payload); if err!=nil{return "",err}
	var written authorizationModelWriteResponse
	if err:=json.Unmarshal(resp,&written);err!=nil || !storeIDPattern.MatchString(written.AuthorizationModelID) { return "",errors.New("OpenFGA returned an invalid authorization model identity") }
	found,err:=c.currentModels(ctx,storeID); if err!=nil{return "",err}
	if len(found)==0 {return "",errors.New("new model is not observable after publication")}
	id,match:=modelMatchesDesired(found[0],desired)
	if !match || id!=written.AuthorizationModelID { return "",errors.New("OpenFGA model publication is not uniquely confirmed") }
	if legacy!=nil && (len(found)!=2 || !reflect.DeepEqual(found[1],legacy)) { return "",errors.New("OpenFGA model migration is not uniquely confirmed") }
	return id,nil
}

// ValidateModelBinding ensures that a previously stored ID remains the
// *latest* approved version, not merely one historical model in a store.
func (c *Client) ValidateModelBinding(ctx context.Context, storeID, modelID string) error {
	if !storeIDPattern.MatchString(storeID) {return errors.New("invalid tenant OpenFGA store ID")}
	if !storeIDPattern.MatchString(modelID) {return errors.New("invalid bound authorization model ID")}
	found,err:=c.currentModels(ctx,storeID)
	if err!=nil{return err}
	if len(found)==0 {return errors.New("tenant OpenFGA store has no authorization model")}
	desired,err:=desiredModel()
	if err!=nil{return err}
	id,match:=modelMatchesDesired(found[0],desired)
	if !match || id!=modelID {
		return fmt.Errorf("tenant OpenFGA authorization model binding drift (expected approved model)")
	}
	return nil
}

