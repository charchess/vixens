# UC-005 — Une WebUI pour discuter avec plusieurs agents

- **Statut** : Besoin confirmé ; développement MVP à lancer
- **Acteurs** : User autorisé, agents Hermes, admin tenant, Fabric WebUI
- **Origine** : décision produit du propriétaire, 2026-10-09
- **Issues** : [#3981](https://github.com/charchess/vixens/issues/3981), [#3832](https://github.com/charchess/vixens/issues/3832), [#3807](https://github.com/charchess/vixens/issues/3807), [#3689](https://github.com/charchess/vixens/issues/3689)
- **Dernière révision** : 2026-10-09

## Intention

En tant qu'**utilisateur**, je veux une **unique application Fabric** où retrouver mes agents, choisir avec lequel converser, accéder à mes paramètres et, si j'y suis autorisé, aux réglages de mon tenant.

## Exemple

Une personne autorisée à Electra et Sabrina s'authentifie via Authentik, ouvre `/agents`, sélectionne Electra, envoie un message et voit le streaming/la progression des outils. Elle passe à Sabrina puis revient à Electra : les fils ne se mélangent pas. Elle trouve `Mes paramètres` et éventuellement `Administration` dans la **même application**. Une personne non autorisée n'a pas accès aux autres agents, même en modifiant l'URL.

## Architecture métier

```text
Browser (Authentik / Fabric session)
  -> Fabric WebUI / API de conversation (IAM, sessions, audit)
  -> AgentIdentity Hermes (ses outils, skills, fichiers, mémoire)
  -> LiteLLM tenant (unique façade IA et metering)
  -> CPA tenant (tous les providers chat)
  -> Provider autorisé
```

**Ne pas** raccorder la WebUI directement à LiteLLM/CPA ; elle court-circuiterait Hermes et ses capacités. **Ne pas** exposer au navigateur les tokens API Hermes, les clés virtuelles ni les identifiants de Pods.

## Contrat attendu

- **Une seule base de code frontend** avec navigation `/agents`, `/settings`, `/admin` (admin tenant uniquement si autorisé) ; backend d'administration privilégiée et administration globale Fabric peuvent rester **isolés** en processus ou services.
- Déploiement UI par tenant privilégié initialement ; ne pas utiliser ce choix de packaging comme unique preuve d'isolation applicative.
- AuthN Authentik, AuthZ backend à chaque requête avec tenant, principal, AgentIdentity, session et action.
- Identifiants `AgentIdentity` stables plutôt que Pod/IP. Session créée et reprise explicitement ; refus cross-user/cross-tenant et pas de fallback vers un autre agent.
- Streaming de réponse ; états d'outils non secrets ; arrêt/annulation, erreurs, retry idempotent, reconnexion et reprise selon capacités réelles du Hermes épinglé.
- Fichiers et compétences utilisent les workspaces autorisés ; un bouton d'upload n'est pas « opérationnel » avant que l'API adaptée soit prouvée.
- UI responsive/mobile, accessibilité de base, français, branding tenant sans fork personnalisé.
- Choix initial réversible de bibliothèque : **assistant-ui** pour le MVP, à évaluer contre **Alibaba ChatUI** et une troisième option dans #3832. Pas de couplage de l'API Fabric à la bibliothèque.

## Recettes

- Un seul humain, deux agents autorisés : message à A, changement pour B, retour A → trois états propres et historisés.
- User tentant un agent sans grant : HTTP refusé, pas de tentative Hermes ni de message cross-tenant.
- Deux sessions appartenant à deux users différents ne partagent pas leur historique, même sur le même agent.
- Restart/remplacement Pod Hermes : la route reste stable ; la reprise de session suit le contrat réellement implémenté, sans garantie inventée.
- Stream lent, coupure réseau, annulation : état utilisateur explicite ; pas de duplication d'un tour à la reconnexion.
- Passage user→admin : l'UI adapte son menu, et **l'API vérifie à nouveau l'autorisation** même si l'utilisateur forge la requête.

## État des preuves / options

Le dashboard Hermes existant et la sélection d'Electra/Sabrina ont déjà été testés sur hAIrem, **pas la WebUI Fabric native**. Dans le tag `nousresearch/hermes-agent:v2026.9.24`, la documentation upstream de l'API Server annonce sessions, API OpenAI-compatible et SSE ; vérifier le fonctionnement **effectivement activé** dans notre image/runtime avant de choisir l'adapter. Le chat Indiba reste dépendant de l'enrôlement volontaire d'un provider CPA via #3979. L'UI complète / voice / Files tab restent des étapes distinctes.
