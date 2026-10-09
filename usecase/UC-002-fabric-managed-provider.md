# UC-002 — Service IA fourni par Fabric et refacturé au tenant

- **Statut** : Besoin confirmé ; provisioning et tarification à concevoir
- **Acteurs** : Fabric Admin, Tenant Admin, plateforme billing
- **Origine** : décision produit du propriétaire, 2026-10-09
- **Issues** : [#3980](https://github.com/charchess/vixens/issues/3980), [#3885](https://github.com/charchess/vixens/issues/3885), [#3868](https://github.com/charchess/vixens/issues/3868), [#3979](https://github.com/charchess/vixens/issues/3979)
- **Dernière révision** : 2026-10-09

## Intention

En tant que **Fabric Admin**, je veux fournir au tenant un service OpenRouter déjà opérationnel, afin que le client puisse utiliser ses agents et Hindsight sans créer immédiatement ses propres comptes, avec consommation attribuée et **refacturée au client**.

## Exemple

Indiba choisit « Provider géré par Fabric ». La plateforme provisionne et affecte au tenant un compte ou une sous-clé strictement isolée, des modèles autorisés et un budget. L'admin Indiba voit le mode, les coûts et les restrictions. S'il a le droit de basculer vers BYOK, il le fait sans toucher aux Pods.

Pour les **conversations Hermes**, le chemin demeure `Hermes -> LiteLLM du tenant -> CPA du tenant -> provider de chat` (OpenRouter depuis CPA **si son support par la version épinglée est vérifié**). Pour **Hindsight**, les routes service dédiées `Hindsight -> LiteLLM du tenant -> endpoint embedding/reasoning` restent séparées.

## Contrat attendu

- La **propriété de la credential** reste Fabric ; son usage est accordé explicitement au tenant. Ne pas dupliquer la clé maître plateforme dans les workspaces, agents, statuts, manifests ou navigateur.
- Aucun CPA du tenant A ne peut utiliser implicitement des comptes du tenant B : une délégation plateforme doit produire des credentials/allocation **tenant-scoped** et respecter la frontière de routage actuelle.
- Le **payeur** peut être le tenant, malgré la propriété Fabric de la credential. Une politique décide des modèles autorisés, marges, tarifs, quotas, seuils et suspension éventuelle.
- Mesurer séparément usage technique (tokens/requêtes), coût fournisseur réel (si mesurable) et prix de refacturation ; ne pas déclarer facturable une valeur manquante ou `spend=0` comme si elle était fiable.
- Connexion, rotation, suspension et fin de contrat ne doivent pas demander de modifier l'état privé des agents. La révocation est effective au broker/gateway.
- Le tenant peut voir sa consommation et la politique commerciale applicables, **jamais la credential maître Fabric ni les usages d'autres clients**.

## Scénarios de recette

- Un client sans compte OpenRouter obtient un embedding Hindsight via son allocation de service, **après provisioning explicite**, et l'usage est attribué au bon tenant.
- Deux tenants utilisant le même service commercial n'échangent aucun credential, contexte ou budget ; l'un ne peut épuiser ou utiliser la sous-clé de l'autre.
- Le tenant admin BYOK autorisé peut remplacer sa source de financement sans recevoir un compte d'agent arbitrairement élargi.
- Une refacturation s'appuie sur un ledger vérifiable et une tarification décidée ; toute donnée insuffisante produit un état « coût non calculable ».

## État des preuves, risques, décisions à prendre

La façade LiteLLM tenant-scoped et les routes d'embedding sur OpenRouter ont été éprouvées pour hAIrem/Indiba ; **cela ne prouve pas** un service OpenRouter managé/refacturé de manière multi-tenant, ni un provider chat OpenRouter inscrit dans CPA. À concevoir : allocation de sous-clés tenant, quotas, workflow contractuel, ledger, prix/marge, exigences fournisseur.
