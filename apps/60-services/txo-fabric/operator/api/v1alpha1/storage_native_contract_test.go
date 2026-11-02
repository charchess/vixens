package v1alpha1

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNativeStorageSpecDoesNotExposeRetiredCSIMigrationFields(t *testing.T) {
	cases := []struct {
		name string
		spec any
	}{
		{
			name: "agent runtime storage",
			spec: AgentRuntimeStorageBinding{RetentionPolicy: "Retain"},
		},
		{
			name: "tenant shared workspace",
			spec: TenantWorkspaceSpec{ProfileRef: "workspace-shared", RetentionPolicy: "Retain"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.spec)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(data, &fields); err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{"migration", "claimSuffix"} {
				if _, exists := fields[forbidden]; exists {
					t.Fatalf("deprecated CSI spec field %q remains in %s", forbidden, data)
				}
			}
			if strings.Contains(string(data), "targetClaimSuffix") || strings.Contains(string(data), "copyJobName") {
				t.Fatalf("migration cutover input remains in native storage spec: %s", data)
			}
		})
	}
}

func TestRetiredCSISpecDoesNotDropStoredAuditStatusContract(t *testing.T) {
	for _, tc := range []struct {
		name string
		status any
		want string
	}{
		{"agent storage evidence", AgentIdentityStatus{RuntimeStorageMigration: &RuntimeStorageMigrationAttestationStatus{}}, "runtimeStorageMigration"},
		{"workspace evidence", TenantBundleStatus{WorkspaceMigration: []WorkspaceMigrationAttestationStatus{{}}}, "workspaceMigration"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.status)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), "\"" + tc.want + "\"") {
				t.Fatalf("retained audit status %q disappeared: %s", tc.want, data)
			}
		})
	}
}
