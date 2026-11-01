# Fabric RWX — production acceptance after native PVC promotion

Scope: #4400, #4357, #4398. Read-only gate; does **not** authorize a migration,
delete/rebind of PV/PVC, restart, forced sync, or removal of rollback volumes.

On 2026-11-01, GitHub's annotated `prod-stable` reference changed to
`prod-v2026.11.4410` / commit `d558bf9`. GitHub promotion **does not**
prove cluster convergence or data integrity. The last physical observation
before this promotion had eight distinct retained RWX PVC names ending
`-synelia` mounted by hAIrem and Indiba agents. Two unsuffixed
claims were also observed Pending. Treat them as real data/provisioning risks.

## Safe structural preflight

On a host with read-only `kubectl get` access to the desired cluster, from
the root of the checked-out release:

```bash
python3 scripts/validation/fabric_rwx_acceptance.py \
  > fabric-rwx-structural-evidence.json
status=$?
printf 'RWX structural gate exit=%s\n' "$status"
```

The program performs **only** `kubectl get pvc`/`pv`/`pods -o json`,
without inspecting any Secret or reading files on the PVCs. It derives
canonical names from hAIrem and Indiba's declared workspace scopes (no
hardcoded physical PVC identities), checks ownership, namespace and exact
PVC UID→PV claimRef linkage, StorageClass, CSI and Retain, checks pod mounts
and read-only reference scopes, and reports lingering legacy claim names
without deleting them. Offline unit tests use synthetic PVC/PV/Pod JSON.

- Exit **2 / FAIL**: violated or missing structural resource; halt acceptance
  and diagnose before touching old volumes.
- Exit **3 / PENDING**: no structural violations observed; **still not a
  release sign-off** because no byte-level read/write proof was performed.
- `INFO` for old claims: protect for rollback; do not delete.
- `PENDING` for unobserved consumers: never infer functional access.

You can rerun offline with `--snapshot` using a JSON dictionary of
`pvcs` and `pods` keyed by tenant slug, and a `pvs` list.

## Mandatory manual functional acceptance (do not simulate)

1. ArgoCD `txo-fabric` actual revision/Sync/Health and deployed operator
   container image SHA **observed**; ensure it matches the promoted release.
2. Identify each old `*-synelia` PVC/PV/UID and each new canonical binding
   in the report. Record that no old source is spuriously recreated, and do
   not treat `Bound` as a file-copy success.
3. Validate known pre-existing canary files/folder listing, byte hashes or
   an independently verified backup restoration for **each of 8 scopes**.
   Use controlled authorized agent sessions; don't leak tenant data in
   public GitHub issues. Check shared RWX collaborative writes and reference
   read-only behavior, plus forbidden foreign tenant/group/user access.
4. Observe at least two reconciles (including a controller restart where
   separately approved) with unchanged PV identities, no extra Jobs,
   no missing mounts and no unexpected Deployment rollout.
5. Retain all previous PV/PVC data for rollback until **separate** review
   of authorized deletion. A Git revert cannot recover overwritten bytes.
6. Inspect independent gates: OpenBao Fabric stateful storage (#4406),
   LiteLLM schema migration evidence (#4387), clean-room tenant bootstrap
   (#4398), UMI Hermes recovery policy (#4405) and mail import Secret
   ownership (#4407). Do **not** close the parent #4357 until every required
   runtime criterion is explicitly proven.

Attach a redacted report and the separate functional evidence/check result
to #4400 and #4357, noting the exact time, Argo revision and image. Redact
hostnames, CSI backing handles, sensitive content and user identifiers when
sharing outside authorized operations. Do not mark PASS from the tool alone.
