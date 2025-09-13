#!/bin/bash
# Mega Pepperoni Talos Cluster Deploy
# ./deploy_vixen-dev.sh [dev|prod] [DEBUG=true]

set -euo pipefail

ENV=${1:-dev}
DEBUG=${DEBUG:-false}
DEBUG=true

ROOT_DIR=~/vixens
TALOSCONFIG="$ROOT_DIR/environments/$ENV/talosconfig"
KUBECONFIG="$ROOT_DIR/environments/$ENV/kubeconfig"
SCRIPT_DIR="$(dirname "$(realpath "$0")")"

declare -A machines=(
    ["onyx"]="192.168.208.82 192.168.208.164"
    ["obsy"]="192.168.208.83 192.168.208.162"
    ["opale"]="192.168.208.84 192.168.208.163"
)

log() { echo "[INFO] $*"; }
debug() { [ "$DEBUG" = true ] && echo "[DEBUG] $*"; }
timer_start() { TIMER_START=$(date +%s); }
timer_end() { echo "⏱ Durée: $(( $(date +%s) - TIMER_START ))s"; }

wait_for_ip() {
    local ip=$1
    printf '⏳ Attente de %s …' "$ip"
    until ping -c1 -W1 "$ip" &>/dev/null; do sleep 1; done
    echo " OK"
}

run_cmd() {
    debug "$*"
    eval "$@"
}

# -------------------------------
# 1️⃣ Reset nodes (conditionnel)
# -------------------------------
log "⚙️ Vérification / reset des nodes..."
for machine in "${!machines[@]}"; do
    read -r deploy_ip bootstrap_ip <<< "${machines[$machine]}"
    if ping -c1 -W1 "$deploy_ip" &>/dev/null; then
        log "...$machine ($deploy_ip) déjà up → pas de reset"
    else
        log "...reset $machine ($bootstrap_ip)"
        run_cmd "talosctl --talosconfig $TALOSCONFIG reset -n $bootstrap_ip -e $bootstrap_ip \
                 --system-labels-to-wipe STATE --system-labels-to-wipe EPHEMERAL \
                 --graceful=false --wait=false --reboot" || true
    fi
done

sleep 5

# -------------------------------
# 2️⃣ Attente que nodes ping
# -------------------------------
log "⏳ Attente que les nodes ping sur IP de déploiement..."
for machine in "${!machines[@]}"; do
    read -r deploy_ip _ <<< "${machines[$machine]}"
    wait_for_ip "$deploy_ip"
done

sleep 5

# -------------------------------
# 3️⃣ Appliquer configs Talos
# -------------------------------
log "⚙️ Application des configurations (controlplane + vixens-$ENV)..."
for machine in "${!machines[@]}"; do
    read -r deploy_ip bootstrap_ip <<< "${machines[$machine]}"
    log "...appliquer $machine ($deploy_ip)"
    run_cmd "talosctl apply-config -i -n $deploy_ip -e $deploy_ip \
             -f $SCRIPT_DIR/controlplane.yaml -p @$SCRIPT_DIR/vixens-$ENV-$machine.yaml"
done

# -------------------------------
# 4️⃣ Pre-bootstrap check
# -------------------------------
bootstrap_ip="${machines[opale]##* }"  # dernier IP = bootstrap
wait_for_ip "$bootstrap_ip"

log "⏳ Vérification que le node $bootstrap_ip écoute sur 50000..."
until nc -z -w2 "$bootstrap_ip" 50000; do sleep 2; done
log "✅ Node $bootstrap_ip reachable on 50000"

log "⏳ Scrutation dmesg pour 'bootstrap' ou 'first node'..."
until talosctl -n "$bootstrap_ip" -e "$bootstrap_ip" dmesg | grep -Eq "bootstrap|first node"; do
    sleep 5
done
log "✅ Message dmesg bootstrap détecté"

# -------------------------------
# 5️⃣ Bootstrap cluster
# -------------------------------
log "🚀 Bootstrap du cluster sur $bootstrap_ip"
timer_start
run_cmd "talosctl -n $bootstrap_ip -e $bootstrap_ip bootstrap --talosconfig $TALOSCONFIG"
timer_end

# -------------------------------
# 6️⃣ Wait services Talos post-bootstrap
# -------------------------------
log "⏳ Attente que machined, etcd et kubelet soient HEALTHY..."
until \
  talosctl --talosconfig "$TALOSCONFIG" -n "$bootstrap_ip" -e "$bootstrap_ip" service machined | grep -q "HEALTH.*OK" && \
  talosctl --talosconfig "$TALOSCONFIG" -n "$bootstrap_ip" -e "$bootstrap_ip" service etcd     | grep -q "HEALTH.*OK" && \
  talosctl --talosconfig "$TALOSCONFIG" -n "$bootstrap_ip" -e "$bootstrap_ip" service kubelet  | grep -q "HEALTH.*OK"; do
    sleep 5
done
log "✅ Services Talos healthy"

# -------------------------------
# 6️⃣bis Attente control plane Kubernetes
# -------------------------------
log "⏳ Attente kube-apiserver sur 6443..."
until nc -z -w2 "$bootstrap_ip" 6443; do sleep 2; done
log "✅ kube-apiserver écoute sur 6443"

log "⏳ Vérification services control-plane (kube-apiserver, scheduler, controller-manager)..."
until \
  kubectl --kubeconfig "$KUBECONFIG" get pods -n kube-system -l component=kube-apiserver   2>/dev/null | grep -q "Running" && \
  kubectl --kubeconfig "$KUBECONFIG" get pods -n kube-system -l component=kube-scheduler   2>/dev/null | grep -q "Running"; do
    sleep 5
done
log "✅ Control plane Kubernetes prêt"

# -------------------------------
# 7️⃣ Détaint nodes
# -------------------------------
log "⚙️ Détaint des nodes..."
run_cmd "untaint-control-plane.sh"

# -------------------------------
# 8️⃣ Installer Cilium
# -------------------------------
log "⚙️ Installation Cilium..."
run_cmd "helm --kubeconfig $KUBECONFIG install cilium cilium/cilium \
         --version 1.18.1 --namespace kube-system -f $SCRIPT_DIR/manifests/cilium-values.yaml"

log "⏳ Attente pods Cilium..."
kubectl --kubeconfig $KUBECONFIG wait --for=condition=ready pod -l k8s-app=cilium -n kube-system --timeout=300s
log "✅ Cilium ready"

# -------------------------------
# 9️⃣ Bootstrap ArgoCD
# -------------------------------
log "⚙️ Bootstrap ArgoCD..."
run_cmd "$SCRIPT_DIR/../bootstrap.sh $ENV"

# -------------------------------
# 🔧 Patch DNS CoreDNS
# -------------------------------
log "⚙️ Patch CoreDNS..."
kubectl --kubeconfig $KUBECONFIG apply -f $SCRIPT_DIR/manifests/fixes.yaml

log "✅ Cluster $ENV déployé et prêt pour GitOps !"
