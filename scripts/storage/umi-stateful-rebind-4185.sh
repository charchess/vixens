#!/usr/bin/env bash
# Issue #4185: owner-operated, one-volume-at-a-time UMI PV/PVC API rebind.
# No TrueNAS mutations. No automatic rollback. No automatic exec.
set -Eeuo pipefail
umask 077

VOLUME="${1:-}"
MODE="${2:---check}"
case "$VOLUME" in
  hermes)
    PV='pvc-5cb237dc-4a12-49ec-80c9-29477d89fa77'; NS='services'; PVC='hermes-data-truenas'
    APP='hermes-umi'; CLASS='truenas-iscsi-retain'; SIZE='80Gi' ;;
  postgres)
    PV='pvc-71395a3d-7d40-4303-bff3-0f3c23849fbf'; NS='hindsight'; PVC='data-hindsight-postgresql-0'
    APP='hindsight-umi'; CLASS='truenas-iscsi-retain'; SIZE='8Gi' ;;
  *) echo 'Usage: sudo bash umi-stateful-rebind-4185.sh {hermes|postgres} {--prepare|--check|--execute}' >&2; exit 64 ;;
esac
case "$MODE" in --prepare|--check|--execute) ;; *) echo 'Invalid mode' >&2; exit 64 ;; esac
[[ $EUID -eq 0 ]] || { echo 'STOP: run as root on UMI' >&2; exit 1; }
for tool in kubectl jq iscsiadm findmnt; do command -v "$tool" >/dev/null || { echo "STOP: missing $tool" >&2; exit 1; }; done
OLD='192.168.200.244:3260'
NEW='nas.truxonline.com:3260'
IQN="iqn.2005-10.org.freenas.ctl:csi-${PV}-umi"
BASE="${UMI_REBIND_ROOT:-/root/umi-iscsi-cutover}"
DIR="${BASE}/${VOLUME}"
PHASE='SETUP'
fail() { echo "STOP [$PHASE]: $*" >&2; exit 1; }
on_error() {
  local rc=$?
  echo "STOP [$PHASE]: command failed (exit $rc). NO AUTO-ROLLBACK. Preserve local files in $DIR; do not retry --execute blindly or resume workloads." >&2
  exit "$rc"
}
trap on_error ERR
# UMI is a k3s host. Do not inherit root's unrelated/default kubectl context.
KUBECONFIG_FILE="${UMI_REBIND_KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
[[ -r "$KUBECONFIG_FILE" ]] || fail "missing or unreadable UMI K3s kubeconfig: $KUBECONFIG_FILE"
k() { kubectl --kubeconfig="$KUBECONFIG_FILE" --request-timeout=15s "$@"; }

verify_live() {
  PHASE='VERIFY ORIGINAL PV/PVC'
  k get pv "$PV" -o json > "$DIR/live-pv.json"
  k -n "$NS" get pvc "$PVC" -o json > "$DIR/live-pvc.json"
  jq -e --arg pv "$PV" --arg ns "$NS" --arg pvc "$PVC" --arg old "$OLD" --arg iqn "$IQN" --arg class "$CLASS" --arg size "$SIZE" '
    .metadata.name == $pv and .metadata.deletionTimestamp == null and
    .status.phase == "Bound" and .spec.persistentVolumeReclaimPolicy == "Retain" and
    .spec.csi.driver == "org.democratic-csi.truenas-iscsi" and
    .spec.csi.volumeHandle == $pv and
    .spec.csi.volumeAttributes.portal == $old and
    .spec.csi.volumeAttributes.iqn == $iqn and
    (.spec.csi.volumeAttributes.lun | tostring) == "0" and
    .spec.claimRef.namespace == $ns and .spec.claimRef.name == $pvc and
    (.spec.claimRef.uid | type) == "string" and
    .spec.storageClassName == $class and .spec.capacity.storage == $size and
    .spec.accessModes == ["ReadWriteOnce"] and
    ([.metadata.finalizers[]?] | all(. == "kubernetes.io/pv-protection")) and
    (.metadata.finalizers | index("kubernetes.io/pv-protection")) != null and
    ([.metadata.annotations // {} | keys[] | select(test("deletion-secret"))] | length <= 2)
  ' "$DIR/live-pv.json" >/dev/null || fail 'PV identity/state/Retain/source/finalizers mismatch'
  jq -e --arg pv "$PV" --arg ns "$NS" --arg pvc "$PVC" --arg class "$CLASS" --arg size "$SIZE" '
    .metadata.name == $pvc and .metadata.namespace == $ns and
    .metadata.deletionTimestamp == null and .status.phase == "Bound" and
    .spec.volumeName == $pv and .spec.storageClassName == $class and
    .spec.resources.requests.storage == $size and
    .spec.volumeMode == "Filesystem" and .spec.accessModes == ["ReadWriteOnce"] and
    ([.metadata.finalizers[]?] | all(. == "kubernetes.io/pvc-protection")) and
    (.metadata.finalizers | index("kubernetes.io/pvc-protection")) != null
  ' "$DIR/live-pvc.json" >/dev/null || fail 'PVC identity/state/spec/finalizers mismatch'
  [[ "$(jq -r '.spec.claimRef.uid' "$DIR/live-pv.json")" == "$(jq -r '.metadata.uid' "$DIR/live-pvc.json")" ]] || fail 'PV claimRef UID != PVC UID'
  [[ "$(k get pv -o json | jq --arg id "$PV" '[.items[] | select(.spec.csi.volumeHandle? == $id)] | length')" == '1' ]] || fail 'duplicate CSI volumeHandle'
}

verify_quiescent() {
  PHASE='VERIFY ARGO/CONSUMERS/ATTACHMENTS/SESSIONS'
  k -n argocd get application "$APP" -o json | jq -e '.spec.syncPolicy.automated.enabled == false' >/dev/null || fail "$APP automated sync not disabled"
  [[ "$(k -n services get deployment hermes -o jsonpath='{.spec.replicas}')" == '0' ]] || fail 'Hermes not scaled 0'
  [[ "$(k -n hindsight get deployment hindsight-api -o jsonpath='{.spec.replicas}')" == '0' ]] || fail 'Hindsight API not scaled 0'
  [[ "$(k -n hindsight get statefulset hindsight-postgresql -o jsonpath='{.spec.replicas}')" == '0' ]] || fail 'Hindsight PostgreSQL not scaled 0'
  k get pods -A -o json | jq -e --arg ns "$NS" --arg pvc "$PVC" \
    '[.items[] | select(.metadata.namespace == $ns and any(.spec.volumes[]?; .persistentVolumeClaim.claimName? == $pvc))] | length == 0' >/dev/null || fail 'a Pod still references this PVC'
  k get volumeattachments.storage.k8s.io -o json | jq -e --arg pv "$PV" \
    '[.items[] | select(.spec.source.persistentVolumeName == $pv)] | length == 0' >/dev/null || fail 'VolumeAttachment remains'
  if iscsiadm -m session 2>/dev/null | grep -Fq "$IQN"; then fail 'manual iSCSI session still present; unmount+logout separately'; fi
  if findmnt -rn -o TARGET | grep -Fq "$PV"; then fail 'PV CSI mount still present'; fi
  if findmnt -rn -o TARGET | grep -Fq "${PV}/mount"; then fail 'PV mount still present'; fi
}

prepare() {
  PHASE='PREPARE READ-ONLY CANDIDATES'
  cp "$DIR/live-pv.json" "$DIR/old-pv.json"
  cp "$DIR/live-pvc.json" "$DIR/old-pvc.json"
  jq --arg portal "$NEW" --arg ns "$NS" --arg pvc "$PVC" '{
    apiVersion:"v1",kind:"PersistentVolume",
    metadata:{name:.metadata.name,labels:(.metadata.labels // {})},
    spec:(.spec | .persistentVolumeReclaimPolicy="Retain" |
      .claimRef={namespace:$ns,name:$pvc} | .csi.volumeAttributes.portal=$portal)
  }' "$DIR/old-pv.json" > "$DIR/new-pv.json"
  jq --arg pv "$PV" '{
    apiVersion:"v1",kind:"PersistentVolumeClaim",
    metadata:{namespace:.metadata.namespace,name:.metadata.name,labels:(.metadata.labels // {}),
      annotations: (if (.metadata.annotations // {}) | has("argocd.argoproj.io/tracking-id")
        then {"argocd.argoproj.io/tracking-id":.metadata.annotations["argocd.argoproj.io/tracking-id"]} else {} end)},
    spec:(.spec | .volumeName=$pv)
  }' "$DIR/old-pvc.json" > "$DIR/new-pvc.json"
}

verify_candidates() {
  PHASE='VERIFY SAVED BASELINE & CANDIDATES'
  for f in old-pv.json old-pvc.json new-pv.json new-pvc.json; do [[ -s "$DIR/$f" ]] || fail "missing $f; first run --prepare"; done
  jq -n -e --slurpfile saved "$DIR/old-pv.json" --slurpfile live "$DIR/live-pv.json" '
    $saved[0].spec == $live[0].spec and $saved[0].metadata.uid == $live[0].metadata.uid' >/dev/null || fail 'live PV drift since preparation'
  jq -n -e --slurpfile saved "$DIR/old-pvc.json" --slurpfile live "$DIR/live-pvc.json" '
    $saved[0].spec == $live[0].spec and $saved[0].metadata.uid == $live[0].metadata.uid' >/dev/null || fail 'live PVC drift since preparation'
  jq -n -e --slurpfile old "$DIR/old-pv.json" --slurpfile new "$DIR/new-pv.json" --arg portal "$NEW" --arg ns "$NS" --arg pvc "$PVC" --arg pv "$PV" '
    ($old[0].spec | .persistentVolumeReclaimPolicy="Retain" |
      .claimRef={namespace:$ns,name:$pvc} | .csi.volumeAttributes.portal=$portal) as $expected |
    $new[0].apiVersion == "v1" and $new[0].kind == "PersistentVolume" and
    ($new[0].metadata | keys | sort) == ["labels","name"] and
    $new[0].metadata.name == $pv and $new[0].metadata.labels == ($old[0].metadata.labels // {}) and
    $new[0].spec == $expected
  ' >/dev/null || fail 'new PV differs from allowed source delta'
  jq -n -e --slurpfile old "$DIR/old-pvc.json" --slurpfile new "$DIR/new-pvc.json" --arg pv "$PV" --arg ns "$NS" --arg pvc "$PVC" '
    (if ($old[0].metadata.annotations // {}) | has("argocd.argoproj.io/tracking-id")
     then {"argocd.argoproj.io/tracking-id":$old[0].metadata.annotations["argocd.argoproj.io/tracking-id"]} else {} end) as $allowedann |
    $new[0].apiVersion == "v1" and $new[0].kind == "PersistentVolumeClaim" and
    ($new[0].metadata | keys | sort) == ["annotations","labels","name","namespace"] and
    $new[0].metadata.namespace == $ns and $new[0].metadata.name == $pvc and
    $new[0].metadata.labels == ($old[0].metadata.labels // {}) and
    $new[0].metadata.annotations == $allowedann and
    $new[0].spec == ($old[0].spec | .volumeName=$pv)
  ' >/dev/null || fail 'new PVC differs from allowed static prebinding'
}

mkdir -p "$DIR"
chmod 700 "$DIR"
verify_live
if [[ "$MODE" == '--prepare' ]]; then
  prepare
  verify_candidates
  echo "PREPARED $VOLUME: $DIR/{old-pv,old-pvc,new-pv,new-pvc}.json ; no cluster writes"
  exit 0
fi
verify_candidates
verify_quiescent
printf 'CHECK OK: %s %s Bound/Retain; preserved original CSI identity; Argo paused; no Pods/VA/session\n' "$NS" "$PVC"
if [[ "$MODE" == '--check' ]]; then
  echo 'NO MUTATION. This check only authorizes operator review, not automatic execution.'
  exit 0
fi

read -r -p "Type EXACT PV suffix ${PV#pvc-} to confirm deleting/recreating API PV/PVC objects (NO TrueNAS destruction): " CONFIRM
[[ "$CONFIRM" == "${PV#pvc-}" ]] || fail 'confirmation did not match'
PHASE='BACKUP BEFORE CUTOVER'
cp "$DIR/live-pv.json" "$DIR/cutover-old-pv.json"
cp "$DIR/live-pvc.json" "$DIR/cutover-old-pvc.json"
[[ -s "$DIR/cutover-old-pv.json" && -s "$DIR/cutover-old-pvc.json" ]] || fail 'backup missing'
verify_quiescent
PHASE='DELETE ONLY ORIGINAL PVC API OBJECT'
k -n "$NS" delete pvc "$PVC" --wait=false
PHASE='WAIT ORIGINAL PVC DELETED'
k -n "$NS" wait --for=delete "pvc/$PVC" --timeout=120s
PHASE='WAIT PV RELEASED'
released=false
for i in $(seq 1 30); do
  phase=$(k get pv "$PV" -o jsonpath='{.status.phase}')
  if [[ "$phase" == 'Released' ]]; then released=true; break; fi
  sleep 2
done
[[ "$released" == true ]] || fail 'PV did not reach Released'
[[ "$(k get pv "$PV" -o jsonpath='{.spec.persistentVolumeReclaimPolicy}')" == 'Retain' ]] || fail 'Retain not preserved'
verify_quiescent
PHASE='DELETE ONLY RETAINED RELEASED PV API OBJECT'
k delete pv "$PV" --wait=false
PHASE='WAIT ORIGINAL PV DELETED'
k wait --for=delete "pv/$PV" --timeout=120s
PHASE='ENSURE NO ALTERNATE PV USES SAME VOLUME HANDLE'
[[ "$(k get pv -o json | jq --arg pv "$PV" '[.items[]|select(.spec.csi.volumeHandle? == $pv)]|length')" == '0' ]] || fail 'another PV already claims CSI volumeHandle'
PHASE='CREATE STATIC REPLACEMENT PV'
k create -f "$DIR/new-pv.json"
[[ "$(k get pv "$PV" -o jsonpath='{.spec.csi.volumeAttributes.portal}')" == "$NEW" ]] || fail 'new portal mismatch'
[[ "$(k get pv "$PV" -o jsonpath='{.spec.persistentVolumeReclaimPolicy}')" == 'Retain' ]] || fail 'new PV not Retain'
PHASE='CREATE PREBOUND PVC'
k -n "$NS" create -f "$DIR/new-pvc.json"
PHASE='VERIFY NEW BINDING'
bound=false
for i in $(seq 1 30); do
  pvcphase=$(k -n "$NS" get pvc "$PVC" -o jsonpath='{.status.phase}')
  pvphase=$(k get pv "$PV" -o jsonpath='{.status.phase}')
  if [[ "$pvcphase" == 'Bound' && "$pvphase" == 'Bound' ]]; then bound=true; break; fi
  sleep 2
done
[[ "$bound" == true ]] || fail 'new objects did not bind; do NOT retry'
newuid=$(k -n "$NS" get pvc "$PVC" -o jsonpath='{.metadata.uid}')
claimuid=$(k get pv "$PV" -o jsonpath='{.spec.claimRef.uid}')
[[ -n "$newuid" && "$newuid" == "$claimuid" ]] || fail 'new UID binding mismatch'
[[ "$newuid" != "$(jq -r '.metadata.uid' "$DIR/old-pvc.json")" ]] || fail 'UID unexpectedly reused'
[[ "$(k -n "$NS" get pvc "$PVC" -o jsonpath='{.spec.volumeName}')" == "$PV" ]] || fail 'wrong PV reference'
[[ "$(k get pv "$PV" -o jsonpath='{.spec.csi.volumeHandle}')" == "$PV" ]] || fail 'wrong CSI handle'
PHASE='DONE'
echo "SUCCESS $VOLUME: $NS/$PVC rebound to $NEW, original CSI handle $PV retained"
echo "NEW_PVC_UID=$newuid"
echo 'Do not resume workloads or Argo auto-sync. Verify next resource and GitOps desired state.'
