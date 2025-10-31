# ADR-001: Talos Linux comme OS des Nodes

## Statut
✅ Accepté

## Contexte

Besoin d'un système d'exploitation pour les nodes Kubernetes qui soit :
- Immutable et sécurisé
- Minimal (sans shell, SSH, packages inutiles)
- API-driven (pas de configuration manuelle)
- Adapté à la reconstruction fréquente (clusters dev/test)

### Alternatives Évaluées

1. **Ubuntu Server + kubeadm**
   - ✅ Très documenté, communauté large
   - ✅ Flexible (apt, systemd)
   - ❌ Mutable (drift config possible)
   - ❌ Surface d'attaque large

2. **Flatcar Container Linux**
   - ✅ Immutable, auto-update
   - ✅ systemd, docker natif
   - ❌ Moins focus Kubernetes
   - ❌ Configuration Ignition complexe

3. **k3OS (abandonné)**
   - ❌ Projet Rancher abandonné
   - ❌ Migration vers Talos recommandée

4. **Talos Linux**
   - ✅ 100% Kubernetes-native
   - ✅ API-only (talosctl), pas de SSH
   - ✅ Immutable, auto-heal
   - ✅ Support VLAN/multi-interface natif
   - ✅ Maintenance mode pour hardware virtualisé
   - ❌ Courbe d'apprentissage (pas de shell)
   - ❌ Debugging plus complexe

## Décision

**Adopter Talos Linux v1.10.7**

### Justifications

1. **Sécurité par Design** : Aucun accès shell = surface d'attaque minimale
2. **API-Driven** : Configuration via machineconfig (YAML) → Infrastructure as Code
3. **Immutabilité** : Pas de drift, reproductibilité totale
4. **VLAN Support** : Configuration multi-interface native (VLAN 111 + 20X)
5. **Maintenance Mode** : Idéal pour VMs virtualisées (pause/resume sans corruption)
6. **Terraform Provider** : `siderolabs/talos` officiel pour IaC complet

## Conséquences

### Positives
- ✅ Clusters reproductibles via Terraform (détruire/reconstruire sans risque)
- ✅ Pas de maintenance OS (pas de patches Ubuntu, pas de apt)
- ✅ Debugging via `talosctl` (logs, shell éphémère si nécessaire)
- ✅ Upgrade Talos atomique (rollback possible)

### Négatives
- ⚠️ Debugging plus difficile (pas de `ssh`, pas de `vim`)
  - **Mitigation** : Utiliser `talosctl shell` pour accès éphémère
- ⚠️ Logs centralisés nécessaires (pas de fichiers locaux persistants)
  - **Mitigation** : Aggregation Loki/Grafana (Phase future)
- ⚠️ Documentation moins abondante que kubeadm
  - **Mitigation** : Communauté Slack active, docs officielles complètes

## Références

- [Talos Linux Documentation](https://www.talos.dev/)
- [Terraform Provider Talos](https://registry.terraform.io/providers/siderolabs/talos/latest)
- [Talos vs Traditional OS](https://www.talos.dev/latest/introduction/what-is-talos/)

## Notes d'Implémentation

**Version cible** : Talos v1.10.7 (compatible Kubernetes v1.30.0)

**Configuration réseau** :
```yaml
machine:
  network:
    interfaces:
      - interface: eth0
        vlans:
          - vlanId: 111
            addresses: [192.168.111.X/24]
          - vlanId: 20X
            addresses: [192.168.20X.X/24]
            routes:
              - network: 0.0.0.0/0
                gateway: 192.168.20X.1
```

**Bootstrap** :
```bash
talosctl bootstrap --nodes 192.168.111.160 --endpoints 192.168.111.160
```

---

**Date** : 2025-10-30
**Auteur** : Infrastructure Team
**Révisé** : N/A
