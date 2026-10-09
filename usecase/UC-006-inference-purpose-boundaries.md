# UC-006 — Séparer conversation, embeddings et reasoning Hindsight

- **Statut** : Besoin confirmé ; Hindsight reasoning pas encore prouvé
- **Acteurs** : Fabric Admin, Tenant Admin, User, Hermes et Hindsight
- **Origine** : décision du propriétaire, 2026-10-09
- **Issues** : [#3885](https://github.com/charchess/vixens/issues/3885), [#3869](https://github.com/charchess/vixens/issues/3869), [#3980](https://github.com/charchess/vixens/issues/3980), [#3979](https://github.com/charchess/vixens/issues/3979)
- **Document** : `docs/adr/035-centralized-fabric-inference-boundary.md`
- **Dernière révision** : 2026-10-09

## Intention

En tant qu'**administrateur**, je veux choisir et verrouiller séparément les modèles et credentials utilisés pour discuter avec Hermes et ceux nécessaires à Hindsight, pour conserver des limites, facturations et usages cohérents.

## Exemple

Indiba reçoit :
- Chat Hermes : alias `txo-agent` via LiteLLM → **CPA** → Codex, Claude ou OpenRouter **si le provider choisi est réellement supporté par CPA**.
- Embeddings Hindsight : `txo-embedding` via LiteLLM → OpenRouter/BGE-M3, avec clé de service et autorisation propres.
- Reasoning/génération Hindsight : capacité **à concevoir et activer explicitement**, avec une route et clé de service distinctes si nécessaire. L'ADR-035 indique que le LLM génératif de Hindsight restait **désactivé par défaut** tant que sa gouvernance n'est pas démontrée.

Le Fabric Admin peut imposer certains modèles. L'admin Indiba gère ses comptes tenant uniquement quand autorisé. Le user ne peut connecter sa propre OAuth conversationnelle que si la policy le permet.

## Contrat attendu

- **LiteLLM est l'unique endpoint d'inférence servi aux workloads du tenant.**
- **Toute sortie conversationnelle Hermes** est raccordée par le **CPA tenant** ; ne pas créer une connexion directe LiteLLM → OpenRouter pour chat afin de contourner l'absence d'OAuth CPA.
- **Hindsight** utilise des modèles/routages/credentials **de service distincts** via LiteLLM ; un credential Hindsight n'autorise pas un appel chat Hermes.
- Les alias, restrictions, scopes de clés virtuelles et affectations sont résolus server-side ; les preferences user ne modifient pas un provider mémoire partagé sans droit explicite.
- Le metering distingue `tenant`, `workload`, `purpose`, `logicalModel`, `credentialOwner`, `payer` et coût vérifiable, pour éviter de mélanger chat et usage mémoire.
- Pas de fallback implicite vers un provider ou un compte non autorisé ; ni le repli intertenant, ni la fuite de token, ni l'escalade par un alias.
- Les services Hindsight restent indépendants d'une session utilisateur active et ne doivent pas tirer un compte OAuth privé sans délégation de service explicite.

## Recettes

- Une clé Hindsight `txo-embedding` ne peut pas appeler `txo-agent`.
- Une clé Hermes `txo-agent` ne peut pas utiliser le compte/service OpenRouter d'embedding.
- OpenRouter configuré comme provider de **chat** ne devient disponible qu'**après** raccordement vérifié au CPA, et sans autoriser la route service du même tenant.
- Un tenant sans credential chat valide ne devient pas miraculeusement conversationnel parce que son embedding fonctionne.
- Modification verrouillée du modèle embedding refusée au tenant/user ; modèle reasoning non activé tant qu'il n'a pas de politique et d'acceptance spécifiques.

## État des preuves

Les embeddings BGE-M3 et le metering tenant ont été prouvés sur Indiba ; hAIrem a validé la conversation Codex via LiteLLM → CPA. Le credential Hindsight et celui de chat sont déjà séparés dans le produit. Le contrat ci-dessus **n'implique pas** que les modèles Hindsight reasoning soient déployés, que les montants soient valorisés correctement, ni qu'OpenRouter chat soit disponible dans CPA aujourd'hui. Les anciens exemples de `gateway/CREDENTIAL_POOLS.md` donnant une sortie de chat générique directe LiteLLM → OpenRouter restent à harmoniser via une PR distincte.
