# Quality Standards (Goldification)

Ce document résume le système de maturité applicative Vixens.

> **Sources de vérité :**
> - [ADR-023 — 7-Tier Goldification System v2](../adr/023-7-tier-goldification-system-v2.md) pour les niveaux et leur intention ;
> - [ADR-029 — Align application maturity with the current platform](../adr/029-align-maturity-with-current-platform.md) pour l'alignement secrets/ressources actuel.
>
> Les détails d'implémentation doivent toujours être vérifiés dans les manifests et politiques courants.

## Système 7 tiers

| Niveau | Nom | Intention |
|---|---|---|
| 🥉 1 | Bronze | L'application existe, tourne et est correctement structurée |
| 🥈 2 | Silver | Production ready : limites, probes, TLS, secrets externes |
| 🥇 3 | Gold | Observable : métriques, alerting pertinent, déploiement maîtrisé |
| 💎 4 | Platinum | Reliable : priorité, sizing justifié, résilience |
| 🟢 5 | Emerald | Data durability : sauvegarde/restauration adaptées aux données |
| 💠 6 | Diamond | Secure & integrated : isolation, durcissement, intégrations |
| 🌟 7 | Orichalcum | Éprouvée : stabilité, sizing validé, dette sécurité maîtrisée |

La progression reste séquentielle : un niveau non satisfait bloque les niveaux supérieurs.

## Bronze — Déployée

Principaux critères :

- image versionnée, pas de `:latest` ;
- CPU/memory **requests** explicites ;
- Service si nécessaire ;
- structure Kustomize cohérente ;
- Ingress si l'application doit être exposée.

## Silver — Production Ready

Bronze + :

- CPU/memory **limits** explicites ;
- readiness et liveness probes ;
- startup probe ou bypass explicite lorsqu'elle est inutile ;
- TLS/HTTPS si exposée ;
- secrets gérés hors Git ;
- stratégie de rollout cohérente avec le stockage persistant.

Le backend secret canonique actuel est :

```text
OpenBao → ClusterSecretStore/openbao → ExternalSecret → Secret → workload
```

Ne pas créer de nouvel `InfisicalSecret`.

## Gold — Observable

Silver + :

- métriques exposées ou exemption explicite ;
- ServiceMonitor lorsque pertinent ;
- Goldilocks/VPA utilisables pour observer le sizing ;
- `revisionHistoryLimit` conforme au pattern courant ;
- ordre ArgoCD/sync-wave lorsque nécessaire ;
- alerting utile lorsque l'application porte un risque opérationnel identifiable.

## Platinum — Reliable

Gold + :

- `priorityClassName` cohérent avec la criticité ;
- sizing choisi et justifié ;
- sizing revu à partir des observations ;
- PDB/topology spread/anti-affinity lorsqu'ils ont un sens ;
- graceful shutdown ou bypass explicite ;
- HPA/KEDA lorsque la charge variable le justifie.

## Emerald — Data Durability

Platinum + :

- profil de backup explicite ;
- restauration cohérente avec le type de données ;
- Litestream si SQLite et si ce pattern est approprié ;
- Config-Syncer si fichiers persistants et si approprié ;
- Velero/CSI/backup validés pour les PVC concernés ;
- ressources explicites pour les sidecars et init containers.

Le mécanisme de backup doit être choisi selon le workload ; ne pas ajouter automatiquement Litestream ou Config-Syncer à une application qui n'en a pas besoin.

## Diamond — Secure & Integrated

Emerald + :

- PSA / SecurityContext adaptés ;
- CiliumNetworkPolicy / NetworkPolicy avec flux minimaux nécessaires ;
- SSO Authentik lorsque pertinent ;
- politique supply-chain/image conforme ;
- restauration réellement testée lorsque requise ;
- intégrations de plateforme utiles (Homepage, etc.) ou bypass documenté.

## Orichalcum — Éprouvée

Diamond + :

- période de stabilité définie par ADR-023 ;
- sizing validé sur données réelles ;
- CVE HIGH/CRITICAL absente ou risque explicitement accepté/documenté.

## Ressources et sizing

Les sizing labels **ne remplacent pas** les ressources Kubernetes explicites.

Pattern attendu :

```yaml
spec:
  template:
    metadata:
      labels:
        vixens.io/sizing.app: V-medium
    spec:
      containers:
        - name: app
          resources:
            requests:
              cpu: 100m
              memory: 256Mi
            limits:
              cpu: 1000m
              memory: 1Gi
```

Les valeurs exactes et la convention de tier se prennent dans :

- [`RESOURCE_STANDARDS.md`](RESOURCE_STANDARDS.md) ;
- les policies/components courants ;
- une application récente comparable ;
- les observations Goldilocks/VPA.

## Bypasses

Les bypasses sont des déclarations intentionnelles, pas un moyen de masquer une dette. Les annotations précises et leur sémantique sont définies dans ADR-023 et les politiques courantes.

Avant d'en ajouter une :

1. vérifier que le critère est réellement non applicable ;
2. vérifier l'annotation actuelle dans Git ;
3. documenter la raison lorsqu'elle n'est pas évidente.

## Validation

Le score de maturité est un outil de complétude, pas un substitut au test fonctionnel.

Après une modification :

1. CI/Kustomize doivent passer ;
2. ArgoCD doit converger ;
3. l'application doit être fonctionnellement validée ;
4. logs, métriques et réseau doivent être vérifiés lorsque pertinents.

## Références

- [ADR-023 — 7-Tier Goldification System v2](../adr/023-7-tier-goldification-system-v2.md)
- [ADR-029 — Platform alignment](../adr/029-align-maturity-with-current-platform.md)
- [Resource Standards](RESOURCE_STANDARDS.md)
- [Deployment Standard](../procedures/deployment-standard.md)
- [Secret Management](../guides/secret-management.md)
- [Adding a New Application](../guides/adding-new-application.md)

**Last Updated:** 2026-09-25
