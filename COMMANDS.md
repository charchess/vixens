plusieurs programmes sont disponibles pour les tests et le diagnostique :
* terraform
* talosctl
* kubectl
* argocd
* git (pour les commits)
* gh (pour les pull requests)
* yamllint

pour basculer les environnement de test et de dev :
* ssh administrator@192.168.200.67 c:\vms\talos.ps1 dev (ou test)

pour valider le cluster de <environnement>
* terraform destroy
* terraform apply
* tester les pings des ips
* recuperer les configuration kubeconfig dans /terraform/environments/<env>/kubeconfig-<env>
* verifier avec un kubectl que tout est ok
* authentifier argocd
* verifier les etats des differenttes applications
* verifier avec curl les differents site (http://traefik.<env>.truxonline.com, http://argocd.<env>.truxonline.com, http://whoami.<env>.truxonline.com)

Processus de commit de modifications :
* les modifications seront toujours commit sur la branche dev
* pour basculer des modifications en environnement de test, il faut commit en dev et faire un PR vers le test
* avant de commit:
** s'assurer que les yaml sont bien conforme (yamllint)
** s'assurer que les fichier terraform sont valides (terraform validate)
** s'assurer que l'ensemble du projet est coherent et conforme aux regles définies