# UC-003 — Le tenant admin connecte les providers de son organisation (BYOK)

- **Statut** : Besoin confirmé ; UI à livrer
- **Acteurs** : Tenant Admin, utilisateurs, CPA et LiteLLM tenant-local
- **Origine** : décision produit du propriétaire, 2026-10-09
- **Issues** : [#3979](https://github.com/charchess/vixens/issues/3979), [#3980](https://github.com/charchess/vixens/issues/3980), [#3868](https://github.com/charchess/vixens/issues/3868), [#3917](https://github.com/charchess/vixens/issues/3917)
- **Dernière révision** : 2026-10-09

## Intention

En tant qu'**admin Indiba**, je veux connecter mes propres comptes/API de providers pour que mon entreprise maîtrise sa facturation et ses modèles sans intervention Kubernetes ni exposition des tokens aux agents.

## Exemple

L'admin se connecte à `indiba.<domaine>`, ouvre **Administration > Providers IA**, visualise les providers autorisés par Fabric, ajoute son compte OAuth Codex ou une API key compatible, puis choisit les capacités et utilisateurs/agents bénéficiaires dans les limites de la politique. Un autre panneau permet de configurer l'OpenRouter **service Hindsight**, indépendamment du provider de chat.

## Contrat attendu

- La liste de providers/actions reflète la **version CPA réellement supportée**, pas une promesse de Claude/OpenRouter non vérifiée.
- Les comptes conversationnels passent via `tenant LiteLLM -> tenant CPA` ; les endpoints service embeddings/reasoning Hindsight demeurent dans les routes dédiées LiteLLM.
- Connexion OAuth, device auth quand supporté, API-key, reconnexion, révocation, désactivation et rotation sont des **opérations utilisateur** médiées par une API Fabric sécurisée. Le parcours normal n'est pas `kubectl exec` ni l'édition de Secrets.
- Aucun token fournisseur n'entre dans le browser storage, les réponses de statut, le GitOps, la CRD, le PVC d'agent ni les logs.
- Un admin de tenant ne peut pas modifier les options verrouillées par Fabric ou celles d'un tenant étranger ; la lecture de métadonnées est également tenant-scoped.
- Si un provider est indisponible, l'UI affiche « non configuré / reconnexion requise / cooldown » et n'annonce pas la disponibilité sur la seule présence d'un alias.
- La politique d'accès précise si l'usage d'un compte d'entreprise est autorisé à tel agent, user ou tâche autonome. Le fait d'être propriétaire tenant n'accorde pas un accès mondial.

## Scénarios de recette

- Compte Codex absent : l'admin configure volontairement un compte par interface ; une conversation autorisée atteint CPA sans changer `txo-agent` côté Hermes.
- Utilisateur sans permission admin : action et API de gestion refusées.
- Tenant `hairem` authentifié : consultation et modification des comptes `indiba` refusées, même avec un ID deviné.
- Retour OAuth annulé / expiré : aucune credential incomplète n'est inscrite comme active.
- Une révocation empêche les prochains usages et une reconnexion restaure le service sans secret distribué à Hermes.

## État des preuves et limites

Le POC hAIrem a prouvé un **enrôlement administratif Codex par CLI** et la conversation Hermes → LiteLLM → CPA. Ce n'est **pas** une UI produit. Indiba ne doit pas recevoir automatiquement un compte Codex. L'enrôlement web, les permissions d'administration tenant et les providers complémentaires restent à livrer sous #3979.
