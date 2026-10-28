// truenas-legacy-migrator creates a narrowly scoped temporary iSCSI bridge
// for one preserved legacy zvol. It never deletes or rewrites that zvol.
package main

import (
	"context"
	"fmt"
	"net"
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
	targetZvol := required("TARGET_ZVOL")
	if !strings.HasPrefix(targetZvol, "tank/k8s/vixens/iscsi/pvc-") {
		panic("TARGET_ZVOL is outside the canonical Vixens target zvol subtree")
	}
	snapshotName := required("SNAPSHOT_NAME")
	if !strings.HasPrefix(snapshotName, "vixens-migrate-") {
		panic("SNAPSHOT_NAME must start with vixens-migrate-")
	}
	name := required("MIGRATION_NAME")
	if !strings.HasPrefix(name, "vixens-migrate-") {
		panic("MIGRATION_NAME must start with vixens-migrate-")
	}
	portalAddress := required("ISCSI_PORTAL_ADDRESS")
	portalHost, _, err := net.SplitHostPort(portalAddress)
	if err != nil {
		panic(fmt.Errorf("ISCSI_PORTAL_ADDRESS must be host:port: %w", err))
	}
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
	// Preserve both exact zvol states before exposing a read-only bridge. The
	// snapshots are intentionally retained as the copy/cutover rollback proof.
	sourceSnapshot, err := c.CreateSnapshot(ctx, source, snapshotName, false)
	if err != nil {
		panic(fmt.Errorf("snapshot legacy source: %w", err))
	}
	targetSnapshot, err := c.CreateSnapshot(ctx, targetZvol, snapshotName, false)
	if err != nil {
		panic(fmt.Errorf("snapshot canonical target: %w", err))
	}

	portal, err := c.GetISCSIPortalByAddress(ctx, portalHost)
	if err != nil {
		panic(fmt.Errorf("find configured iSCSI portal: %w", err))
	}
	if portal == nil {
		panic(fmt.Errorf("no configured iSCSI portal listens on %s", portalHost))
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

	// Deliberately print only durable object identities; credentials and source
	// contents stay private. TrueNAS derives the target IQN from this name.
	fmt.Printf("created migration=%s source_snapshot=%s target_snapshot=%s target_id=%d extent_id=%d association_id=%d initiator_group_id=%d\n", name, sourceSnapshot.ID, targetSnapshot.ID, target.ID, extent.ID, association.ID, initiator.ID)
}
