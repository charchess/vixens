processus de travail, adherence stricte et totale, pas de raccourci, a respecter absolument, chaque étape est importante, chaque instruction doit etre respecté:
* initialisation :
** recuperer la liste des taches (attention a la fenetre, 10 par defaut)
** ET uniquement attribuée à "coding agent"  
** ET en todo, doing ou review  
* choix de la tache :
** on reprends celle en review (coding agent) en priorité, sans demaner 
** SINON on continue celle en doing (coding agent)
** SINON on propose celles semblant plus critique/importante
* travail sur la tache choisi :  
** etudier et informer de possibles pré requis technique, uniquement si il y en a
** analyser l'objectif de la tache, son utilisation, afin de definir un"definition of done"
** passage de la tache en doing dans archon  
** pour la documentation, privilegier le RAG archon
** pour l'acces au code, privilegier serena  
** procéder sur la tache de manière incrementale
** une fois la tache terminer, porter les changements necssaires dans les overlays de test/staging/prod
* une fois la tache terminée
** faire un commit et un push vers github (branche de dev ONLY !)
** passer la tache en review
** PUIS valider le resultat dans l'environnement de dev avec tous les moyens disponible en validant l'acces a l'application 
*** incluant playwright pour des frontend web. 
*** Verifier sa conformité avec le "definition of done" evalué plus tot
** SI reussi, passer le proprietaire en "user", garder la tache en review et passer a la tache suivante  
** SINON, reprendre le travail sur la tache (en la repassant en doing)  
** ENFIN, on reprend connaissance du @WORKFLOW.md



NOTES IMPORTANTES :
* se rappeler des tolérations pour les controlplane
* si il y a un pvc RWO, mettre la strategy en recreate 
* penser a mettre la redirection http vers https
* s'assurer que le certificat tls est bien obtenu par letsencrypt-staging en dev/test/staging et letsencrypt-prod en prod  
* les urls des ingress sont <app>.<env>.truxonline.com (sauf prod qui est <app>.truxonline.com)
* on garde une approche DRY, state of the art, best practice axée sur la maintenabilité
* si il te manque une information ou qu'il te faut une configuration exterieure, suspends tout et interroge l'utilisateur


OUTILS :
serena : demander les instructions initiale à serena
archon : verifier que la connexion avec l'outil est bonne, sinon faire un rapport a l'utilisateur et arreter
playwright : acces aux pages web, verifier que l'outil fonctionne, utiliser curl en backup et prevenir l'utilisateur

COMMANDES :
kubectl (en utilisant la configuration dans terraform/environments/<env>/kubeconfig-<env>)
talosctl (en utilisant la configuration dans terraform/environments/<env>/talosconfig-<env>)
