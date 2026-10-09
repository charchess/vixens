# UC-001 — Politiques hiérarchiques et options verrouillables

- **Statut** : Besoin confirmé ; contrat technique à préciser
- **Acteurs** : Fabric Admin, Tenant Admin, User
- **Origine** : décision produit du propriétaire, 2026-10-09
- **Issues** : [#3980](https://github.com/charchess/vixens/issues/3980), [#3979](https://github.com/charchess/vixens/issues/3979), [#3869](https://github.com/charchess/vixens/issues/3869), [#3807](https://github.com/charchess/vixens/issues/3807)
- **Documents** : `docs/adr/035-centralized-fabric-inference-boundary.md`
- **Dernière révision** : 2026-10-09

## Intention

En tant que **Fabric Admin**, je veux proposer ou imposer les paramètres IA d'un tenant ou d'un utilisateur, afin de garantir les choix de sécurité et les engagements commerciaux, tout en laissant l'autonomie restante aux administrateurs et utilisateurs.

En tant que **Tenant Admin**, je peux personnaliser les options non verrouillées pour mon tenant et restreindre les choix laissés aux utilisateurs.

En tant que **User**, je peux modifier seulement mes préférences et connexions autorisées.

## Exemples concrets

1. Fabric Admin impose `txo-embedding -> BGE-M3` sur Indiba ; ni tenant ni user ne peut déplacer cette route.
2. Fabric fournit un pool de chat managé **par défaut sans lock** ; le tenant bascule vers son provider BYOK.
3. Indiba impose son provider de chat ; un user ne peut pas forcer son OAuth personnel.
4. Indiba autorise le BYOK utilisateur : chaque compte autorisé reste affecté à son propriétaire.
5. Fabric Admin verrouille un paramètre pour **un seul utilisateur** ; le résultat ne doit pas dépendre d'une manipulation de l'interface par cet utilisateur.

## Contrat attendu

- Une politique effective est évaluée **côté backend**, à partir du principal authentifié, tenant, ressource/capacité et opération.
- Une valeur `enforced` d'un niveau supérieur ne peut être écrasée par une valeur inférieure ; une `allowedValues`/une limite parent ne peut être élargie par l'enfant.
- Les valeurs par défaut non verrouillées sont configurables par les descendants habilités.
- Les refus, contraintes et restrictions cumulatives doivent être explicites ; ne pas assimiler « champ non verrouillé » à « tout provider autorisé ».
- La politique expose une **provenance non secrète** (niveau ayant fixé/verrouillé, motif, limite) afin d'expliquer l'UI et faciliter l'audit.
- Les modifications de droit/retrait de lock doivent être auditées et prises en compte pour la **prochaine opération sensible** sans conserver d'anciens droits actifs.
- Le navigateur ne peut pas créer une autorisation en passant un tenant ou user arbitraire : IAM et portée résolues serveur.

## Matrice de précédence — proposition à valider techniquement

| Origine du paramètre | Peut imposer / restreindre | Override possible si permis |
|---|---|---|
| Fabric Admin (global, tenant ciblé, user ciblé) | Tenant et user dans sa portée | Seulement si explicitement déverrouillé |
| Tenant Admin | Users du tenant, dans limites Fabric | Selon politique tenant |
| User | Ses propres préférences | Jamais au-delà des parents |

**À concevoir :** ordre exact quand une restriction tenant et une exception Fabric ciblée user sont contradictoires. Éviter un faux algorithme « dernier écrit gagne ». Les interdictions de sécurité Fabric ne sont jamais contournables par une exception de confort.

## Scénarios de recette

- **Étant donné** un modèle imposé par Fabric, **quand** l'admin tenant tente une modification par UI **ou API**, **alors** l'écriture est refusée et l'origine du lock est visible.
- **Étant donné** une valeur Fabric non verrouillée, **quand** l'admin tenant change le provider, **alors** le résultat effectif de son tenant change, pas celui d'hAIrem.
- **Étant donné** un user autorisé BYOK, **quand** le tenant retire ce droit, **alors** toute nouvelle utilisation d'un compte individuel est refusée, même si la préférence historique subsiste.
- **Étant donné** une interdiction de provider Fabric, **quand** user ou tenant tente de l'ajouter sous un autre alias, **alors** aucun routage alternatif ne contourne la restriction.
- **Étant donné** une exception utilisateur déclarée par Fabric, **quand** elle coexiste avec une restriction tenant, **alors** la résolution suit la règle approuvée documentée et testée (non arrêtée ici).

## État des preuves et questions

Le modèle hiérarchique décrit une **attente produit confirmée**, pas une capacité runtime livrée. #3980 porte la formalisation des objets, l'algorithme de calcul de policy et les tests. Politique effective, provenance, versioning, révocation, exceptions ciblées et RBAC précis restent à implémenter/vérifier.
