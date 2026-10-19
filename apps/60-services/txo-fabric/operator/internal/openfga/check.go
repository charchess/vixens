package openfga

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// CheckRequest is constructed only by server-owned controller state. The
// browser authority route never accepts its store/model/relation/object/user
// values from a browser or BFF request.
type CheckRequest struct {
	StoreID              string `json:"store_id"`
	AuthorizationModelID string `json:"authorization_model_id"`
	TupleKey             Tuple   `json:"tuple_key"`
}

type checkResponse struct { Allowed bool `json:"allowed"` }

// Check performs one exact, model-pinned authorization decision. It does not
// permit contextual tuples, pagination, a caller-selected endpoint, or a
// fallback/default model.
func (c *Client) Check(ctx context.Context, request CheckRequest) (bool, error) {
	if c == nil || !storeIDPattern.MatchString(request.StoreID) ||
		!storeIDPattern.MatchString(request.AuthorizationModelID) ||
		request.TupleKey.User == "" || request.TupleKey.Relation == "" || request.TupleKey.Object == "" {
		return false, errors.New("invalid pinned OpenFGA check")
	}
	body, err := json.Marshal(request)
	if err != nil { return false, errors.New("unable to encode pinned OpenFGA check") }
	data, err := c.request(ctx, "POST", fmt.Sprintf("/stores/%s/check", request.StoreID), body)
	if err != nil { return false, err }
	var result checkResponse
	if json.Unmarshal(data, &result) != nil { return false, errors.New("invalid OpenFGA check response") }
	return result.Allowed, nil
}
