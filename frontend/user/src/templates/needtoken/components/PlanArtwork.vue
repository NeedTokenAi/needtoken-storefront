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
    <ellipse class="nt-plan-artwork__shadow" cx="58" cy="76" rx="36" ry="4" fill="#26392F" fill-opacity=".08" />

    <template v-if="plan === 'plus'">
      <g class="nt-plan-artwork__back">
        <path d="m31 22 57 8-7 43-57-8z" fill="#E5ECE5" stroke="#CDD8D0" />
        <path d="m32 27 50 7" stroke="#F9FBF8" />
      </g>
      <g class="nt-plan-artwork__front">
        <path d="m23 22 59-8 7 49-59 8z" fill="#D8DDD8" fill-opacity=".5" />
        <path d="m22 19 59-8 7 49-59 8z" fill="#FCFDF9" stroke="#C7D1C9" stroke-linejoin="round" />
        <path d="m27 24 49-6" stroke="#E5EAE3" />
        <path d="m35 59 23-3" stroke="#D7DFD6" stroke-width="2" stroke-linecap="round" />
        <path d="m35 53 35-5" stroke="#E4E9E1" stroke-linecap="round" />
        <path d="m31 32 10-1 1 9-10 1z" fill="var(--nt-plan-wash)" stroke="var(--nt-plan-ink)" stroke-opacity=".35" />
        <path d="m34 38-.5-4 4 3-.5-4" stroke="var(--nt-plan-stamp)" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round" />
        <path d="m74 13 7-1 2 15-4-2-3 3z" fill="var(--nt-plan-ink)" />
      </g>
      <g class="nt-plan-artwork__detail">
        <path d="m68 56 23-3 2 14-23 3z" fill="#F7F9F4" stroke="#CCD6CC" />
        <path d="m74 60 12-2m-11 6 7-1" stroke="var(--nt-plan-ink)" stroke-opacity=".6" stroke-linecap="round" />
      </g>
    </template>

    <template v-else-if="plan === '5x'">
      <g class="nt-plan-artwork__back">
        <path d="m15 29 34-7 8 6 32-5 8 41-35 7-10-4-30 5z" fill="var(--nt-plan-ink)" fill-opacity=".2" stroke="#B9C5B9" stroke-linejoin="round" />
        <path d="m21 68 30-6 11 5 32-6" stroke="#81917D" stroke-opacity=".45" />
      </g>
      <g class="nt-plan-artwork__front">
        <path d="m16 24 29-5c5-1 8 1 12 4l30-5 8 40-29 6c-5 1-9-1-13-3l-29 6z" fill="#F9FBF5" stroke="#C8D2C4" stroke-linejoin="round" />
        <path d="m18 23 26-5c5-1 8 1 12 4l8 39c-4-3-7-5-12-4l-26 5z" fill="#FFFEFA" stroke="#D3DACC" stroke-linejoin="round" />
        <path d="m56 22 29-6 8 39-29 6z" fill="#F4F6ED" stroke="#D3DACC" stroke-linejoin="round" />
        <path d="m56 23 8 37" stroke="#A8B4A0" />
        <path d="m27 32 19-4m-17 11 19-4m-18 11 20-4m-18 11 19-4" stroke="#DEE4D6" stroke-linecap="round" />
        <path d="m65 30 13-3m-12 10 17-4m-15 12 17-4m-15 12 11-3" stroke="#D9E0D0" stroke-linecap="round" />
        <path d="m35 27 3 15 2-3 3 2-3-15" fill="var(--nt-plan-ink)" />
        <path d="m74 47 1 5 4-1-1-5z" stroke="var(--nt-plan-stamp)" stroke-opacity=".6" />
      </g>
      <g class="nt-plan-artwork__detail">
        <path d="m79 68 16-31 3 2-16 31-3 2z" fill="#D4DACB" stroke="#A8B49E" stroke-linejoin="round" />
        <path d="m79 68 3 2-3 2z" fill="var(--nt-plan-ink)" />
        <path d="m95 37 2-3 3 2-2 3z" fill="var(--nt-plan-ink)" />
      </g>
    </template>

    <template v-else-if="plan === '20x'">
      <g class="nt-plan-artwork__back">
        <path d="m25 19 62 3 1 43-61-3z" fill="#DFE6E3" stroke="#BECBC4" stroke-linejoin="round" />
        <path d="m28 20 56 3" stroke="#F6F8F4" />
      </g>
      <g class="nt-plan-artwork__front">
        <path d="m23 17 62 3 1 42-62-3z" fill="#F7FAF6" stroke="#B8C7C0" stroke-linejoin="round" />
        <path d="m28 23 51 2 1 30-51-2z" fill="#E5ECE6" stroke="#CBD7CF" stroke-linejoin="round" />
        <path d="m28 23 51 2v6l-51-2z" fill="#F3F6F0" />
        <path d="m33 26 1 .1m3 .1 1 .1m3 .1 1 .1" stroke="#9DAC9F" stroke-width="1.5" stroke-linecap="round" />
        <path d="m35 35 4 3-4 3m8 0 5 .2" stroke="var(--nt-plan-ink)" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
        <path d="m35 46 19 1m4 .2 10 .4" stroke="#BBCAC0" stroke-linecap="round" />
        <path d="m70 50-.2-6 5 5-.2-6" stroke="var(--nt-plan-stamp)" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round" />
        <path d="m24 59 62 3 14 9-80-3z" fill="#E8EDE7" stroke="#BCC9C1" stroke-linejoin="round" />
        <path d="m20 68 80 3-3 3-75-3z" fill="#D8E0D9" stroke="#BCC9C1" stroke-linejoin="round" />
        <path d="m34 62 43 2 5 3-52-2z" fill="#C9D4CC" />
        <path d="m49 67 18 .7" stroke="#B2C1B6" stroke-linecap="round" />
      </g>
      <g class="nt-plan-artwork__detail">
        <path d="m13 53 9-1 2 11-9 1z" fill="#FDFDF9" stroke="#C7D1C7" />
        <path d="m15 55 4-.5m-3 4 3-.4" stroke="var(--nt-plan-ink)" stroke-opacity=".5" stroke-linecap="round" />
      </g>
    </template>

    <template v-else-if="plan === '50x'">
      <g class="nt-plan-artwork__back">
        <path d="m27 18 56 9-7 47-56-9z" fill="#E5E9DE" stroke="#C7CFC1" stroke-linejoin="round" />
        <path d="m24 59 49 8" stroke="#CDD6C7" />
        <path d="m78 28 3-9 10 3-3 9z" fill="var(--nt-plan-ink)" fill-opacity=".65" />
      </g>
      <g class="nt-plan-artwork__front">
        <path d="m34 16 55 5-4 50-55-5z" fill="#D8DDD3" fill-opacity=".6" />
        <path d="m33 13 55 5-4 50-55-5z" fill="#FDFDF7" stroke="#C8D1C2" stroke-linejoin="round" />
        <path d="m40 23 38 3-3 29-38-3z" fill="#E6EBDF" stroke="#D3DACB" />
        <path d="m41 48 9-21 8 22 9-20 6 22z" fill="var(--nt-plan-ink)" fill-opacity=".8" />
        <path d="m41 48 17 1-8-22z" fill="#A9BA9B" />
        <path d="m58 49 15 2-6-22z" fill="#B6C4A8" />
        <circle cx="70" cy="32" r="3" fill="#F8FAF3" />
        <path d="m44 58 23 2" stroke="#D4DCCD" stroke-linecap="round" />
        <path d="m75 60 .3-4 3 3 .3-4" stroke="var(--nt-plan-stamp)" stroke-linecap="round" stroke-linejoin="round" />
      </g>
      <g class="nt-plan-artwork__detail">
        <path d="m24 36 3-16 8 1-3 17" stroke="#A6B09F" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        <path d="m27 34 2-11 3 .5-2 11" stroke="#E8ECE1" stroke-linecap="round" />
      </g>
    </template>

    <template v-else>
      <g class="nt-plan-artwork__back">
        <path d="m25 26 58-7 5 47-58 7z" fill="#E5EBE3" stroke="#C8D2C7" />
      </g>
      <g class="nt-plan-artwork__front">
        <path d="m31 17 42 1 12 12-1 36-54-1z" fill="#FCFDF8" stroke="#C8D2C7" stroke-linejoin="round" />
        <path d="m73 18-.3 12 12 .3" fill="#EAF0E6" stroke="#C8D2C7" stroke-linejoin="round" />
        <path d="m41 48 .2-12 11 12 .2-12" stroke="var(--nt-plan-stamp)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
        <path d="m41 56 28 .5" stroke="#DDE5D9" stroke-linecap="round" />
      </g>
      <g class="nt-plan-artwork__detail">
        <path d="m67 59 20-2 1 11-20 2z" fill="#F0F4EB" stroke="#C8D2C7" />
        <path d="m73 63 8-1" stroke="var(--nt-plan-ink)" stroke-opacity=".5" stroke-linecap="round" />
      </g>
    </template>
  </svg>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  plan: 'plus' | '5x' | '20x' | '50x' | 'neutral'
  brand?: 'gpt' | 'claude' | 'unknown'
}>(), {
  brand: 'unknown',
})
</script>

<style scoped>
.nt-plan-artwork {
  --nt-plan-ink: #647E6B;
  --nt-plan-wash: #E5ECE0;
  --nt-plan-stamp: var(--nt-plan-ink);
  display: block;
  width: 100%;
  height: auto;
  overflow: visible;
  pointer-events: none;
}

.nt-plan-artwork--5x { --nt-plan-ink: #788367; }
.nt-plan-artwork--20x { --nt-plan-ink: #607D7C; }
.nt-plan-artwork--50x { --nt-plan-ink: #8B8768; }
.nt-plan-artwork--neutral { --nt-plan-ink: #7E8C7A; }
.nt-plan-artwork--gpt { --nt-plan-stamp: #5C7A68; }
.nt-plan-artwork--claude { --nt-plan-stamp: #9B7964; }

.nt-plan-artwork__front,
.nt-plan-artwork__back,
.nt-plan-artwork__detail,
.nt-plan-artwork__shadow {
  transform-box: fill-box;
  transform-origin: 50% 70%;
  transition: transform 180ms cubic-bezier(.22, 1, .36, 1), opacity 180ms cubic-bezier(.22, 1, .36, 1);
}

/* The card owns interaction; the paper layers separate as it receives focus. */
:global(.home-price-card:is(:hover, :focus-visible, :focus-within) .nt-plan-artwork__front) {
  transform: translateY(-2px) rotate(-1deg);
}

:global(.home-price-card:is(:hover, :focus-visible, :focus-within) .nt-plan-artwork__back) {
  transform: translate(1px, 1px) rotate(1deg);
}

:global(.home-price-card:is(:hover, :focus-visible, :focus-within) .nt-plan-artwork__detail) {
  transform: translate(1px, -1px);
}

:global(.home-price-card:is(:hover, :focus-visible, :focus-within) .nt-plan-artwork__shadow) {
  opacity: .6;
  transform: translateY(1px);
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
