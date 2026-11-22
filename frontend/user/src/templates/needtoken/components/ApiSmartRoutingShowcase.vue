<template>
  <section class="routing-showcase" aria-labelledby="api-smart-routing-title">
    <div class="routing-stage">
      <div class="routing-stage-bg" aria-hidden="true" />
      <header class="routing-stage-head">
        <span class="routing-eyebrow">{{ t('needtoken.apiServices.routing.eyebrow') }}</span>
        <h2 id="api-smart-routing-title">{{ t('needtoken.apiServices.routing.title') }}</h2>
        <p>{{ t('needtoken.apiServices.routing.subtitle') }}</p>
      </header>

      <div class="routing-diagram">
        <!-- 左侧：Model Providers -->
        <div class="routing-column routing-providers">
          <p class="routing-col-label">
            <span class="routing-col-dot" aria-hidden="true" />
            <span>MODEL PROVIDERS</span>
          </p>
          <ul class="routing-node-list">
            <li v-for="p in providers" :key="p.id" class="routing-node">
              <span class="routing-node-badge" :style="{ '--node-color': p.color }" aria-hidden="true">{{ p.mark }}</span>
              <div class="routing-node-text">
                <strong translate="no">{{ p.name }}</strong>
                <small>{{ p.models }}</small>
              </div>
            </li>
          </ul>
        </div>

        <!-- 中间：Smart Routing Hub + SVG 贝塞尔连线 -->
        <div class="routing-center" aria-hidden="true">
          <svg class="routing-wires" viewBox="0 0 360 220" preserveAspectRatio="none">
            <defs>
              <linearGradient id="nt-wire-left" x1="0%" y1="0%" x2="100%" y2="0%">
                <stop offset="0%" stop-color="var(--ui-accent)" stop-opacity="0.22" />
                <stop offset="100%" stop-color="var(--ui-accent)" stop-opacity="0.85" />
              </linearGradient>
              <linearGradient id="nt-wire-right" x1="0%" y1="0%" x2="100%" y2="0%">
                <stop offset="0%" stop-color="var(--ui-accent)" stop-opacity="0.85" />
                <stop offset="100%" stop-color="#f97316" stop-opacity="0.55" />
              </linearGradient>
            </defs>
            <!-- Left curves into hub -->
            <path d="M 0 32 C 85 32, 110 110, 175 110" fill="none" stroke="url(#nt-wire-left)" stroke-width="2" />
            <path d="M 0 84 C 85 84, 110 110, 175 110" fill="none" stroke="url(#nt-wire-left)" stroke-width="2" />
            <path d="M 0 136 C 85 136, 110 110, 175 110" fill="none" stroke="url(#nt-wire-left)" stroke-width="2" />
            <path d="M 0 188 C 85 188, 110 110, 175 110" fill="none" stroke="url(#nt-wire-left)" stroke-width="2" />
            <!-- Right curves out of hub -->
            <path d="M 185 110 C 250 110, 275 32, 360 32" fill="none" stroke="url(#nt-wire-right)" stroke-width="2" />
            <path d="M 185 110 C 250 110, 275 84, 360 84" fill="none" stroke="url(#nt-wire-right)" stroke-width="2" />
            <path d="M 185 110 C 250 110, 275 136, 360 136" fill="none" stroke="url(#nt-wire-right)" stroke-width="2" />
            <path d="M 185 110 C 250 110, 275 188, 360 188" fill="none" stroke="url(#nt-wire-right)" stroke-width="2" />
          </svg>
          <div class="routing-hub">
            <div class="routing-hub-ring">
              <img src="/needtoken-icon.svg" alt="" />
            </div>
            <span class="routing-hub-label">
              <i class="hub-dot" />
              <span>{{ t('needtoken.apiServices.routing.hubLabel') }}</span>
            </span>
          </div>
        </div>

        <!-- 右侧：Coding Agents & Clients -->
        <div class="routing-column routing-clients">
          <p class="routing-col-label routing-col-label-right">
            <span class="routing-col-dot" aria-hidden="true" />
            <span>CODING AGENTS &amp; CLIENTS</span>
          </p>
          <ul class="routing-node-list">
            <li v-for="c in clients" :key="c.id" class="routing-node">
              <span class="routing-node-badge" :style="{ '--node-color': c.color }" aria-hidden="true">{{ c.mark }}</span>
              <div class="routing-node-text">
                <strong translate="no">{{ c.name }}</strong>
                <small>{{ t(`needtoken.apiServices.routing.clients.${c.id}`) }}</small>
              </div>
            </li>
          </ul>
          <p class="routing-more-note">— {{ t('needtoken.apiServices.routing.worksWithMore') }} —</p>
        </div>
      </div>

      <!-- 底部 6 个核心能力胶囊 -->
      <ul class="routing-pills" :aria-label="t('needtoken.apiServices.routing.featuresLabel')">
        <li v-for="feat in features" :key="feat.key" class="routing-pill">
          <component :is="feat.icon" :size="15" aria-hidden="true" />
          <span>{{ t(`needtoken.apiServices.routing.features.${feat.key}`) }}</span>
        </li>
      </ul>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Database, GitFork, LineChart, Receipt, RefreshCw, ShieldCheck } from 'lucide-vue-next'

const { t } = useI18n()

const providers = [
  { id: 'anthropic', name: 'Claude', mark: 'C', color: '#d97706', models: 'Opus · Sonnet · Haiku' },
  { id: 'openai', name: 'OpenAI GPT', mark: 'O', color: '#10b981', models: 'GPT-5 · Codex · o3' },
  { id: 'google', name: 'Gemini', mark: 'G', color: '#3b82f6', models: 'Gemini 2.5 Pro · Flash' },
  { id: 'xai', name: 'xAI & Multimodal', mark: 'X', color: '#8b5cf6', models: 'Grok · Image · Video' },
] as const

const clients = [
  { id: 'claudeCode', name: 'Claude Code', mark: 'CC', color: '#ea580c' },
  { id: 'codex', name: 'Codex CLI', mark: 'CX', color: '#059669' },
  { id: 'ccSwitch', name: 'CC Switch', mark: 'SW', color: '#0284c7' },
  { id: 'openClaw', name: 'OpenClaw', mark: 'OC', color: '#dc2626' },
] as const

const features = [
  { key: 'smartRouting', icon: GitFork },
  { key: 'promptCaching', icon: Database },
  { key: 'autoFallbacks', icon: RefreshCw },
  { key: 'oneBill', icon: Receipt },
  { key: 'spendTracking', icon: LineChart },
  { key: 'neverDowngraded', icon: ShieldCheck },
] as const
</script>

<style scoped>
.routing-showcase { padding-block: 44px; border-top: 1px solid var(--ui-border); }
.routing-stage {
  position: relative;
  overflow: hidden;
  padding: 36px 32px 30px;
  border: 1px solid var(--ui-border);
  border-radius: 24px;
  background: var(--ui-bg-elevated);
}
.routing-stage-bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    radial-gradient(60% 75% at 14% 85%, rgba(251, 146, 60, 0.16), transparent 68%),
    radial-gradient(50% 65% at 88% 20%, rgba(56, 189, 248, 0.15), transparent 68%),
    radial-gradient(45% 55% at 50% 50%, rgba(16, 185, 129, 0.12), transparent 65%);
}
.dark .routing-stage-bg {
  background:
    radial-gradient(60% 75% at 14% 85%, rgba(251, 146, 60, 0.12), transparent 68%),
    radial-gradient(50% 65% at 88% 20%, rgba(56, 189, 248, 0.12), transparent 68%),
    radial-gradient(45% 55% at 50% 50%, rgba(16, 185, 129, 0.14), transparent 65%);
}
.routing-stage-head { position: relative; z-index: 1; text-align: center; max-width: 680px; margin-inline: auto; }
.routing-eyebrow { font-size: 11.5px; font-weight: 700; letter-spacing: .12em; text-transform: uppercase; color: var(--ui-accent); }
.routing-stage-head h2 { margin-top: 8px; font-size: clamp(22px, 2.6vw, 30px); font-weight: 700; letter-spacing: -.025em; color: var(--ui-text-primary); }
.routing-stage-head p { margin-top: 8px; font-size: 13.5px; line-height: 1.75; color: var(--ui-text-secondary); }

.routing-diagram {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: minmax(0, 230px) minmax(0, 1fr) minmax(0, 230px);
  align-items: center;
  gap: 0;
  margin-top: 28px;
}
.routing-col-label {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  margin-bottom: 12px;
  font: 600 11px var(--nt-font-mono);
  letter-spacing: .08em;
  color: var(--ui-text-muted);
}
.routing-col-label-right { display: flex; justify-content: flex-end; }
.routing-col-dot { width: 6px; height: 6px; border-radius: 2px; background: var(--ui-accent); }

.routing-node-list { display: grid; gap: 10px; list-style: none; padding: 0; margin: 0; }
.routing-node {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 13px;
  border: 1px solid var(--ui-border);
  border-radius: 13px;
  background: color-mix(in srgb, var(--ui-bg-elevated) 88%, transparent);
  backdrop-filter: blur(8px);
  box-shadow: 0 4px 16px rgba(15, 23, 42, 0.05);
}
.routing-node-badge {
  display: grid;
  place-items: center;
  flex: none;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--node-color) 14%, transparent);
  color: var(--node-color);
  font: 700 12px var(--nt-font-mono);
}
.routing-node-text { min-width: 0; }
.routing-node-text strong { display: block; font-size: 13px; font-weight: 650; color: var(--ui-text-primary); line-height: 1.25; }
.routing-node-text small { display: block; margin-top: 2px; font-size: 11px; color: var(--ui-text-muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

.routing-center { position: relative; display: grid; place-items: center; min-height: 224px; }
.routing-wires { position: absolute; inset: 0; width: 100%; height: 100%; }
.routing-hub { position: relative; z-index: 2; display: flex; flex-direction: column; align-items: center; gap: 10px; }
.routing-hub-ring {
  display: grid;
  place-items: center;
  width: 78px;
  height: 78px;
  border-radius: 999px;
  border: 1px solid var(--nt-accent-border-strong, var(--ui-accent));
  background: var(--ui-bg-elevated);
  box-shadow: 0 0 0 10px color-mix(in srgb, var(--ui-accent) 12%, transparent), 0 12px 30px rgba(15, 23, 42, 0.12);
}
.routing-hub-ring img { width: 38px; height: 38px; }
.routing-hub-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 999px;
  border: 1px solid var(--ui-border);
  background: var(--ui-bg-elevated);
  font-size: 12px;
  font-weight: 650;
  color: var(--ui-text-primary);
}
.hub-dot { width: 7px; height: 7px; border-radius: 999px; background: var(--ui-accent); }
.routing-more-note { margin-top: 10px; text-align: right; font-size: 11.5px; color: var(--ui-text-muted); }

.routing-pills {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 10px;
  margin-top: 28px;
  padding: 0;
  list-style: none;
}
.routing-pill {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  min-height: 42px;
  padding: 8px 10px;
  border: 1px solid var(--ui-border);
  border-radius: 11px;
  background: color-mix(in srgb, var(--ui-bg-elevated) 84%, transparent);
  backdrop-filter: blur(8px);
  font-size: 12.5px;
  font-weight: 600;
  color: var(--ui-text-primary);
  text-align: center;
}
.routing-pill svg { flex: none; color: var(--ui-accent); }

@media (max-width: 960px) {
  .routing-diagram { grid-template-columns: minmax(0, 1fr); gap: 18px; }
  .routing-wires { display: none; }
  .routing-center { min-height: auto; padding-block: 8px; }
  .routing-col-label-right { justify-content: flex-start; }
  .routing-more-note { text-align: left; }
  .routing-node-list { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .routing-pills { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}
@media (max-width: 560px) {
  .routing-stage { padding: 24px 16px 20px; }
  .routing-node-list { grid-template-columns: minmax(0, 1fr); }
  .routing-pills { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
</style>
