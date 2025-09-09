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
for machine in "${!machines[@]}"; do
    read -r ip1 _ <<< "${machines[$machine]}"
    read -r _ ip2 <<< "${machines[$machine]}"
    echo "resetting $machine ($ip2)"
    talosctl --talosconfig $TALOSCONFIG reset -n $ip2 -e $ip2 --system-labels-to-wipe STATE --system-labels-to-wipe EPHEMERAL  --graceful=true --reboot --wait=false
done

# on attend 5 secondes, pour etre sur
sleep 5

# on attend que les machine repasse sur le reseau de deploiement
echo "waiting for nodes..."
for machine in "${!machines[@]}"; do
    read -r ip1 _ <<< "${machines[$machine]}"
    read -r _ ip2 <<< "${machines[$machine]}"
    echo "waiting for $machine ($ip1)"
    wait_for $ip1
done
  
# on deploie les config
echo "applying configs"
read -p "Appuie sur Entrée pour continuer..."

for machine in "${!machines[@]}"; do
    read -r ip1 _ <<< "${machines[$machine]}"
    read -r _ ip2 <<< "${machines[$machine]}"
    echo "waiting for $machine ($ip1)"
    talosctl apply-config -i -n $ip1 -e $ip1 -f $SCRIPT_DIR/controlplane.yaml -p @$SCRIPT_DIR/vixens-$ENV-$machine.yaml
done


# attendre que le node 163 soit up and running
echo "waiting for node to bootstrap"
wait_for 192.168.208.163

# on attend 5 secondes de plus pour laisser le temps a la machine de se poser
sleep 5

read -p "Appuie sur Entrée pour continuer..."
# on bootstrap
talosctl -n 192.168.208.163 -e 192.168.208.163 bootstrap --talosconfig $TALOSCONFIG

sleep 10
# attendre ques les 3 nodes soient deployer
until \
  talosctl --talosconfig $TALOSCONFIG -n 192.168.208.163 -e 192.168.208.163 service machined | grep -q "HEALTH.*OK" && \
  talosctl --talosconfig $TALOSCONFIG -n 192.168.208.163 -e 192.168.208.163 service etcd     | grep -q "HEALTH.*OK" && \
  talosctl --talosconfig $TALOSCONFIG -n 192.168.208.163 -e 192.168.208.163 service kubelet  | grep -q "HEALTH.*OK" && \
  echo "waiting for etcd to be up"
do
	sleep 5
done

read -p "Appuie sur Entrée pour continuer..."
# on detaint, histoire d'etre tranquille
untaint-control-plane.sh

# patch dns foireux
kubectl --kubeconfig $KUBECONFIG apply -f $SCRIPT_DIR/manifests/fixes.yaml

read -p "Appuie sur Entrée pour continuer..."
# on a besoin d'une couche reseau CNI, on l'installe a la main :/
helm --kubeconfig $KUBECONFIG install cilium cilium/cilium --version 1.18.1  --namespace kube-system -f $SCRIPT_DIR/manifests/cilium-values.yaml

read -p "Appuie sur Entrée pour continuer..."
# on bootstrap argocd
$SCRIPT_DIR/../../clusters/vixens/argocd/bootstrap.sh dev
