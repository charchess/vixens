#!/bin/bash
ENV=dev

declare -A machines=(
    ["onyx"]="192.168.208.82 192.168.208.164"
    ["obsy"]="192.168.208.83 192.168.208.162" 
    ["opale"]="192.168.208.84 192.168.208.163"
)




TALOSCONFIG=~/vixens/talosconfig-$ENV
KUBECONFIG=~/vixens//kubeconfig-$ENV
SCRIPT_DIR=$(dirname "$(realpath "$0")")


wait_for() {
  local host=$1
  [[ -z $host ]] && { echo "Usage: wait_for <ip|hostname>"; return 1; }

  printf 'Attente de %s …' "$host"
  while ! ping -c1 -W1 "$host" &>/dev/null; do
    sleep 1
  done
  printf ' OK\n'
}

# on reset les 3 nodes de dev
echo "reinitialisation des nodes"
for machine in "${!machines[@]}"; do
    read -r ip1 _ <<< "${machines[$machine]}"
    read -r _ ip2 <<< "${machines[$machine]}"
    echo "...resetting $machine ($ip2)"
    talosctl --talosconfig $TALOSCONFIG reset -n $ip2 -e $ip2 --system-labels-to-wipe STATE --system-labels-to-wipe EPHEMERAL  --graceful=true --reboot --wait=false
done

# on attend 5 secondes, pour etre sur
sleep 5

# on attend que les machine repasse sur le reseau de deploiement
echo "on attend que les nodes ping sur les nouvelles ip..."
for machine in "${!machines[@]}"; do
    read -r ip1 _ <<< "${machines[$machine]}"
    read -r _ ip2 <<< "${machines[$machine]}"
    echo "...waiting for $machine ($ip1)"
    wait_for $ip1
done
  
# on deploie les config
read -p "Verifiez que les nodes sont en maintenance et pressez entrée"

echo "on applique les configurations (controlplane et vixens-$ENV"
for machine in "${!machines[@]}"; do
    read -r ip1 _ <<< "${machines[$machine]}"
    read -r _ ip2 <<< "${machines[$machine]}"
    echo "...applying for $machine ($ip1)"
    talosctl apply-config -i -n $ip1 -e $ip1 -f $SCRIPT_DIR/controlplane.yaml -p @$SCRIPT_DIR/vixens-$ENV-$machine.yaml
done


# attendre que le node 163 soit up and running
echo "On attend que le node a bootstrap soit up"
wait_for 192.168.208.163

# on attend 5 secondes de plus pour laisser le temps a la machine de se poser
sleep 5

read -p "Verifiez que les nodes ont leurs IP de prod et appuez sur entrée"
# on bootstrap
echo "on bootstrap"
talosctl -n 192.168.208.163 -e 192.168.208.163 bootstrap --talosconfig $TALOSCONFIG

sleep 10
# attendre ques les 3 nodes soient deployer
echo "on attend que le node de bootstrap soit ready (machined, etcd et kubelet)"
until \
  talosctl --talosconfig $TALOSCONFIG -n 192.168.208.163 -e 192.168.208.163 service machined | grep -q "HEALTH.*OK" && \
  talosctl --talosconfig $TALOSCONFIG -n 192.168.208.163 -e 192.168.208.163 service etcd     | grep -q "HEALTH.*OK" && \
  talosctl --talosconfig $TALOSCONFIG -n 192.168.208.163 -e 192.168.208.163 service kubelet  | grep -q "HEALTH.*OK" && \
  echo "waiting for etcd to be up"
do
	sleep 5
done

# on detaint, histoire d'etre tranquille
read -p "Une fois que les nodes sont healthy, appuyez sur entrée"
echo "on untaint les nodes"
untaint-control-plane.sh

# on a besoin d'une couche reseau CNI, on l'installe a la main :/
echo "on applique cilium"
helm --kubeconfig $KUBECONFIG install cilium cilium/cilium --version 1.18.1  --namespace kube-system -f $SCRIPT_DIR/manifests/cilium-values.yaml

sleep 10

echo "⏳ Attente Cilium..."
kubectl --kubeconfig ./kubeconfig-dev wait --for=condition=ready pod -l k8s-app=cilium -n kube-system --timeout=300s

sleep 10

# on bootstrap argocd
echo "on bootstrap argocd"
$SCRIPT_DIR/../bootstrap.sh dev

# patch du dns
echo "on patch le dns de coredns"
kubectl --kubeconfig $KUBECONFIG apply -f $SCRIPT_DIR/manifests/fixes.yaml

sleep 5
# parce qu'une fois, ca suffit pas
echo "on untaint les nodes (encore)"
untaint-control-plane.sh

sleep 5
# on installe la app of apps
kubectl --kubeconfig $KUBECONFIG  apply -f ~/vixens/clusters/vixens/argocd/01-vixens-dev-root.yaml
