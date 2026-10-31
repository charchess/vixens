package controller

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedEnrollmentSchemaExcludesStaticHumanIdentifiers(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	crd, err := os.ReadFile(filepath.Join(root, "config/crd/bases/fabric.truxonline.io_tenantbundles.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"enrollmentIntents:", "authentikUserUUID:", "globalOidcIssuer:", "globalOidcSubject:", "oidcSubject:"} {
		if strings.Contains(string(crd), forbidden) {
			t.Fatalf("generated CRD permits static human enrollment identifier %q", forbidden)
		}
	}
}
