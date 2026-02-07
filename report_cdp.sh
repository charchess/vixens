#!/bin/bash
date > /root/vixens/report_cdp.txt
echo "---" >> /root/vixens/report_cdp.txt
bd list --limit 0 --pretty --json --all >> /root/vixens/report_cdp.txt
echo "---" >> /root/vixens/report_cdp.txt
kubectl get all -A >> /root/vixens/report_cdp.txt
echo "---" >> /root/vixens/report_cdp.txt
tree -up >> /root/vixens/report_cdp.txt
