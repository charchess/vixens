# hAIrem A2UI Specification (Agent-to-User Interface)

**Version:** 1.0 (Visual Novel & Vocal-First)  
**Status:** Finalized  
**Author:** Sally (UX Expert)  

---

## 1. Introduction & Principes UX
hAIrem est une interface de **Présence** et de **Narration** conçue pour transformer des agents IA en un équipage vivant.
*   **Vocal-First :** L'interface visuelle est un support émotionnel, l'action doit être possible sans écran.
*   **Incarnation (A2UI) :** Les émotions des agents sont les premiers indicateurs d'état du système.
*   **Priorité au Contexte :** En cas d'alerte P0, la narration s'efface devant l'urgence.

---

## 2. Architecture de l'Information (The Stage)
L'interface est un "Théâtre" plein écran composé de 4 calques :
1.  **Z-0 : Background Layer** (Décors générés selon le DNA visuel).
2.  **Z-10 : Agent Layer** (Jusqu'à 3 personnages simultanés).
3.  **Z-20 : Ambiant Data Rails** (Widgets domotiques apparaissant sur les bords).
4.  **Z-30 : Dialogue Layer** (Ardoise Narrative et Totems d'agents).

---

## 3. Mise en Scène Dynamique (Staging)
*   **Focus Rule :** L'agent qui parle est à 100% d'opacité et au premier plan. Les autres sont à 70% et décalés.
*   **Transitions :** Utilisation de Framer Motion pour des glissements fluides (300ms) lors des changements de focus.
*   **Top Corners :** Gauche pour le contexte (Heure/Scène) ; Droite pour le contrôle système (Chaos Switch).

---

## 4. Composants Cœurs & Style
*   **Ardoise Narrative :** Fond "Obsidian" (95% opaque), pas de gradients. Bordure de 2px de la couleur de l'agent.
*   **Effet Typewriter :** Texte streamé synchronisé avec l'audio (TTS).
*   **Totems (Icônes) :** Symboles SVG minimalistes dans la barre de statut pour une présence passive.
*   **Glow Sync :** La bordure de l'Ardoise pulse selon l'amplitude vocale de l'agent.

---

## 5. DNA Visuel & Uniformité
L'uniformité graphique est gérée par un **Style Prompt central** (ex: "Cyber-Cozy" ou "Picasso Style"). 
*   Tous les agents et décors sont générés en injectant ce DNA.
*   Interdiction des textures bruyantes ou des dégradés complexes pour maintenir la lisibilité.

---

## 6. Accessibilité & Responsivité
*   **Mode Kiosque (Wall Tablet) :** Augmentation de 20% des polices, protection OLED (micro-décalages de 1px).
*   **Breakpoints :**
    *   *Mobile :* Portrait Focal (1 agent max).
    *   *Tablet/Desktop :* The Stage (3 agents).
    *   *Ultra-Wide :* Command Center (Sidebars HA permanentes).

---

## 7. Prochaines Étapes
1.  Implémentation du composant `Stage.tsx` avec Framer Motion.
2.  Développement du `DialogueBox` avec support du streaming Redis.
3.  Création du premier set de poses (sprites placeholders) respectant le DNA visuel.

---
*Designed by Sally, UX Expert (BMAD™ Core)*

---

## 8. Gestion des Erreurs & Soundscape
*   **UX de Défaillance :** Utilisation de la pose `glitch` (distorsion visuelle) pour signaler une panne LLM/Réseau sans briser le personnage.
*   **Soundscape Cyber-Cozy :** Utilisation de sons organiques (bois, tissu, air) pour les interactions système afin de maintenir un environnement apaisant.
*   **Typewriter Sync :** Les micro-sons de texte sont synchronisés avec l'affichage et l'audio TTS.

---

## 9. Le Hub de l'Équipage (Backstage)
*   **Vue Gestion :** Grille des agents installés, accès aux UUID et aux statistiques d'affinité.
*   **Memory Peek :** Affichage des derniers fragments de mémoire subjective pour chaque expert.

---

## 10. Alphabet Émotionnel (Core Emotion Set)
Tout agent doit fournir un mapping pour les 10 états de base :
`idle`, `thinking`, `happy`, `sad`, `alert`, `emergency`, `confused`, `shy`, `angry`, `glitch`.

---
*Final UX Review completed. System ready for implementation.*
