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
	"reflect"
	"strings"
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
			// OpenFGA JSON/protobuf default serialization differs for
			// omitted zero fields; never normalize away a meaningful
			// userset node such as {"this":{}}.
			if value == nil || value == "" { continue }
			out[k] = normalizedModelValue(value)
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
		if token!="" { path+="&continuation_token="+urlQueryEscape(token) }
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
	if !storeIDPattern.MatchString(storeID) {
		return "",errors.New("invalid tenant OpenFGA store ID")
	}
	desired,err:=desiredModel()
	if err!=nil{return "",err}
	found,err:=c.currentModels(ctx,storeID)
	if err!=nil{return "",err}
	if len(found)>0 {
		id,match:=modelMatchesDesired(found[0],desired)
		if !match {return "",errors.New("tenant has an unknown or changed OpenFGA authorization model")}
		return id,nil
	}
	payload,err:=json.Marshal(desired)
	if err!=nil{return "",errors.New("invalid Fabric authorization model JSON")}
	resp,err:=c.request(ctx,http.MethodPost,"/stores/"+storeID+"/authorization-models",payload)
	if err!=nil{return "",err}
	var written authorizationModelWriteResponse
	if err:=json.Unmarshal(resp,&written);err!=nil || !storeIDPattern.MatchString(written.AuthorizationModelID) {
		return "",errors.New("OpenFGA returned an invalid authorization model identity")
	}
	// A successful POST does not prove that the server made this model
	// visible or that concurrent writes have not superseded it.
	found,err=c.currentModels(ctx,storeID)
	if err!=nil{return "",err}
	if len(found)==0 {return "",errors.New("new model is not observable after publication")}
	id,match:=modelMatchesDesired(found[0],desired)
	if !match || id!=written.AuthorizationModelID {
		return "",errors.New("OpenFGA model publication is not uniquely confirmed")
	}
	return id,nil
}

// ValidateModelBinding ensures that a previously stored ID remains the
// *latest* approved version, not merely one historical model in a store.
func (c *Client) ValidateModelBinding(ctx context.Context, storeID, modelID string) error {
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

func urlQueryEscape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s, "%", "%25"), "+", "%2B"), "&", "%26")
}
