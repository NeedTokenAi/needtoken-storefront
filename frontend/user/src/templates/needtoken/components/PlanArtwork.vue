<template>
  <svg
    class="nt-plan-artwork"
    :class="[`nt-plan-artwork--${plan}`, `nt-plan-artwork--${brand}`]"
    viewBox="0 0 112 88"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    role="presentation"
    aria-hidden="true"
    focusable="false"
  >
    <ellipse class="nt-plan-artwork__shadow" cx="58" cy="76" rx="36" ry="4" fill="var(--nt-art-shadow, rgba(38,57,47,0.1))" />

    <template v-if="plan === 'plus'">
      <g class="nt-plan-artwork__back">
        <path d="m31 22 57 8-7 43-57-8z" fill="var(--nt-art-paper-back)" stroke="var(--nt-art-stroke)" />
        <path d="m32 27 50 7" stroke="var(--nt-art-line)" />
      </g>
      <g class="nt-plan-artwork__front">
        <path d="m23 22 59-8 7 49-59 8z" fill="var(--nt-art-paper-mid)" fill-opacity=".45" />
        <path d="m22 19 59-8 7 49-59 8z" fill="var(--nt-art-paper)" stroke="var(--nt-art-stroke)" stroke-linejoin="round" />
        <path d="m27 24 49-6" stroke="var(--nt-art-line)" />
        <path d="m35 59 23-3" stroke="var(--nt-art-line)" stroke-width="2" stroke-linecap="round" />
        <path d="m35 53 35-5" stroke="var(--nt-art-line)" stroke-linecap="round" />
        <path d="m31 32 10-1 1 9-10 1z" fill="var(--nt-plan-wash)" stroke="var(--nt-plan-ink)" stroke-opacity=".42" />
        <path d="m34 38-.5-4 4 3-.5-4" stroke="var(--nt-plan-stamp)" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round" />
        <path d="m74 13 7-1 2 15-4-2-3 3z" fill="var(--nt-plan-ink)" />
      </g>
      <g class="nt-plan-artwork__detail">
        <path d="m68 56 23-3 2 14-23 3z" fill="var(--nt-art-paper-alt)" stroke="var(--nt-art-stroke)" />
        <path d="m74 60 12-2m-11 6 7-1" stroke="var(--nt-plan-ink)" stroke-opacity=".68" stroke-linecap="round" />
      </g>
    </template>

    <template v-else-if="plan === '5x'">
      <g class="nt-plan-artwork__back">
        <path d="m15 29 34-7 8 6 32-5 8 41-35 7-10-4-30 5z" fill="var(--nt-plan-ink)" fill-opacity=".18" stroke="var(--nt-art-stroke)" stroke-linejoin="round" />
        <path d="m21 68 30-6 11 5 32-6" stroke="var(--nt-art-stroke)" stroke-opacity=".55" />
      </g>
      <g class="nt-plan-artwork__front">
        <path d="m16 24 29-5c5-1 8 1 12 4l30-5 8 40-29 6c-5 1-9-1-13-3l-29 6z" fill="var(--nt-art-paper-alt)" stroke="var(--nt-art-stroke)" stroke-linejoin="round" />
        <path d="m18 23 26-5c5-1 8 1 12 4l8 39c-4-3-7-5-12-4l-26 5z" fill="var(--nt-art-paper)" stroke="var(--nt-art-stroke)" stroke-linejoin="round" />
        <path d="m56 22 29-6 8 39-29 6z" fill="var(--nt-art-paper-back)" stroke="var(--nt-art-stroke)" stroke-linejoin="round" />
        <path d="m56 23 8 37" stroke="var(--nt-art-stroke)" />
        <path d="m27 32 19-4m-17 11 19-4m-18 11 20-4m-18 11 19-4" stroke="var(--nt-art-line)" stroke-linecap="round" />
        <path d="m65 30 13-3m-12 10 17-4m-15 12 17-4m-15 12 11-3" stroke="var(--nt-art-line)" stroke-linecap="round" />
        <path d="m35 27 3 15 2-3 3 2-3-15" fill="var(--nt-plan-ink)" />
        <path d="m74 47 1 5 4-1-1-5z" stroke="var(--nt-plan-stamp)" stroke-opacity=".7" />
      </g>
      <g class="nt-plan-artwork__detail">
        <path d="m79 68 16-31 3 2-16 31-3 2z" fill="var(--nt-art-paper-mid)" stroke="var(--nt-art-stroke)" stroke-linejoin="round" />
        <path d="m79 68 3 2-3 2z" fill="var(--nt-plan-ink)" />
        <path d="m95 37 2-3 3 2-2 3z" fill="var(--nt-plan-ink)" />
      </g>
    </template>

    <template v-else-if="plan === '20x'">
      <g class="nt-plan-artwork__back">
        <path d="m25 19 62 3 1 43-61-3z" fill="var(--nt-art-paper-back)" stroke="var(--nt-art-stroke)" stroke-linejoin="round" />
        <path d="m28 20 56 3" stroke="var(--nt-art-line)" />
      </g>
      <g class="nt-plan-artwork__front">
        <path d="m23 17 62 3 1 42-62-3z" fill="var(--nt-art-paper)" stroke="var(--nt-art-stroke)" stroke-linejoin="round" />
        <path d="m28 23 51 2 1 30-51-2z" fill="var(--nt-art-paper-back)" stroke="var(--nt-art-stroke)" stroke-linejoin="round" />
        <path d="m28 23 51 2v6l-51-2z" fill="var(--nt-art-paper-alt)" />
        <path d="m33 26 1 .1m3 .1 1 .1m3 .1 1 .1" stroke="var(--nt-plan-ink)" stroke-opacity=".6" stroke-width="1.5" stroke-linecap="round" />
        <path d="m35 35 4 3-4 3m8 0 5 .2" stroke="var(--nt-plan-ink)" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
        <path d="m35 46 19 1m4 .2 10 .4" stroke="var(--nt-art-line)" stroke-linecap="round" />
        <path d="m70 50-.2-6 5 5-.2-6" stroke="var(--nt-plan-stamp)" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round" />
        <path d="m24 59 62 3 14 9-80-3z" fill="var(--nt-art-paper-alt)" stroke="var(--nt-art-stroke)" stroke-linejoin="round" />
        <path d="m20 68 80 3-3 3-75-3z" fill="var(--nt-art-paper-mid)" stroke="var(--nt-art-stroke)" stroke-linejoin="round" />
        <path d="m34 62 43 2 5 3-52-2z" fill="var(--nt-art-paper-back)" />
        <path d="m49 67 18 .7" stroke="var(--nt-art-stroke)" stroke-linecap="round" />
      </g>
      <g class="nt-plan-artwork__detail">
        <path d="m13 53 9-1 2 11-9 1z" fill="var(--nt-art-paper)" stroke="var(--nt-art-stroke)" />
        <path d="m15 55 4-.5m-3 4 3-.4" stroke="var(--nt-plan-ink)" stroke-opacity=".6" stroke-linecap="round" />
      </g>
    </template>

    <template v-else-if="plan === '50x'">
      <g class="nt-plan-artwork__back">
        <path d="m27 18 56 9-7 47-56-9z" fill="var(--nt-art-paper-back)" stroke="var(--nt-art-stroke)" stroke-linejoin="round" />
        <path d="m24 59 49 8" stroke="var(--nt-art-line)" />
        <path d="m78 28 3-9 10 3-3 9z" fill="var(--nt-plan-ink)" fill-opacity=".65" />
      </g>
      <g class="nt-plan-artwork__front">
        <path d="m34 16 55 5-4 50-55-5z" fill="var(--nt-art-paper-mid)" fill-opacity=".55" />
        <path d="m33 13 55 5-4 50-55-5z" fill="var(--nt-art-paper)" stroke="var(--nt-art-stroke)" stroke-linejoin="round" />
        <path d="m40 23 38 3-3 29-38-3z" fill="var(--nt-art-paper-back)" stroke="var(--nt-art-stroke)" />
        <path d="m41 48 9-21 8 22 9-20 6 22z" fill="var(--nt-plan-ink)" fill-opacity=".75" />
        <path d="m41 48 17 1-8-22z" fill="var(--nt-plan-ink)" fill-opacity=".42" />
        <path d="m58 49 15 2-6-22z" fill="var(--nt-plan-ink)" fill-opacity=".55" />
        <circle cx="70" cy="32" r="3" fill="var(--nt-art-paper)" />
        <path d="m44 58 23 2" stroke="var(--nt-art-line)" stroke-linecap="round" />
        <path d="m75 60 .3-4 3 3 .3-4" stroke="var(--nt-plan-stamp)" stroke-linecap="round" stroke-linejoin="round" />
      </g>
      <g class="nt-plan-artwork__detail">
        <path d="m24 36 3-16 8 1-3 17" stroke="var(--nt-art-stroke)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        <path d="m27 34 2-11 3 .5-2 11" stroke="var(--nt-art-line)" stroke-linecap="round" />
      </g>
    </template>

    <template v-else>
      <g class="nt-plan-artwork__back">
        <path d="m25 26 58-7 5 47-58 7z" fill="var(--nt-art-paper-back)" stroke="var(--nt-art-stroke)" />
      </g>
      <g class="nt-plan-artwork__front">
        <path d="m31 17 42 1 12 12-1 36-54-1z" fill="var(--nt-art-paper)" stroke="var(--nt-art-stroke)" stroke-linejoin="round" />
        <path d="m73 18-.3 12 12 .3" fill="var(--nt-art-paper-alt)" stroke="var(--nt-art-stroke)" stroke-linejoin="round" />
        <path d="m41 48 .2-12 11 12 .2-12" stroke="var(--nt-plan-stamp)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        <path d="m41 56 28 .5" stroke="var(--nt-art-line)" stroke-linecap="round" />
      </g>
      <g class="nt-plan-artwork__detail">
        <path d="m67 59 20-2 1 11-20 2z" fill="var(--nt-art-paper-alt)" stroke="var(--nt-art-stroke)" />
        <path d="m73 63 8-1" stroke="var(--nt-plan-ink)" stroke-opacity=".55" stroke-linecap="round" />
      </g>
    </template>
  </svg>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  plan: 'plus' | '5x' | '20x' | '50x' | 'neutral'
  brand?: 'gpt' | 'claude' | 'meta' | 'gemini' | 'grok' | 'unknown'
}>(), {
  brand: 'unknown',
})
</script>

<style scoped>
.nt-plan-artwork {
  --nt-plan-ink: var(--ui-accent, #0d9473);
  --nt-plan-wash: var(--ui-accent-soft, rgba(13, 148, 115, 0.14));
  --nt-plan-stamp: var(--ui-accent, #0d9473);
  display: block;
  width: 100%;
  height: auto;
  overflow: visible;
  pointer-events: none;
}

.nt-plan-artwork--5x { --nt-plan-ink: var(--nt-tier-5x, #475b74); }
.nt-plan-artwork--20x { --nt-plan-ink: var(--nt-tier-20x, #966622); }
.nt-plan-artwork--50x { --nt-plan-ink: var(--nt-tier-50x, #7c4d93); }
.nt-plan-artwork--claude { --nt-plan-stamp: var(--nt-brand-claude, #c25d3f); }

.nt-plan-artwork__front,
.nt-plan-artwork__back,
.nt-plan-artwork__detail,
.nt-plan-artwork__shadow {
  transform-box: fill-box;
  transform-origin: 50% 70%;
  transition: transform 440ms var(--nt-ease-spring, cubic-bezier(0.34, 1.4, 0.64, 1)), opacity 260ms ease;
}

/* Layered paper parallax inside the 3D card hover container. */
:global(.home-price-card:is(:hover, :focus-visible, :focus-within) .nt-plan-artwork__front) {
  transform: translate3d(0, -4px, 0) rotate(-2deg) scale(1.03);
}

:global(.home-price-card:is(:hover, :focus-visible, :focus-within) .nt-plan-artwork__back) {
  transform: translate3d(2px, 1.5px, 0) rotate(2deg);
}

:global(.home-price-card:is(:hover, :focus-visible, :focus-within) .nt-plan-artwork__detail) {
  transform: translate3d(2.5px, -3px, 0) rotate(1.5deg);
}

:global(.home-price-card:is(:hover, :focus-visible, :focus-within) .nt-plan-artwork__shadow) {
  opacity: .55;
  transform: translateY(2px) scaleX(1.06);
}

@media (prefers-reduced-motion: reduce) {
  .nt-plan-artwork__front,
  .nt-plan-artwork__back,
  .nt-plan-artwork__detail,
  .nt-plan-artwork__shadow {
    transition: none;
  }

  :global(.home-price-card:is(:hover, :focus-visible, :focus-within) :is(.nt-plan-artwork__front, .nt-plan-artwork__back, .nt-plan-artwork__detail, .nt-plan-artwork__shadow)) {
    transform: none;
  }
}
</style>
