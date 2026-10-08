package controller

import (
 "context"
 "os"
 "os/exec"
 "os/user"
 "path/filepath"
 "strings"
 "testing"

 appsv1 "k8s.io/api/apps/v1"
 "k8s.io/apimachinery/pkg/types"
 "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Reproduces the unwritable install.lock/config backup modes on a retained
// private workspace, and protects non-runtime personal state from repair.
func TestHermesRetainedPVCBootstrapPermissions(t *testing.T) {
 scheme := testScheme(t)
 tenant := testTenant()
 agent := testAgentIdentity()
 agent.UID = types.UID("bootstrap-test-agent")
 profile := testRuntimeProfile()
 c := fake.NewClientBuilder().WithScheme(scheme).Build()
 r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
 ns := tenantNamespace(tenant.Name)
 if err := r.ensureDeploymentWithIntegrations(context.Background(), agent, tenant, profile, ns,
  "model-secret-uid", "model-rev", effectiveToolsetPolicy{Revision: "tools-v1"}, integrationResolution{}); err != nil {
  t.Fatal(err)
 }
 var dep appsv1.Deployment
 if err := c.Get(context.Background(), types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: ns}, &dep); err != nil {
  t.Fatal(err)
 }
 var script string
 for _, init := range dep.Spec.Template.Spec.InitContainers {
  if init.Name == "bootstrap-profile" {
   script = init.Command[2]
  }
 }
 if script == "" { t.Fatal("bootstrap-profile missing") }
 for _, fragment := range []string{
  "\"${HERMES_HOME}/backups\" \"${HERMES_HOME}/installs\"",
  "! -user hermes -o ! -group hermes",
  "-exec chown hermes:hermes -- {} +",
  "-xdev -type f ! -perm -u+rw",
  "-xdev -type d ! -perm -u+wx",
  "if [ -L \"$entry\" ]",
  "exit 78",
 } {
  if !strings.Contains(script, fragment) { t.Errorf("bootstrap missing %q", fragment) }
 }
 if strings.Contains(script, "chown -R hermes:hermes \"${HERMES_HOME}\"") {
  t.Fatal("must not recursively chown entire private PVC")
 }
 if b, err := exec.Command("sh", "-n", "-c", script).CombinedOutput(); err != nil {
  t.Fatalf("bootstrap shell invalid: %v %s", err, b)
 }

 // Isolate exactly the repair block, replacing the image's hermes identity
 // with the CI runner identity so the test needs no privileged container.
 start := strings.Index(script, "# The init container runs as root;")
 end := strings.Index(script, "if [ -e \"${HERMES_HOME}/config.yaml\" ]; then", start)
 if start < 0 || end <= start { t.Fatal("permission block not found") }
 repair := script[start:end]
 me, err := user.Current()
 if err != nil { t.Fatal(err) }
 g, err := exec.Command("id", "-gn").Output()
 if err != nil { t.Fatal(err) }
 group := strings.TrimSpace(string(g))
 for _, swap := range [][2]string{
  {"hermes:hermes", me.Username+":"+group},
  {"-user hermes", "-user "+me.Username},
  {"-group hermes", "-group "+group},
  {"-o hermes -g hermes", "-o "+me.Username+" -g "+group},
 } {
  repair = strings.ReplaceAll(repair, swap[0], swap[1])
 }
 run := func(home string) (string,error) {
  cmd := exec.Command("sh", "-c", repair)
  cmd.Env = append(os.Environ(), "HERMES_HOME="+home, "TXO_LEGACY_PROFILE_ADOPTION=false")
  b, err := cmd.CombinedOutput()
  return string(b),err
 }

 home := t.TempDir()
 lock := filepath.Join(home,"installs","hash",".install.lock")
 backup := filepath.Join(home,"backups","config","config.yaml.good.1")
 personal := filepath.Join(home,"skills","personal.txt")
 external := filepath.Join(t.TempDir(),"external.txt")
 for _, path := range []string{lock,backup,personal,external} {
  if err := os.MkdirAll(filepath.Dir(path),0750); err != nil { t.Fatal(err) }
  if err := os.WriteFile(path,[]byte("redacted test fixture"),0400); err != nil { t.Fatal(err) }
  if err := os.Chmod(path,0400); err != nil { t.Fatal(err) }
 }
 if err := os.Symlink(external,filepath.Join(home,"installs","hash","link")); err != nil { t.Fatal(err) }
 if output,err := run(home); err != nil { t.Fatalf("repair failed: %v %s",err,output) }
 for _, path := range []string{lock,backup} {
  info,err := os.Stat(path);if err != nil { t.Fatal(err) }
  if info.Mode().Perm()&0200 == 0 { t.Errorf("runtime file is still unwritable: %s",path) }
 }
 for _, path := range []string{personal,external} {
  info,err := os.Stat(path);if err != nil { t.Fatal(err) }
  if info.Mode().Perm()!=0400 { t.Errorf("unmanaged file altered: %s",path) }
 }

 other := t.TempDir()
 if err := os.Symlink(t.TempDir(),filepath.Join(other,"backups"));err!=nil { t.Fatal(err) }
 output,err := run(other)
 if err==nil || !strings.Contains(output,"refusing symlinked") {
  t.Fatalf("symlink state root not rejected: %v %s",err,output)
 }
}
