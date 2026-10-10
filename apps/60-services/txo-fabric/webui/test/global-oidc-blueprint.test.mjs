import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const blueprint = readFileSync(new URL(
  "../../../../03-security/authentik/base/configmap.yaml", import.meta.url), "utf8");
const worker = readFileSync(new URL(
  "../../../../03-security/authentik/base/deployment-worker.yaml", import.meta.url), "utf8");

// Static assertion of the reviewed source contract. GitHub CI separately
// builds/render-validates Kustomize manifests; only physical Authentik
// observation can prove the provider is actually published.
test("global WebUI browser provider is independent from tenant OIDC apps", () => {
  assert.match(blueprint, /txo-fabric-webui\.yaml: \|/);
  const global = blueprint.split("txo-fabric-webui.yaml: |")[1];
  assert.ok(global);
  assert.match(global, /id: txo-fabric-webui-provider/);
  assert.match(global, /client_id: txo-fabric-webui/);
  assert.match(global, /client_type: public/);
  assert.match(global, /grant_types:\n\s+- authorization_code/);
  assert.doesNotMatch(global, /- refresh_token|- implicit|- password/);
  assert.match(global, /slug: txo-fabric-webui/);
  assert.doesNotMatch(global, /hairem|indiba|client_secret|Secret|root_token/);
});

test("global OIDC claim is signed, immutable Authentik UUID, not a group grant", () => {
  const global = blueprint.split("txo-fabric-webui.yaml: |")[1];
  assert.match(global, /scope_name: txo_fabric_identity/);
  assert.match(global, /return \{"txo_fabric_user_uuid": str\(request\.user\.uuid\)\}/);
  assert.match(global, /include_claims_in_id_token: true/);
  assert.match(global, /signing_key: !Find/);
  assert.doesNotMatch(global, /groups:|members:|tenant_admin|can_chat|can_manage/);
  assert.match(global, /- !KeyOf txo-fabric-webui-immutable-human-uuid-scope/);
});

test("only an exact HTTPS callback is registered, no regex, wildcard or tenant user redirect", () => {
  const global = blueprint.split("txo-fabric-webui.yaml: |")[1];
  assert.match(global, /redirect_uris:\n\s+- matching_mode: strict\n\s+url: https:\/\/webui\.truxonline\.com\/auth\/callback/);
  assert.equal((global.match(/redirect_uri_type: authorization/g) ?? []).length, 1);
  assert.doesNotMatch(global, /matching_mode: regex/);
  assert.doesNotMatch(global, /http:\/\/|\/auth\/callback\$/);
});

test("AuthentiK worker mounts the versioned platform static blueprint", () => {
  assert.match(worker, /mountPath: \/blueprints\/vixens-fabric-webui\.yaml\n\s+subPath: txo-fabric-webui\.yaml/);
  assert.match(worker, /- name: blueprints\n\s+configMap:\n\s+name: authentik-config/);
  assert.match(worker, /mountPath: \/blueprints\/txo-fabric/);
});
