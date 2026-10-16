# UC-007 — POC 1 : tenant déclaratif, identité humaine et agents

- **Statut** : Besoin confirmé ; contrat et livraison POC 1 à implémenter
- **Acteurs** : Propriétaire Fabric, opérateur TXO Fabric, utilisateur Authentik, agents Hermes
- **Origine** : décision du propriétaire, 2026-10-16
- **Issues** : [#4087](https://github.com/charchess/vixens/issues/4087) ; dépend de #3852, #4037, #3981 et #3996
- **Dernière révision** : 2026-10-16

## Intention

En tant que propriétaire Fabric, je veux décrire un tenant, ses groupes, ses utilisateurs et ses agents dans le désiré GitOps, afin que l’opérateur soit l’unique autorité qui matérialise les ressources tenant, les groupes Authentik dérivés, les identités Fabric et les runtimes Hermes associés.

En tant qu’utilisateur Authentik explicitement déclaré pour un tenant, je veux pouvoir m’authentifier puis converser uniquement avec les agents auxquels mon tenant et mes groupes m’autorisent.

## Contrat produit proposé

Le tenant YAML est la source de vérité. Une projection illustrative est :

```yaml
spec:
  tenantId: TEN00001
  humanAccess:
    web:
      iamGroups: [client0]
  identities:
    - key: charchess
      authentik:
        username: charchess
      groups: [client0]
  agents:
    - key: a1
    - key: a2
    - key: a3
```

Les identifiants humains lisibles sont des références de déclaration et non des identifiants durables. L’opérateur résout un unique compte Authentik actif, épingle son identité immuable et refuse toute ambiguïté, recréation ou substitution silencieuse.

## Comportement attendu

1. L’opérateur crée et maintient les groupes Authentik tenant-locaux dérivés de `iamGroups`, par exemple `txo-fabric-hairem-client0`.
2. L’opérateur réconcilie uniquement les memberships explicitement déclarées pour les identités du tenant ; il ne gère pas la création de mots de passe, invitations ou MFA humains dans POC 1.
3. L’opérateur matérialise les agents déclarés comme `AgentIdentity` enfants stables, avec ownership explicite et sans coupler identité métier, Pod, profil runtime ou nom d’affichage.
4. Lors de la première connexion, le BFF privé vérifie un ID token Authentik signé puis transmet seulement une preuve de travail authentifiée et tenant-bound au chemin Fabric privé.
5. Fabric associe la preuve à l’identité déclarée, écrit le binding du ledger avec CAS et relecture, puis réconcilie les tuples OpenFGA requis pour l’appartenance tenant et `agent.chat`.
6. Toute absence de preuve, dérive d’issuer/sujet/UUID, compte inactif, groupe manquant, membership non déclaré, store/modèle invalide, écriture concurrente ou vérification indisponible échoue fermée, sans grant partiel.

## Livrable POC 1 — critères d’acceptation

Le POC 1 est considéré utilisable seulement si les preuves suivantes sont obtenues pour hAIrem :

- [ ] le tenant YAML hAIrem déclare au moins un groupe, une identité Authentik existante et au moins deux agents ;
- [ ] l’opérateur crée le groupe Authentik dérivé attendu et le réconcilie de façon idempotente ;
- [ ] une identité déclarée est résolue exactement une fois vers un compte Authentik actif et une identité Fabric stable ;
- [ ] les agents déclarés sont matérialisés avec des identités stables et restent isolés par tenant/utilisateur ;
- [ ] le BFF privé vérifie une authentification Authentik réelle et le bridge vers Fabric est authentifié, limité au tenant et sans endpoint public ;
- [ ] le ledger d’identité, le store et le modèle OpenFGA hAIrem sont confirmés ;
- [ ] `OpenFGAAuthorizationReady=True` n’est atteint qu’après lecture de contrôle des memberships et tuples ;
- [ ] l’utilisateur déclaré peut converser avec au moins deux agents autorisés via la WebUI privée ;
- [ ] toute tentative cross-tenant, agent non autorisé, token forgé, compte désactivé ou groupe retiré est refusée côté backend ;
- [ ] ArgoCD est `Synced/Healthy`, les workloads requis sont Ready et les preuves fonctionnelles sont jointes à l’issue ;
- [ ] aucune valeur secrète, token, UUID privé ou contenu de ledger n’est committé, journalisé ou affiché dans les preuves.

## Non-conformités POC 1 explicitement acceptables

Ces écarts sont acceptables uniquement s’ils sont visibles dans l’issue de livraison :

- l’opérateur référence un compte Authentik préexistant ; il ne crée pas encore les comptes humains, mots de passe, invitations ou MFA ;
- le POC est limité à hAIrem, à un utilisateur initial explicitement déclaré et à un ensemble borné d’agents ;
- les identités/agents existants peuvent rester des ressources séparées pendant une migration non destructive vers la déclaration compacte ;
- l’interface peut être minimale si les contrôles serveur, Authentik, BFF, bridge et OpenFGA sont réellement prouvés.

## Non-conformités non acceptables

- binding créé depuis un username, groupe ou header non vérifié ;
- groupes Authentik créés ou modifiés manuellement hors opérateur ;
- bootstrap de membership à partir d’un inventaire IAM non déclaré ;
- fallback OpenBao Infra pour un secret tenant/runtime ;
- token permanent du bridge dans Git, ConfigMap, logs ou navigateur ;
- service/Ingress public ou pont non authentifié présenté comme “privé” ;
- grant OpenFGA implicite, cross-tenant ou non relu ;
- remplacement/suppression de modèle OpenFGA, PVC ou identité stable durant la livraison POC ;
- association automatique d’un username réutilisé à un ancien principal Fabric.

## Dépendances et décisions à tracer

- #3852 : domaine d’identités techniques stables, migration des agents et des banques mémoire ;
- #4037 : BFF global/privé vers Fabric avec preuve workload et association issuer-safe ;
- #3981 : WebUI conversationnelle ;
- #3996 : ledger, projection IAM et tuples OpenFGA ;
- ADR OpenFGA et ADR identité/bridge à compléter avant la promotion d’une implémentation d’accès.

## Recettes principales

- hAIrem déclare `client0`, une identité et deux agents ; l’opérateur matérialise le groupe dérivé, les agents et les associations attendues sans intervention Authentik manuelle.
- Le membre déclaré se connecte réellement ; Fabric crée au plus une identité opaque stable, et une seconde connexion est idempotente.
- Le membre voit deux agents autorisés, converse avec l’un puis l’autre ; les sessions et historiques ne se mélangent pas.
- Un compte actif non déclaré dans le tenant, un membre retiré, un compte désactivé et un token tenant différent sont tous refusés avant Hermes.
