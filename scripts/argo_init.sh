ARGOPASSWORD=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)
argocd login argocd.truxonline.com --username admin --password $ARGOPASSWORD
