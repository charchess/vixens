# UC-004 — Comptes individuels OAuth, device auth et API utilisateur

- **Statut** : Besoin confirmé ; contrat d'isolation critique non résolu
- **Acteurs** : User, Tenant Admin, Fabric Admin, agent Hermes et CPA
- **Origine** : décision produit du propriétaire, 2026-10-09
- **Issues** : [#3980](https://github.com/charchess/vixens/issues/3980), [#3979](https://github.com/charchess/vixens/issues/3979), [#3868](https://github.com/charchess/vixens/issues/3868), [#3856](https://github.com/charchess/vixens/issues/3856)
- **Dernière révision** : 2026-10-09

## Intention

En tant que **salarié Indiba** disposant de mon propre abonnement/compte fournisseur, je veux le connecter dans **Mes paramètres** pour mes conversations autorisées, sans donner mon accès à tous les collègues ni imposer au tenant admin de gérer mes tokens.

## Exemple

Le tenant admin autorise les comptes personnels. Alice ouvre `/settings/providers`, initie un OAuth ou un flux device-code pour son propre compte, et utilise Sam. Bob utilise le **même AgentIdentity Sam**, mais il doit utiliser son propre compte ou un pool d'entreprise explicitement autorisé : jamais celui d'Alice.

## Contrat attendu

- La politique Fabric **et** tenant doit autoriser le provider, le modèle, le BYOK individuel et la nature de l'authentification ; aucun droit implicite.
- Le credential porte un **owner utilisateur** indépendant de l'AgentIdentity et du tenant ; son secret reste dans un stockage approprié, pas dans le state de l'agent.
- À chaque opération, l'identité de l'**utilisateur initiateur est attestée par le backend Fabric** et transmise sans usurpation possible jusqu'à la sélection du compte; le simple bearer LiteLLM actuel, par AgentIdentity, **ne suffit pas** pour choisir un compte individuel.
- La mise en pool tenant de credentials OAuth personnels, sans sélection par principal ou délégation explicite, est **interdite**.
- Les tâches autonomes / scheduled runs / agents de groupe sans humain initiateur courant ne peuvent **jamais emprunter automatiquement** le credential personnel du dernier humain. Prévoir une délégation révocable explicite pour ces cas, sinon user-pool indisponible.
- Les conditions d'utilisation et limites des abonnements OAuth personnels varient selon provider ; n'implémenter un mode commercial/collectif qu'après vérification contractuelle.
- Stockage/audit non-secret par user, tenant, compte, permission d'usage, identité de l'agent et payeur ; révocation rapide lors des changements de membership/IAM.

## Scénarios de recette

- Alice connecte son compte. Bob discute avec Sam. **Aucune** requête de Bob n'utilise le compte Alice, même si l'agent et la route LiteLLM sont identiques.
- Un user quitte Indiba : sa credential personnelle ne peut plus être sélectionnée par les agents du tenant ; le traitement de la propriété/récupération du secret suit la politique de conservation.
- Le tenant interdit le BYOK user : les nouveaux usages sont rejetés ; la simple présence d'un secret historique n'accorde aucun droit.
- Un agent lance un travail différé sans user présent : échec explicite ou usage d'un pool tenant **autorisé**, jamais choix silencieux du dernier utilisateur.
- Une rotation de la clé virtuelle par AgentIdentity ne réactive pas un ancien credential user révoqué.

## État des preuves / décision structurante

**Non implémenté et non validé physiquement.** Vérifier si CPA peut choisir un compte par principal/credential binding dans la version épinglée. Si non, construire une isolation de broker adaptée ou refuser cette fonctionnalité pour le MVP. Une UI BYOK ne prouve pas l'isolation du chemin complet. Ne pas coder un compte personnel sous forme d'une clé d'agent permanente.
