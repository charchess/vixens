package authentik

import (
    "context"
    "encoding/json"
    "errors"
    "net/url"
    "reflect"
)

// UserAccountSnapshot is a verified, complete Authentik API result, NOT an
// authorization grant and NOT an OIDC subject. A user can be inactive while
// its immutable UUID remains known to Fabric.
type UserAccountSnapshot struct {
    UUID string
    Active bool
}

// userAccountOnce queries solely by the immutable UUID; neither a username,
// email address, browser-selected tenant nor a fuzzy search is accepted.
func (c *Client) userAccountOnce(ctx context.Context, uuid string) (UserAccountSnapshot, error) {
    rawUsers, err := c.collect(ctx, "/core/users/", url.Values{
        "uuid": {uuid},
        "include_groups": {"false"},
    })
    if err != nil { return UserAccountSnapshot{}, err }
    if len(rawUsers) != 1 { return UserAccountSnapshot{}, errors.New("missing or ambiguous Authentik user UUID") }
    var user struct {
        PK int `json:"pk"`
        UUID string `json:"uuid"`
        Active *bool `json:"is_active"`
    }
    if err := json.Unmarshal(rawUsers[0], &user); err != nil ||
        user.PK <= 0 || user.UUID != uuid || user.Active == nil {
        return UserAccountSnapshot{}, errors.New("untrusted Fabric Authentik user identity or account state")
    }
    return UserAccountSnapshot{UUID: uuid, Active: *user.Active}, nil
}

// SnapshotUserAccount performs independent complete reads to detect account
// deletion, deactivation and UUID mismatches during enrollment. Callers must
// also check freshness at the actual authorization boundary: this is a
// point-in-time source check, not an authorization or revocation guarantee.
func (c *Client) SnapshotUserAccount(ctx context.Context, uuid string) (UserAccountSnapshot, error) {
    if !uuidPattern.MatchString(uuid) {
        return UserAccountSnapshot{}, errors.New("invalid immutable Authentik user UUID")
    }
    first, err := c.userAccountOnce(ctx, uuid)
    if err != nil { return UserAccountSnapshot{}, err }
    second, err := c.userAccountOnce(ctx, uuid)
    if err != nil { return UserAccountSnapshot{}, err }
    if !reflect.DeepEqual(first, second) {
        return UserAccountSnapshot{}, errors.New("Authentik user account state changed during enrollment snapshot")
    }
    return second, nil
}
