# 😈 Lisa's Cluster Audit & TODO

Ce fichier consigne les anomalies détectées par votre Régente et les actions correctives.
**Note pour l'IA stagiaire :** Suis ces instructions à la lettre. Ton physique ne te sauvera pas si tu casses le cluster.

---

## 🛠️ Actions Prioritaires (APPROVED)

### 1. Installation d'un mécanisme de purge (TTLController)
- **Problème** : Le cluster est rempli de pods "cadavres" (`Completed` ou `Error`) vieux de 20 jours.
- **Solution suggérée** : 
    - **Option A (Native)** : Ajouter `ttlSecondsAfterFinished: 3600` dans les specs de tous les `Jobs`.
    - **Option B (Contrôleur)** : Installer `kruise-rollout` ou un CronJob simple qui exécute `kubectl delete pod --field-selector=status.phase==Succeeded -A`.
- **Statut** : APPROVED. Priorité : Haute.

### 2. Suppression de `node-collector`
- **Problème** : Ce déploiement est en `CreateContainerError`. Il tente de créer `/etc/systemd` sur **Talos Linux**, qui est un OS immuable sans systemd. C'est conceptuellement impossible.
- **Action** : Supprimer l'application `node-collector` d'ArgoCD. Chercher une alternative compatible Talos si les metrics de nœuds sont vitales.
- **Statut** : APPROVED.

- [ ] **Implémenter les Kustomize Components** :
    - **Problème** : Le "DRY" (Don't Repeat Yourself) est impossible car Kustomize refuse de remonter au-dessus de la racine de l'application pour chercher des bases partagées.
    - **Solution 2026** : Créer des `Kustomize Components` dans `infrastructure/kustomize/components/`. Ces composants (ex: `priority-vixens`, `standard-probes`) peuvent être injectés de manière modulaire dans chaque `kustomization.yaml` sans les contraintes de hiérarchie des bases classiques.
    - **Statut** : Suggéré par Lisa, à implémenter par la stagiaire.

---

## 🔍 Investigations & Réparations (IA Stagiaire)

### 4. Goldilocks (OutOfSync)
- **Diagnostic** : Les déploiements `goldilocks-controller` et `goldilocks-dashboard` refusent de se synchroniser. Les `spec.selector` ont été modifiés dans Git, mais ce champ est **immuable** dans Kubernetes.
- **Action corrective** :
    1. Supprimer manuellement les deux déploiements : `kubectl delete deploy -n monitoring goldilocks-controller goldilocks-dashboard`.
    2. Cliquer sur "Sync" dans ArgoCD pour les laisser se recréer proprement avec les nouveaux labels.

### 5. External-DNS Gandi (Le mystère du Progressing)
- **Diagnostic** : Le pod redémarre en boucle (85 restarts en 14 jours). Les logs disent "All records up to date" juste avant de recevoir un `SIGTERM`.
- **Cause probable** : Absence de `Liveness/Readiness probes`. Le pod met trop de temps à s'initialiser ou ne répond pas sur son port de metrics, et Kubernetes le tue pour "non-réponse".
- **Action corrective** : Ajouter des probes HTTP sur le port 7979 (metrics) dans les `values.yaml` du Helm Chart.

### 7. Connecter Robusta aux "fesses d'Electra"
- **Objectif** : Recevoir les alertes critiques directement dans une interface IA ou un canal de traitement.
- **Technique** : Configurer un `Webhook Sink` dans Robusta pointant vers l'URL d'OpenClaw ou un endpoint géré par Electra.
- **Action** : Modifier la `Secret` ou la `ConfigMap` de Robusta pour ajouter le sink.

### 8. Nettoyage des vieux trucs (Stirling-PDF & Co)
- **Problème** : Confusion entre `revisionHistoryLimit` et nettoyage des pods.
- **Solution** : 
    - `revisionHistoryLimit: 3` (ou moins) sert à nettoyer les vieux **ReplicaSets** (les versions précédentes de l'app). À généraliser via le point 6.
    - Pour les **Pods**, Kubernetes Deployment gère le nettoyage lors des updates. Si des vieux pods restent, c'est que le Deployment est "stuck" (voir point 5).
    - Pour les **Jobs**, utiliser `ttlSecondsAfterFinished` (voir point 1).
- **IA Stagiaire** : Vérifie si des ReplicaSets orphelins traînent (`kubectl get rs -A`) et pourquoi Stirling-PDF déclenche une alerte Kyverno.

### 9. Probes & PriorityClasses
- **Guideline** : 
    - Tout pod critique DOIT avoir une `priorityClassName` commençant par `vixens-`.
    - Tout pod DOIT avoir des `Liveness` et `Readiness` probes.
- **IA Stagiaire** : Parcours les dossiers `apps/` et injecte ces éléments partout où ils manquent.

---
*Signé : Lisa, Régente Infernale. (Maintenant, au travail.)* 🍷⛓️
