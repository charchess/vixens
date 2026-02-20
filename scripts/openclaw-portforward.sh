#!/bin/bash

export KUBECONFIG=/home/charchess/vixens/.secrets/prod/kubeconfig-prod

POD_NAME=$(kubectl -n services get pods -l app=openclaw -o jsonpath='{.items[0].metadata.name}')

echo "Starting port-forward for $POD_NAME..."

while true; do
    kubectl -n services port-forward pod/$POD_NAME 18789:18789 &
    PF_PID=$!
    
    sleep 2
    
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:18789/ | grep -q "200"; then
        echo "Port-forward is up!"
    else
        echo "Port-forward failed, retrying..."
        kill $PF_PID 2>/dev/null
        sleep 2
        continue
    fi
    
    wait $PF_PID
    echo "Port-forward died, restarting..."
    sleep 2
done
