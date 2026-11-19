<template>
  <span
    :key="burst || 0"
    class="nt-anim-icon"
    :class="[`nt-anim-icon--${name}`, { 'is-active': active, 'is-burst': !!burst }]"
    aria-hidden="true"
  >
    <!-- Home: architectural console house with roof hop -->
    <svg v-if="name === 'home'" viewBox="0 0 24 24" fill="none">
      <path
        class="nt-ai-fill"
        d="M4.5 10.75L12 4.5l7.5 6.25V19a1.75 1.75 0 0 1-1.75 1.75H6.25A1.75 1.75 0 0 1 4.5 19v-8.25Z"
      />
      <path
        class="nt-ai-roof"
        d="M3.5 11.2L12 4.2l8.5 7"
        stroke="currentColor"
        stroke-width="1.8"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <path
        class="nt-ai-body"
        d="M5.5 10.2V18.5a1.5 1.5 0 0 0 1.5 1.5h10a1.5 1.5 0 0 0 1.5-1.5v-8.3"
        stroke="currentColor"
        stroke-width="1.75"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <path
        class="nt-ai-door"
        d="M9.75 20v-4.75a1.25 1.25 0 0 1 1.25-1.25h2a1.25 1.25 0 0 1 1.25 1.25V20"
        stroke="currentColor"
        stroke-width="1.75"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>

    <!-- Personal products: 4-quadrant blocks with staggered spring burst -->
    <svg v-else-if="name === 'personal'" viewBox="0 0 24 24" fill="none">
      <rect class="nt-ai-q nt-ai-q0" x="4" y="4" width="6.75" height="6.75" rx="2" stroke="currentColor" stroke-width="1.75" />
      <rect class="nt-ai-q nt-ai-q1" x="13.25" y="4" width="6.75" height="6.75" rx="2" stroke="currentColor" stroke-width="1.75" />
      <rect class="nt-ai-q nt-ai-q2" x="4" y="13.25" width="6.75" height="6.75" rx="2" stroke="currentColor" stroke-width="1.75" />
      <rect class="nt-ai-q nt-ai-q3" x="13.25" y="13.25" width="6.75" height="6.75" rx="2" stroke="currentColor" stroke-width="1.75" />
    </svg>

    <!-- Need Gateway: neural routing hub with center pulse -->
    <svg v-else-if="name === 'gateway'" viewBox="0 0 24 24" fill="none">
      <path
        class="nt-ai-trace"
        d="M12 8.5V5.5M8.9 13.8L6.2 15.4M15.1 13.8l2.7 1.6"
        stroke="currentColor"
        stroke-width="1.75"
        stroke-linecap="round"
      />
      <circle class="nt-ai-node nt-ai-node-0" cx="12" cy="4.25" r="2" stroke="currentColor" stroke-width="1.7" />
      <circle class="nt-ai-node nt-ai-node-1" cx="4.8" cy="16.5" r="2" stroke="currentColor" stroke-width="1.7" />
      <circle class="nt-ai-node nt-ai-node-2" cx="19.2" cy="16.5" r="2" stroke="currentColor" stroke-width="1.7" />
      <circle class="nt-ai-hub" cx="12" cy="12" r="3.4" stroke="currentColor" stroke-width="1.8" />
    </svg>

    <!-- Reset Radar: sweep hand + expanding pulse ring -->
    <svg v-else-if="name === 'radar'" viewBox="0 0 24 24" fill="none">
      <circle class="nt-ai-ring" cx="12" cy="12" r="8.5" stroke="currentColor" stroke-width="1.5" />
      <circle class="nt-ai-outer" cx="12" cy="12" r="8.5" stroke="currentColor" stroke-width="1.7" />
      <circle class="nt-ai-mid" cx="12" cy="12" r="4.6" stroke="currentColor" stroke-width="1.5" opacity="0.6" />
      <path
        class="nt-ai-sweep"
        d="M12 12L17.4 6.6"
        stroke="currentColor"
        stroke-width="1.9"
        stroke-linecap="round"
      />
      <circle class="nt-ai-dot" cx="12" cy="12" r="1.55" fill="currentColor" />
    </svg>

    <!-- Account Check: shield fill + drawn checkmark -->
    <svg v-else-if="name === 'accountCheck'" viewBox="0 0 24 24" fill="none">
      <path
        class="nt-ai-shield"
        d="M12 3.5L19 6.3v5.4c0 4.35-2.95 7.55-7 8.8-4.05-1.25-7-4.45-7-8.8V6.3L12 3.5Z"
        stroke="currentColor"
        stroke-width="1.75"
        stroke-linejoin="round"
      />
      <path
        class="nt-ai-check"
        d="M8.8 12.1l2.15 2.15L15.4 9.8"
        stroke="currentColor"
        stroke-width="1.95"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>

    <!-- Help Center: rounded guide bubble with spring tilt -->
    <svg v-else-if="name === 'help'" viewBox="0 0 24 24" fill="none">
      <path
        class="nt-ai-bubble"
        d="M4.5 6.75A2.75 2.75 0 0 1 7.25 4h9.5A2.75 2.75 0 0 1 19.5 6.75v7.5A2.75 2.75 0 0 1 16.75 17H10.6l-3.9 3.1a.6.6 0 0 1-.97-.47V17a2.75 2.75 0 0 1-1.23-2.3V6.75Z"
        stroke="currentColor"
        stroke-width="1.75"
        stroke-linejoin="round"
      />
      <path
        class="nt-ai-lines"
        d="M9.6 9.15a2.4 2.4 0 0 1 4.65.8c0 1.4-1.85 1.85-2.1 2.75"
        stroke="currentColor"
        stroke-width="1.75"
        stroke-linecap="round"
      />
      <circle cx="12.15" cy="14.7" r="0.95" fill="currentColor" />
    </svg>

    <!-- Recharge: high-voltage bolt with energetic kick -->
    <svg v-else-if="name === 'recharge'" viewBox="0 0 24 24" fill="none">
      <path
        class="nt-ai-bolt"
        d="M13.2 3.2L5.6 13.1h5.8l-1 7.7 8-10.1h-5.8l.6-7.5Z"
        stroke="currentColor"
        stroke-width="1.8"
        stroke-linejoin="round"
      />
    </svg>

    <!-- Rental / Ready Accounts: layered package card with check -->
    <svg v-else-if="name === 'rental'" viewBox="0 0 24 24" fill="none">
      <path
        class="nt-ai-box"
        d="M4.5 8.5L12 4.5l7.5 4v7L12 19.5l-7.5-4v-7Z"
        stroke="currentColor"
        stroke-width="1.75"
        stroke-linejoin="round"
      />
      <path
        class="nt-ai-lid"
        d="M4.8 8.7L12 12.6l7.2-3.9M12 12.6v6.8"
        stroke="currentColor"
        stroke-width="1.65"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>

    <!-- Telegram paper plane -->
    <svg v-else viewBox="0 0 24 24" fill="none">
      <path
        class="nt-ai-plane"
        d="M20.6 4.7L3.7 11.4c-.9.35-.88 1.62.03 1.94l4.45 1.54 1.72 4.92c.25.73 1.2.9 1.69.3l2.48-3.02 4.44 3.22c.68.49 1.65.12 1.83-.7l2.53-13.5c.17-.9-.73-1.64-1.57-1.4Z"
        stroke="currentColor"
        stroke-width="1.7"
        stroke-linejoin="round"
      />
      <path
        d="M8.3 14.8L18.2 7.5"
        stroke="currentColor"
        stroke-width="1.6"
        stroke-linecap="round"
      />
    </svg>
  </span>
</template>

<script setup lang="ts">
export type NavIconName =
  | 'home'
  | 'personal'
  | 'gateway'
  | 'radar'
  | 'accountCheck'
  | 'help'
  | 'recharge'
  | 'rental'
  | 'telegram'

defineProps<{
  name: NavIconName
  active?: boolean
  burst?: number
}>()
</script>

<style scoped>
.nt-anim-icon {
  display: inline-grid;
  place-items: center;
  width: 17px;
  height: 17px;
  flex: none;
}
.nt-anim-icon svg {
  width: 100%;
  height: 100%;
  overflow: visible;
}

/* Base dual-state fill (wireframe when idle, soft tinted fill when active) */
.nt-ai-fill,
.nt-ai-q,
.nt-ai-hub,
.nt-ai-node,
.nt-ai-shield,
.nt-ai-bubble,
.nt-ai-bolt,
.nt-ai-box,
.nt-ai-plane {
  fill: transparent;
  transition: fill 220ms var(--nt-motion-ease, ease-out), transform 380ms var(--nt-ease-spring, ease-out), opacity 200ms ease;
}

.nt-anim-icon.is-active .nt-ai-fill,
.nt-anim-icon.is-active .nt-ai-q,
.nt-anim-icon.is-active .nt-ai-hub,
.nt-anim-icon.is-active .nt-ai-shield,
.nt-anim-icon.is-active .nt-ai-bubble,
.nt-anim-icon.is-active .nt-ai-bolt,
.nt-anim-icon.is-active .nt-ai-box,
.nt-anim-icon.is-active .nt-ai-plane {
  fill: color-mix(in srgb, currentColor 20%, transparent);
}
.nt-anim-icon.is-active .nt-ai-node {
  fill: currentColor;
}

/* 1. Home roof hop */
.nt-ai-roof { transform-origin: 12px 7px; }
@media (prefers-reduced-motion: no-preference) {
  .nt-anim-icon--home:is(.is-active, .is-burst) .nt-ai-roof,
  :is(a, button):hover > .nt-anim-icon--home .nt-ai-roof {
    animation: nt-ai-roof-hop 460ms var(--nt-ease-spring, cubic-bezier(0.34, 1.4, 0.64, 1));
  }
}
@keyframes nt-ai-roof-hop {
  0% { transform: translateY(0) scale(1); }
  45% { transform: translateY(-2px) scale(1.06); }
  100% { transform: translateY(0) scale(1); }
}

/* 2. Personal 4-quadrant staggered spring pop */
.nt-ai-q0 { transform-origin: 7.4px 7.4px; }
.nt-ai-q1 { transform-origin: 16.6px 7.4px; }
.nt-ai-q2 { transform-origin: 7.4px 16.6px; }
.nt-ai-q3 { transform-origin: 16.6px 16.6px; }
@media (prefers-reduced-motion: no-preference) {
  .nt-anim-icon--personal:is(.is-active, .is-burst) .nt-ai-q0,
  :is(a, button):hover > .nt-anim-icon--personal .nt-ai-q0 { animation: nt-ai-q-pop 420ms var(--nt-ease-spring) 0ms; }
  .nt-anim-icon--personal:is(.is-active, .is-burst) .nt-ai-q1,
  :is(a, button):hover > .nt-anim-icon--personal .nt-ai-q1 { animation: nt-ai-q-pop 420ms var(--nt-ease-spring) 45ms; }
  .nt-anim-icon--personal:is(.is-active, .is-burst) .nt-ai-q2,
  :is(a, button):hover > .nt-anim-icon--personal .nt-ai-q2 { animation: nt-ai-q-pop 420ms var(--nt-ease-spring) 90ms; }
  .nt-anim-icon--personal:is(.is-active, .is-burst) .nt-ai-q3,
  :is(a, button):hover > .nt-anim-icon--personal .nt-ai-q3 { animation: nt-ai-q-pop 420ms var(--nt-ease-spring) 135ms; }
}
@keyframes nt-ai-q-pop {
  0% { transform: scale(1); }
  45% { transform: scale(1.18); }
  100% { transform: scale(1); }
}

/* 3. Gateway routing hub pulse */
.nt-ai-hub { transform-origin: 12px 12px; }
.nt-ai-node-0 { transform-origin: 12px 4.25px; }
.nt-ai-node-1 { transform-origin: 4.8px 16.5px; }
.nt-ai-node-2 { transform-origin: 19.2px 16.5px; }
@media (prefers-reduced-motion: no-preference) {
  .nt-anim-icon--gateway:is(.is-active, .is-burst) .nt-ai-hub,
  :is(a, button):hover > .nt-anim-icon--gateway .nt-ai-hub { animation: nt-ai-hub-pulse 460ms var(--nt-ease-spring); }
  .nt-anim-icon--gateway:is(.is-active, .is-burst) .nt-ai-node-0,
  :is(a, button):hover > .nt-anim-icon--gateway .nt-ai-node-0 { animation: nt-ai-q-pop 400ms var(--nt-ease-spring) 40ms; }
  .nt-anim-icon--gateway:is(.is-active, .is-burst) .nt-ai-node-1,
  :is(a, button):hover > .nt-anim-icon--gateway .nt-ai-node-1 { animation: nt-ai-q-pop 400ms var(--nt-ease-spring) 90ms; }
  .nt-anim-icon--gateway:is(.is-active, .is-burst) .nt-ai-node-2,
  :is(a, button):hover > .nt-anim-icon--gateway .nt-ai-node-2 { animation: nt-ai-q-pop 400ms var(--nt-ease-spring) 140ms; }
}
@keyframes nt-ai-hub-pulse {
  0% { transform: scale(1); }
  42% { transform: scale(1.22); }
  100% { transform: scale(1); }
}

/* 4. Radar sweep + pulse ring */
.nt-ai-sweep { transform-origin: 12px 12px; }
.nt-ai-ring { transform-origin: 12px 12px; opacity: 0; }
@media (prefers-reduced-motion: no-preference) {
  .nt-anim-icon--radar:is(.is-active, .is-burst) .nt-ai-sweep,
  :is(a, button):hover > .nt-anim-icon--radar .nt-ai-sweep { animation: nt-ai-radar-sweep 680ms cubic-bezier(0.22, 1, 0.36, 1); }
  .nt-anim-icon--radar:is(.is-active, .is-burst) .nt-ai-ring,
  :is(a, button):hover > .nt-anim-icon--radar .nt-ai-ring { animation: nt-ai-radar-ring 720ms cubic-bezier(0.22, 1, 0.36, 1); }
}
@keyframes nt-ai-radar-sweep {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
@keyframes nt-ai-radar-ring {
  0% { transform: scale(0.75); opacity: 0.65; }
  100% { transform: scale(1.38); opacity: 0; }
}

/* 5. Account check shield + drawn check */
.nt-ai-shield { transform-origin: 12px 12px; }
.nt-ai-check { stroke-dasharray: 16; stroke-dashoffset: 0; }
@media (prefers-reduced-motion: no-preference) {
  .nt-anim-icon--accountCheck:is(.is-active, .is-burst) .nt-ai-shield,
  :is(a, button):hover > .nt-anim-icon--accountCheck .nt-ai-shield { animation: nt-ai-hub-pulse 420ms var(--nt-ease-spring); }
  .nt-anim-icon--accountCheck:is(.is-active, .is-burst) .nt-ai-check,
  :is(a, button):hover > .nt-anim-icon--accountCheck .nt-ai-check { animation: nt-ai-check-draw 420ms cubic-bezier(0.22, 1, 0.36, 1); }
}
@keyframes nt-ai-check-draw {
  0% { stroke-dashoffset: 16; }
  100% { stroke-dashoffset: 0; }
}

/* 6. Help bubble spring tilt */
.nt-anim-icon--help svg { transform-origin: 12px 14px; }
@media (prefers-reduced-motion: no-preference) {
  .nt-anim-icon--help:is(.is-active, .is-burst) svg,
  :is(a, button):hover > .nt-anim-icon--help svg { animation: nt-ai-tilt-pop 460ms var(--nt-ease-spring); }
}
@keyframes nt-ai-tilt-pop {
  0% { transform: rotate(0deg) scale(1); }
  38% { transform: rotate(-9deg) scale(1.08); }
  72% { transform: rotate(4deg) scale(1.02); }
  100% { transform: rotate(0deg) scale(1); }
}

/* 7. Recharge bolt kick */
.nt-ai-bolt { transform-origin: 12px 12px; }
@media (prefers-reduced-motion: no-preference) {
  .nt-anim-icon--recharge:is(.is-active, .is-burst) .nt-ai-bolt,
  :is(a, button):hover > .nt-anim-icon--recharge .nt-ai-bolt { animation: nt-ai-bolt-kick 440ms var(--nt-ease-spring); }
}
@keyframes nt-ai-bolt-kick {
  0% { transform: scale(1) rotate(0deg); }
  40% { transform: scale(1.16) rotate(-7deg); }
  100% { transform: scale(1) rotate(0deg); }
}

/* 8. Rental box bounce */
.nt-anim-icon--rental svg { transform-origin: 12px 14px; }
@media (prefers-reduced-motion: no-preference) {
  .nt-anim-icon--rental:is(.is-active, .is-burst) svg,
  :is(a, button):hover > .nt-anim-icon--rental svg { animation: nt-ai-roof-hop 460ms var(--nt-ease-spring); }
}

/* 9. Telegram plane take-off */
.nt-anim-icon--telegram svg { transform-origin: 12px 12px; }
@media (prefers-reduced-motion: no-preference) {
  .nt-anim-icon--telegram.is-burst svg,
  :is(a, button):hover > .nt-anim-icon--telegram svg { animation: nt-ai-plane-fly 420ms var(--nt-ease-spring); }
}
@keyframes nt-ai-plane-fly {
  0% { transform: translate(0, 0) scale(1); }
  50% { transform: translate(2px, -2px) scale(1.08); }
  100% { transform: translate(0, 0) scale(1); }
}
</style>
