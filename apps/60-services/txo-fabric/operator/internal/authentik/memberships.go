// Package authentik reads live human IAM state over the private authentik
// service boundary. It never creates, joins or deletes human memberships.
package authentik

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	privateOrigin = "https://authentik.truxonline.com"
	apiPrefix = "/api/v3"
	pageSize = 100
	maxPages = 100
	maxBodyBytes = 1048576
)

var (
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	fabricGroupPattern = regexp.MustCompile(`^txo-fabric-[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)
)

// Client is operator-owned; its token is NOT a tenant-supplied OAuth token.
type Client struct {
	token string
	httpClient *http.Client
}

// transientTransportFailure admits only a bounded retry for a timed-out
// transport while the caller's own context remains live. Authentication,
// TLS/protocol and response-validation errors remain terminal.
func transientTransportFailure(ctx context.Context, err error) bool {
	if err==nil || ctx.Err()!=nil {return false}
	var transportErr net.Error
	return errors.As(err,&transportErr) && transportErr.Timeout()
}

func NewClient(token string, transport http.RoundTripper) (*Client, error) {
	if len(token)<16 || strings.TrimSpace(token)!=token || strings.ContainsAny(token,"\r\n") {
		return nil,errors.New("Fabric Authentik service credential is missing or malformed")
	}
	if transport==nil {
		// Authentik's ClusterIP only serves cleartext HTTP: never transmit
		// this privileged API token to port 9000. Instead verify the exact
		// public TLS hostname/certificate while routing TCP inside the
		// cluster through the existing Traefik websecure service. No proxy,
		// arbitrary URL, redirect or plaintext fallback is permitted.
		dialer := &net.Dialer{Timeout: 3*time.Second}
		transport = &http.Transport{
			Proxy: nil,
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if network != "tcp" || addr != "authentik.truxonline.com:443" {
					return nil, errors.New("unapproved Fabric IAM destination")
				}
				return dialer.DialContext(ctx, network, "traefik.traefik.svc.cluster.local:443")
			},
		}
	}
	return &Client{token:token,httpClient:&http.Client{
		Transport:transport,Timeout:15*time.Second,
		CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse},
	}},nil
}

type pagination struct {
	Next int `json:"next"`
	Count int `json:"count"`
	Current int `json:"current"`
	TotalPages int `json:"total_pages"`
}

type rawPage struct {
	Pagination *pagination `json:"pagination"`
	Results []json.RawMessage `json:"results"`
}

// GroupMembership is a source snapshot, NOT a Fabric grant. UUIDs are
// Authentik identities and MUST be explicitly mapped to immutable Fabric
// identity IDs before creating OpenFGA tuples.
type GroupMembership struct {
	Name string
	AuthentikGroupUUID string
	ActiveAuthentikUserUUIDs []string
}

type authentikGroup struct {
	PK string `json:"pk"`
	Name string `json:"name"`
}

type authentikUser struct {
	PK int `json:"pk"`
	UUID string `json:"uuid"`
	IsActive *bool `json:"is_active"`
	Groups []string `json:"groups"`
}

func (c *Client) get(ctx context.Context, path string, values url.Values) ([]byte,error) {
	if c==nil || c.httpClient==nil || len(c.token)<16 {
		return nil,errors.New("Fabric Authentik client is not initialized")
	}
	// Authentik reads are idempotent. Retry exactly once only when the
	// transport failed before any response; authorization and response
	// validation failures remain fail-closed and are never retried.
	for attempt:=0;attempt<2;attempt++ {
		// Revalidate before every attempt. The context can be cancelled in the
		// narrow interval after a timed transport failure is classified and
		// before the retry request is constructed.
		if err:=ctx.Err();err!=nil {return nil,fmt.Errorf("Fabric Authentik API unavailable: %w",err)}
		req,err:=http.NewRequestWithContext(ctx,http.MethodGet,privateOrigin+apiPrefix+path+"?"+values.Encode(),nil)
		if err!=nil{return nil,errors.New("cannot prepare Fabric Authentik request")}
		req.Header.Set("Authorization","Bearer "+c.token)
		req.Header.Set("Accept","application/json")
		req.Header.Set("Cache-Control","no-store")
		resp,requestErr:=c.httpClient.Do(req)
		if requestErr!=nil {
			if attempt==1 || !transientTransportFailure(ctx,requestErr) {return nil,fmt.Errorf("Fabric Authentik API unavailable: %w",requestErr)}
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode!=http.StatusOK {
			return nil,fmt.Errorf("Fabric Authentik API returned HTTP %d",resp.StatusCode)
		}
		raw,readErr:=io.ReadAll(io.LimitReader(resp.Body,maxBodyBytes+1))
		if readErr!=nil || len(raw)>maxBodyBytes {return nil,errors.New("Fabric Authentik API response invalid or oversized")}
		return raw,nil
	}
	return nil,errors.New("Fabric Authentik API unavailable")
}

// collect enumerates and verifies EVERY page of a filtered result set. It
// fails closed for missing, contradictory or cyclic pagination metadata.
func (c *Client) collect(ctx context.Context, path string, filter url.Values) ([]json.RawMessage,error) {
	var result []json.RawMessage
	totalPages,expectedCount:=-1,-1
	for page:=1;page<=maxPages;page++ {
		params:=url.Values{}
		for k,values:=range filter {params[k]=append([]string(nil),values...)}
		params.Set("page",strconv.Itoa(page))
		params.Set("page_size",strconv.Itoa(pageSize))
		raw,err:=c.get(ctx,path,params)
		if err!=nil{return nil,err}
		var reply rawPage
		if err:=json.Unmarshal(raw,&reply);err!=nil || reply.Pagination==nil || reply.Results==nil {
			return nil,errors.New("invalid or incomplete Fabric Authentik pagination response")
		}
		meta:=reply.Pagination
		if meta.Current!=page || meta.Next<0 || meta.Count<0 ||
			meta.TotalPages<0 || meta.TotalPages>maxPages ||
			(meta.Count>0 && meta.TotalPages==0) ||
			(meta.Count==0 && meta.TotalPages>1) {
			return nil,errors.New("inconsistent Fabric Authentik pagination metadata")
		}
		if page==1 {
			totalPages,expectedCount=meta.TotalPages,meta.Count
		} else if meta.TotalPages!=totalPages || meta.Count!=expectedCount {
			return nil,errors.New("Authentik membership changed during pagination")
		}
		result=append(result,reply.Results...)
		if len(result)>expectedCount {
			return nil,errors.New("Authentik pagination contained duplicate or extra records")
		}
		if meta.Next==0 {
			if (meta.TotalPages!=0 && page!=meta.TotalPages) ||
				len(result)!=expectedCount {
				return nil,errors.New("incomplete Fabric Authentik membership inventory")
			}
			return result,nil
		}
		if meta.Next!=page+1 || page>=meta.TotalPages {
			return nil,errors.New("untrusted Fabric Authentik pagination cursor")
		}
	}
	return nil,errors.New("Fabric Authentik membership pagination bound exceeded")
}

func (c *Client) snapshotOnce(ctx context.Context, groupName string) (GroupMembership,error) {
	rawGroups,err:=c.collect(ctx,"/core/groups/",url.Values{
		"name":{groupName},"include_users":{"false"},
	})
	if err!=nil{return GroupMembership{},err}
	groupUUID:=""
	for _,raw:=range rawGroups {
		var group authentikGroup
		if err:=json.Unmarshal(raw,&group);err!=nil || !uuidPattern.MatchString(group.PK) {
			return GroupMembership{},errors.New("invalid Fabric Authentik group identity")
		}
		if group.Name!=groupName {continue} // defense against fuzzy upstream search
		if groupUUID!="" {return GroupMembership{},errors.New("ambiguous Fabric Authentik group name")}
		groupUUID=group.PK
	}
	if groupUUID=="" {return GroupMembership{},errors.New("Fabric tenant Authentik group not provisioned")}

	rawUsers,err:=c.collect(ctx,"/core/users/",url.Values{
		"groups_by_pk":{groupUUID},"include_groups":{"true"},
	})
	if err!=nil{return GroupMembership{},err}
	snapshot:=GroupMembership{Name:groupName,AuthentikGroupUUID:groupUUID,
		ActiveAuthentikUserUUIDs:[]string{}}
	seenPK:=map[int]bool{}
	seenUUID:=map[string]bool{}
	for _,raw:=range rawUsers {
		var user authentikUser
		if err:=json.Unmarshal(raw,&user);err!=nil ||
			user.PK<=0 || !uuidPattern.MatchString(user.UUID) ||
			user.IsActive==nil || user.Groups==nil {
			return GroupMembership{},errors.New("invalid Fabric Authentik user identity or membership")
		}
		if seenPK[user.PK] || seenUUID[user.UUID] {
			return GroupMembership{},errors.New("duplicate Fabric Authentik user in group inventory")
		}
		seenPK[user.PK]=true
		seenUUID[user.UUID]=true
		member:=false
		for _,id:=range user.Groups {
			if !uuidPattern.MatchString(id) {return GroupMembership{},errors.New("malformed Authentik group membership reference")}
			if id==groupUUID {member=true}
		}
		if !member {
			return GroupMembership{},errors.New("Authentik returned user outside requested group")
		}
		if *user.IsActive {
			snapshot.ActiveAuthentikUserUUIDs=append(snapshot.ActiveAuthentikUserUUIDs,user.UUID)
		}
	}
	sort.Strings(snapshot.ActiveAuthentikUserUUIDs)
	return snapshot,nil
}

// SnapshotGroupMembership double-reads the membership set to catch changes
// during pagination and rejects mismatches. Without an Authentik transactional
// snapshot revision, the Fabric reconciler MUST still revalidate freshness
// and enforce fail-closed authorization on any source outage.
func (c *Client) SnapshotGroupMembership(ctx context.Context, groupName string) (GroupMembership,error) {
	if !fabricGroupPattern.MatchString(groupName) || len(groupName)>128 {
		return GroupMembership{},errors.New("invalid Fabric-owned Authentik group name")
	}
	one,err:=c.snapshotOnce(ctx,groupName)
	if err!=nil{return GroupMembership{},err}
	two,err:=c.snapshotOnce(ctx,groupName)
	if err!=nil{return GroupMembership{},err}
	if !reflect.DeepEqual(one,two) {
		return GroupMembership{},errors.New("Fabric Authentik group membership changed during snapshot")
	}
	return two,nil
}
