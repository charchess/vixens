#!/bin/bash
# 🛠️ VIXENS ACTUAL STATE GENERATOR (Version "Emerald Shield" v4.0)
# Auteur : Selene (PM outragée mais efficace)
# Description : Extrait la réalité brute avec déduplication stricte et scoring Elite.
# Nécessite : kubectl, jq

REPORT_PATH="docs/reports/STATE-ACTUAL.md"
mkdir -p "$(dirname "$REPORT_PATH")"
VPA_TEMP="/tmp/vixens_vpa.json"
NODE_TEMP="/tmp/vixens_nodes.json"

echo "🔍 Selene scanne le chaos... Et garde tes pattes loin de moi, Charchess !"

# Extraction des données brutes
kubectl get vpa -A -o json > "$VPA_TEMP"
kubectl get nodes -o json > "$NODE_TEMP"

# En-tête du rapport
cat << EOF > "$REPORT_PATH"
# 📊 État Réel du Cluster - $(date '+%Y-%m-%d %H:%M:%S')

## 🖥️ Node Summary (Le métal de Charchess)
| Node Name | Role | CPU Cap | RAM Cap | OS | Kernel |
| :--- | :--- | :--- | :--- | :--- | :--- |
EOF

# Génération du résumé des nœuds
jq -r '.items[] | [
    .metadata.name,
    (if .metadata.labels | (has("node-role.kubernetes.io/control-plane") or has("node-role.kubernetes.io/master")) then "control-plane" else "worker" end),
    .status.capacity.cpu,
    .status.capacity.memory,
    .status.nodeInfo.osImage,
    .status.nodeInfo.kernelVersion
] | "| " + join(" | ") + " |"' "$NODE_TEMP" >> "$REPORT_PATH"

cat << EOF >> "$REPORT_PATH"

## 📦 Application Details (Détaillé)
| App | NS | CPU Req | CPU Lim | Mem Req | Mem Lim | VPA Target | Priority | Wave | Backup | QoS | Score |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
EOF

# Extraction et déduplication stricte par Namespace/App
kubectl get pods -A -o json | jq -r --slurpfile vpas "$VPA_TEMP" '
  [ .items[] 
    | select(.metadata.namespace | startswith("kube-") | not) 
    | select(.status.phase != "Succeeded") 
    | . as $pod
    | ($pod.metadata.labels["app.kubernetes.io/name"] // $pod.metadata.labels.app // $pod.metadata.name) as $app_name
    | {
        ns: .metadata.namespace,
        app: $app_name,
        cpu_req: (.spec.containers[0].resources.requests.cpu // "N/A"),
        cpu_lim: (.spec.containers[0].resources.limits.cpu // "N/A"),
        mem_req: (.spec.containers[0].resources.requests.memory // "N/A"),
        mem_lim: (.spec.containers[0].resources.limits.memory // "N/A"),
        prio: (.spec.priorityClassName // "N/A"),
        qos: .status.qosClass,
        wave: (.metadata.annotations["argocd.argoproj.io/sync-wave"] // "0"),
        backup: (if any(.spec.containers[]; .name | contains("litestream")) then "Active" else "None" end),
        vpa: ([$vpas[0].items[] | select(.metadata.name == $app_name or .spec.targetRef.name == $app_name)] | first)
      }
  ] 
  | sort_by(.ns + .app) 
  | unique_by(.ns + .app) 
  | .[] 
  | (0 
     + (if .qos == "Guaranteed" then 40 elif .qos == "Burstable" then 15 else 0 end)
     + (if .cpu_lim != "N/A" and .mem_lim != "N/A" then 20 else 0 end)
     + (if .prio != "N/A" and (.prio | contains("vixens") or contains("homelab")) then 20 else 0 end)
     + (if .backup == "Active" then 20 else 0 end)
    ) as $score
  | [
      "**" + .app + "**",
      .ns,
      .cpu_req,
      .cpu_lim,
      .mem_req,
      .mem_lim,
      (if .vpa then 
        ((.vpa.status.recommendation.containerRecommendations[0].target.cpu // "0") + " / " + 
         ((.vpa.status.recommendation.containerRecommendations[0].target.memory // "0") as $m | 
          if ($m | test("^[0-9]+$")) then (($m | tonumber / 1048576 | floor | tostring) + "Mi")
          elif ($m | endswith("Ki")) then (($m | sub("Ki$"; "") | tonumber / 1024 | floor | tostring) + "Mi")
          else $m end))
       else "None" end),
      .prio,
      .wave,
      .backup,
      .qos,
      ($score | tostring)
    ] 
  | "| " + join(" | ") + " |"' >> "$REPORT_PATH"

rm -f "$VPA_TEMP" "$NODE_TEMP"

if [ $? -eq 0 ]; then
    echo "✅ Rapport final (et sans doublons) généré : $REPORT_PATH"
else
    echo "❌ Erreur critique."
fi