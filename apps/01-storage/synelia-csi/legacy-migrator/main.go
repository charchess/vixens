// truenas-legacy-migrator creates a narrowly scoped temporary iSCSI bridge
// for one preserved legacy zvol. It never deletes or rewrites that zvol.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/truenas/truenas-csi/pkg/client"
)

func required(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		panic("missing required environment variable " + name)
	}
	return value
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	source := required("SOURCE_ZVOL")
	if !strings.HasPrefix(source, "tank/k8s/vixens/iscsi/vols/pvc-") {
		panic("SOURCE_ZVOL is outside the preserved Vixens legacy zvol subtree")
	}
	name := required("MIGRATION_NAME")
	if !strings.HasPrefix(name, "vixens-migrate-") {
		panic("MIGRATION_NAME must start with vixens-migrate-")
	}
	portalAddress := required("ISCSI_PORTAL_ADDRESS")
	initiators := strings.Fields(required("ALLOWED_INITIATORS"))
	if len(initiators) == 0 {
		panic("ALLOWED_INITIATORS must contain Kubernetes node IQNs")
	}

	tlsConfig, err := client.NewTLSConfig(false, "")
	if err != nil {
		panic(fmt.Errorf("configure TrueNAS TLS: %w", err))
	}
	c := client.New(client.Config{
		URL:                required("TRUENAS_URL"),
		APIKey:             required("TRUENAS_API_KEY"),
		TLSConfig:          tlsConfig,
		InsecureSkipVerify: false,
		CallTimeout:        30 * time.Second,
	})
	defer c.Close()
	if err := c.Connect(ctx); err != nil {
		panic(fmt.Errorf("connect TrueNAS: %w", err))
	}

	portal, err := c.GetISCSIPortalByAddress(ctx, portalAddress)
	if err != nil {
		panic(fmt.Errorf("find configured iSCSI portal: %w", err))
	}
	initiator, err := c.CreateISCSIInitiator(ctx, &client.ISCSIInitiatorCreateOptions{
		Initiators: initiators,
		Comment:    "temporary migration bridge " + name,
	})
	if err != nil {
		panic(fmt.Errorf("create restricted initiator group: %w", err))
	}
	group := client.ISCSITargetGroup{Portal: portal.ID, Initiator: initiator.ID}
	target, err := c.CreateISCSITargetWithGroup(ctx, name, "temporary migration bridge "+name, group)
	if err != nil {
		_ = c.DeleteISCSIInitiator(ctx, initiator.ID)
		panic(fmt.Errorf("create temporary target: %w", err))
	}
	extent, err := c.CreateISCSIExtent(ctx, name, "zvol/"+source, 512)
	if err != nil {
		_ = c.DeleteISCSITarget(ctx, target.ID, nil)
		_ = c.DeleteISCSIInitiator(ctx, initiator.ID)
		panic(fmt.Errorf("create temporary extent: %w", err))
	}
	association, err := c.CreateISCSITargetExtent(ctx, target.ID, extent.ID, 0)
	if err != nil {
		_ = c.DeleteISCSIExtent(ctx, extent.ID, nil)
		_ = c.DeleteISCSITarget(ctx, target.ID, nil)
		_ = c.DeleteISCSIInitiator(ctx, initiator.ID)
		panic(fmt.Errorf("associate temporary target and extent: %w", err))
	}

	// Deliberately print IDs only; credentials, initiator IQNs, and source contents stay private.
	fmt.Printf("created migration=%s target_id=%d extent_id=%d association_id=%d initiator_group_id=%d\n", name, target.ID, extent.ID, association.ID, initiator.ID)
}
