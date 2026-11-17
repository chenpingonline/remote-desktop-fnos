<script setup lang="ts">
import { computed } from "vue";
import { Check, Monitor, X, Minus } from "lucide-vue-next";
export interface ProgressState {
  token: number;
  connection: { id: string; name: string; host: string; protocol: string };
  step: number;
  mode: "running" | "certificate" | "login" | "failed" | "ready" | "cancelled";
  error: string;
}
const props = defineProps<{ progress: ProgressState }>();
const emit = defineEmits<{ retry: []; cancel: []; resume: [] }>();
const titles = [
  "正在检查远程证书",
  "正在准备登录信息",
  "正在连接远程电脑",
  "正在等待远程画面",
];
const title = computed(() =>
  props.progress.mode === "failed"
    ? "远程连接未完成"
    : props.progress.mode === "certificate"
      ? "等待确认远程证书"
      : props.progress.mode === "login"
        ? "等待输入登录信息"
        : titles[props.progress.step],
);
const steps = computed(() => [
  {
    label: "检查远程证书",
    detail:
      props.progress.connection.protocol === "vnc"
        ? "VNC 跳过此步"
        : props.progress.mode === "certificate"
          ? "等待确认"
          : "RDP 证书校验",
  },
  {
    label: "准备登录信息",
    detail:
      props.progress.mode === "login"
        ? "等待输入账号密码"
        : "保存的凭据或本次登录",
  },
  { label: "建立远程连接", detail: "创建会话并连接桌面通道" },
  { label: "等待远程画面", detail: "接收并渲染首帧画面" },
]);
</script>
<template>
  <section
    role="region"
    class="connect"
    aria-label="连接进度"
    :aria-busy="progress.mode === 'running'"
  >
    <div class="connect__grid" aria-hidden="true" />
    <button
      class="connect__close"
      aria-label="取消连接"
      @click="emit('cancel')"
    >
      <X :size="19" />
    </button>
    <div class="connect__center">
      <div class="connect__pulse" aria-hidden="true">
        <span v-if="progress.mode !== 'failed'" class="connect__ring" /><span
          v-if="progress.mode !== 'failed'"
          class="connect__ring connect__ring--2"
        /><span class="connect__core"
          ><Monitor :size="27" :stroke-width="1.7"
        /></span>
      </div>
      <p class="connect__eyebrow">REMOTE SESSION</p>
      <h2 aria-live="polite">{{ title }}</h2>
      <p class="connect__name">{{ progress.connection.name }}</p>
      <p class="connect__host">
        {{ progress.connection.host }}<span class="connect__dot">·</span
        >{{ progress.connection.protocol.toUpperCase() }}
      </p>
      <ol class="connect__steps">
        <li
          v-for="(step, index) in steps"
          :key="step.label"
          class="connect__step"
          :class="{
            'is-done': progress.step > index,
            'is-current': progress.step === index,
            'is-failed': progress.step === index && progress.mode === 'failed',
          }"
          :aria-current="progress.step === index ? 'step' : undefined"
          :data-step="index"
        >
          <span class="connect__check" aria-hidden="true"
            ><Minus
              v-if="index === 0 && progress.connection.protocol === 'vnc'"
              :size="12" /><Check
              v-else-if="progress.step > index"
              :size="12" /><X
              v-else-if="progress.step === index && progress.mode === 'failed'"
              :size="12" /><span v-else class="connect__step-dot"
          /></span>
          <span class="connect__step-copy"
            ><span>{{ step.label }}</span
            ><span class="connect__step-detail">{{ step.detail }}</span></span
          >
          <span
            v-if="progress.step === index && progress.mode === 'running'"
            class="connect__spinner"
            aria-hidden="true"
          />
        </li>
      </ol>
      <p v-if="progress.error" class="connect__error" role="alert">
        {{ progress.error }}
      </p>
      <div class="connect__actions">
        <button
          v-if="progress.mode === 'failed'"
          class="connect__primary"
          @click="emit('retry')"
        >
          重试连接
        </button>
        <button
          v-else-if="
            progress.mode === 'certificate' || progress.mode === 'login'
          "
          class="connect__primary"
          @click="emit('resume')"
        >
          {{
            progress.mode === "certificate" ? "确认远程证书" : "输入登录信息"
          }}
        </button>
        <button @click="emit('cancel')">返回管理</button>
      </div>
    </div>
  </section>
</template>
<style scoped>
.connect {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  overflow: hidden;
  background:
    radial-gradient(
      circle at 50% 36%,
      rgba(215, 251, 98, 0.08),
      transparent 28%
    ),
    radial-gradient(
      circle at 75% 75%,
      rgba(99, 114, 90, 0.14),
      transparent 35%
    ),
    #080b09;
}

.connect__grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.035) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.035) 1px, transparent 1px);
  background-size: 56px 56px;
  mask-image: radial-gradient(circle at 50% 45%, #000 0%, transparent 70%);
}

.connect__close {
  position: absolute;
  top: 24px;
  right: 24px;
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 10px;
  color: #8b9688;
  transition:
    color 0.16s ease,
    background-color 0.16s ease;
}

.connect__close:hover {
  background: rgba(255, 255, 255, 0.06);
  color: #fff;
}

.connect__center {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  max-width: 420px;
  padding: 0 24px;
  text-align: center;
}

.connect__pulse {
  position: relative;
  display: grid;
  place-items: center;
  width: 102px;
  height: 102px;
  margin-bottom: 33px;
}

.connect__ring {
  position: absolute;
  inset: 0;
  border: 1px solid rgba(215, 251, 98, 0.16);
  border-radius: 50%;
  animation: ring-pulse 2.8s ease-out infinite;
}

.connect__ring--2 {
  animation-delay: 1.4s;
}

@keyframes ring-pulse {
  0% {
    transform: scale(0.72);
    opacity: 0.7;
  }
  100% {
    transform: scale(1.5);
    opacity: 0;
  }
}

.connect__core {
  position: relative;
  z-index: 1;
  display: grid;
  place-items: center;
  width: 70px;
  height: 70px;
  border: 1px solid rgba(215, 251, 98, 0.25);
  border-radius: 22px;
  background: linear-gradient(145deg, #20291c, #111610);
  color: var(--accent);
  box-shadow: 0 0 50px rgba(215, 251, 98, 0.1);
}

.connect__eyebrow {
  margin-bottom: 9px;
  color: var(--accent);
  font-size: 9px;
  font-weight: 750;
  letter-spacing: 0.19em;
}

.connect h2 {
  color: #f5f7ef;
  font-size: 24px;
  font-weight: 650;
  letter-spacing: -0.035em;
}

.connect__host {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  color: #7f8a7b;
  font-size: 11.5px;
}

.connect__dot {
  color: #4e594b;
}

.connect__steps {
  width: 100%;
  margin-top: 42px;
  padding: 8px 0 2px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}

.connect__step {
  display: flex;
  align-items: center;
  gap: 11px;
  height: 45px;
  color: #566152;
  text-align: left;
  transition: color 0.25s ease;
}

.connect__step.is-done {
  color: #aab3a5;
}

.connect__step.is-current {
  color: #edf1e9;
}

.connect__check {
  display: grid;
  place-items: center;
  width: 20px;
  height: 20px;
  flex: 0 0 auto;
  border: 1px solid rgba(255, 255, 255, 0.11);
  border-radius: 50%;
  font-size: 9px;
  font-weight: 750;
}

.connect__step.is-done .connect__check {
  border-color: rgba(215, 251, 98, 0.25);
  background: rgba(215, 251, 98, 0.1);
  color: var(--accent);
}

.connect__step.is-current .connect__check {
  border-color: rgba(255, 255, 255, 0.3);
}

.connect__step-dot {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: #677263;
}

.connect__step-copy {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  font-size: 11.5px;
  font-weight: 500;
}

.connect__step-detail {
  color: #596454;
  font-size: 9.5px;
}

.connect__step.is-current .connect__step-detail {
  color: #7f8b7b;
}

.connect__spinner {
  width: 12px;
  height: 12px;
  border: 1.5px solid rgba(215, 251, 98, 0.2);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.connect__foot {
  position: absolute;
  bottom: 30px;
  display: flex;
  align-items: center;
  gap: 5px;
  color: #4e594b;
  font-size: 9.5px;
}

.connect {
  position: absolute;
  inset: 0;
  z-index: 30;
  overflow: auto;
  padding: 36px 16px;
  min-height: 100%;
}
.connect__center {
  max-width: 480px;
  margin: auto;
}
.connect__name {
  margin-top: 12px;
  color: #bdc7b8;
  font-size: 14px;
  overflow-wrap: anywhere;
}
.connect__host {
  flex-wrap: wrap;
  justify-content: center;
  overflow-wrap: anywhere;
  color: #a0ab9b;
}
.connect__step-detail {
  color: #899581;
  text-align: right;
}
.connect__step.is-current .connect__step-detail {
  color: #bdc7b8;
}
.connect__step.is-failed,
.connect__error {
  color: #ffaaa3;
}
.connect__error {
  width: 100%;
  margin-top: 16px;
  font-size: 12px;
  line-height: 1.7;
  overflow-wrap: anywhere;
}
.connect__actions {
  display: flex;
  gap: 10px;
  margin-top: 20px;
  justify-content: center;
}
.connect__actions button {
  min-height: 36px;
  padding: 8px 16px;
  border: 1px solid #ffffff26;
  border-radius: 9px;
  color: #c7d0c1;
  font-size: 12px;
}
.connect__actions .connect__primary {
  background: var(--accent);
  border-color: var(--accent);
  color: var(--accent-ink);
  font-weight: 600;
}
.connect__close {
  z-index: 2;
  top: 12px;
  right: 12px;
}
.connect button:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 3px;
}
@media (max-width: 520px), (max-height: 650px) {
  .connect {
    padding: 24px 12px;
  }
  .connect__center {
    padding: 12px;
  }
  .connect__pulse {
    width: 72px;
    height: 72px;
    margin-bottom: 18px;
  }
  .connect__core {
    width: 54px;
    height: 54px;
    border-radius: 18px;
  }
  .connect h2 {
    font-size: 20px;
  }
  .connect__steps {
    margin-top: 24px;
  }
  .connect__step-copy {
    flex-wrap: wrap;
    gap: 2px;
  }
  .connect__step-detail {
    font-size: 10px;
  }
  .connect__step {
    min-height: 48px;
    height: auto;
    padding: 8px 0;
  }
}
.connect .connect__close {
  background: #121911;
  border-color: #ffffff26;
  color: #aab6a3;
}
.connect .connect__close:hover {
  background: #20291c;
  color: #fff;
}
.connect .connect__actions button {
  background: #141b11;
  color: #c7d0c1;
}
.connect .connect__actions button:hover {
  background: #20291c;
  color: #fff;
}
.connect .connect__actions button.connect__primary {
  background: var(--accent);
  color: var(--accent-ink);
  border-color: var(--accent);
}
</style>
