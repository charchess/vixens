# Boîte à idées et attentes — non qualifiées

Notes rapides pour conserver les propositions avant d'en faire une issue ou un UC formalisé. **Les éléments ici ne sont ni décidés, ni livrés.** Une fois qualifiés, les transformer en fiche `UC-NNN` et rattacher à l'issue/ADR ; ne pas faire de cette liste un backlog parallèle à GitHub.

## Pistes à instruire

- **Librairie de préférences utilisateur** : modèle favori par type de tâche (rédaction, code, recherche), sans élargir la liste effective autorisée. Réfs : #3869, UC-001, UC-005.
- **Compte personnel versus agent partagé** : lorsqu'un agent appartient à un groupe mais qu'une personne le consulte, quel credential l'agent utilise-t-il pendant le chat, puis lors d'une tâche autonome déclenchée plus tard ? Réfs : #3856, #3980, UC-004.
- **Plan de secours / repli du provider** : permission explicite de basculer d'un compte utilisateur vers un pool de tenant ou managé, avec consentement et information de facturation, sans surprise ni fuite intertenant. Réfs : #3868, #3980.
- **Portail de transparence des coûts** : consommation par agent, utilisateur, provider, modèle et payeur ; prix fournisseur versus refacturation, enveloppes et alertes. Ne pas considérer les métriques nulles de LiteLLM comme une comptabilité prouvée. Réfs : #3885, #3980.
- **Choix de provider Hindsight** : embeddings et reasoning peuvent nécessiter des contrats, prix et gouvernances différents du chat. Reasoning Hindsight n'est pas à déclarer actif sans validation (ADR-035). Réfs : UC-006.
- **Administration de la plateforme** : expérience éventuellement commune visuellement à la WebUI tenant, mais backend et accès privilégiés séparés. Réfs : #3807, #3808.

Pour ajouter une idée : noter sa provenance, un scénario simple et l'issue candidate. Ne jamais inclure de token, de secret ou d'exemple de clé réelle.
