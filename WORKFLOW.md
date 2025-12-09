processus de travail:
* initialisation :
** recuperer la liste des taches (attention a la fenetre, 10 par defaut)
** ET uniquement attribuée à "coding agent"  
** ET en todo, doing ou review  
* choix de la tache :
** on reprends celle en review (coding agent) en priorité, sans demaner 
** SINON on continue celle en doing (coding agent) 
** SINON on propose celles semblant plus critique/importante
* travail sur la tache choisi :  
** passage de la tache en doing dans archon  
** pour la documentation, privilegier le RAG archon
** pour l'acces au code, privilegier serena  
** procéder sur la tache de manière incrementale
* une fois la tache terminée
** passer la tache en review  
** PUIS valider le resultat avec tous les moyens disponible en validant l'acces a l'application (incluant playwright), pas juste son etat 
** SI reussi, passer le proprietaire en "user", garder la tache en review et passer a la tache suivante  
** SINON, reprendre le travail sur la tache (en la repassant en doing)  

NOTE :
* se rappeler des tolérations pour les controlplane
* si il y a un pvc RWO, mettre la trategy en recreate 
* penser a mettre la redirection http vers https
* s'assurer que le certificat tls est bien obtenu par letsencrypt-staging en dev/test/staging et letsencrypt-prod en prod  

