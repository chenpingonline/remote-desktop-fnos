<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  ref,
  useAttrs,
  useId,
  watch,
} from "vue";
import { Check, ChevronDown } from "lucide-vue-next";
defineOptions({ inheritAttrs: false });
interface Option {
  value: string;
  label: string;
  disabled?: boolean;
}
const props = defineProps<{
  modelValue: string | number;
  options: Option[];
  disabled?: boolean;
}>();
const emit = defineEmits<{
  "update:modelValue": [value: string];
  change: [value: string];
}>();
const attrs = useAttrs(),
  id = useId(),
  trigger = ref<HTMLButtonElement>(),
  menu = ref<HTMLDivElement>(),
  opened = ref(false),
  active = ref(-1),
  position = ref<Record<string, string>>({});
const supportsPopover =
  typeof HTMLElement !== "undefined" && "showPopover" in HTMLElement.prototype;
const forwarded = computed(() =>
  Object.fromEntries(
    Object.entries(attrs).filter(([key]) => key !== "class" && key !== "style"),
  ),
);
const current = computed(() =>
  props.options.find((o) => o.value === String(props.modelValue)),
);
let search = "",
  searchTimer: ReturnType<typeof setTimeout> | undefined;
function place() {
  const r = trigger.value?.getBoundingClientRect();
  if (!r) return;
  const width = Math.min(Math.max(r.width, 160), window.innerWidth - 16),
    height = Math.min(props.options.length * 34 + 8, 260),
    below = window.innerHeight - r.bottom - 8,
    above = r.top - 8,
    up = below < Math.min(height, 140) && above > below,
    maxHeight = Math.max(40, Math.min(260, up ? above : below));
  position.value = {
    left: Math.max(8, Math.min(r.left, window.innerWidth - width - 8)) + "px",
    top:
      (up
        ? Math.max(8, r.top - Math.min(height, maxHeight) - 4)
        : r.bottom + 4) + "px",
    width: width + "px",
    maxHeight: maxHeight + "px",
  };
}
function removeListeners() {
  window.removeEventListener("resize", close);
  document.removeEventListener("scroll", onScroll, true);
  document.removeEventListener("pointerdown", outside, true);
}
function close() {
  opened.value = false;
  if (supportsPopover && menu.value?.matches(":popover-open"))
    menu.value.hidePopover();
  removeListeners();
}
function outside(e: PointerEvent) {
  const target = e.target as Node;
  if (!trigger.value?.contains(target) && !menu.value?.contains(target))
    close();
}
function onScroll(e: Event) {
  if (!menu.value?.contains(e.target as Node)) close();
}
function reveal() {
  nextTick(() =>
    menu.value
      ?.querySelector<HTMLElement>("[data-active=true]")
      ?.scrollIntoView({ block: "nearest" }),
  );
}
async function open() {
  if (props.disabled || !props.options.length) return;
  active.value = props.options.findIndex(
    (o) => o.value === String(props.modelValue) && !o.disabled,
  );
  if (active.value < 0)
    active.value = props.options.findIndex((o) => !o.disabled);
  place();
  opened.value = true;
  await nextTick();
  if (supportsPopover) menu.value?.showPopover();
  window.addEventListener("resize", close);
  document.addEventListener("scroll", onScroll, true);
  document.addEventListener("pointerdown", outside, true);
  reveal();
}
function toggle() {
  if (opened.value) close();
  else open();
}
function choose(index: number) {
  const option = props.options[index];
  if (!option || option.disabled) return;
  if (option.value !== String(props.modelValue)) {
    emit("update:modelValue", option.value);
    emit("change", option.value);
  }
  close();
  trigger.value?.focus();
}
function move(step: number) {
  for (let n = 1; n <= props.options.length; n++) {
    const index =
      (active.value + step * n + props.options.length * 2) %
      props.options.length;
    if (!props.options[index].disabled) {
      active.value = index;
      reveal();
      break;
    }
  }
}
function keydown(e: KeyboardEvent) {
  if (props.disabled) return;
  if (["ArrowDown", "ArrowUp", "Home", "End", "Enter", " "].includes(e.key)) {
    e.preventDefault();
    if (!opened.value) {
      open();
      return;
    }
    if (e.key === "ArrowDown" || e.key === "ArrowUp")
      move(e.key === "ArrowDown" ? 1 : -1);
    else if (e.key === "Home" || e.key === "End") {
      active.value = e.key === "Home" ? -1 : 0;
      move(e.key === "Home" ? 1 : -1);
    } else choose(active.value);
  } else if (e.key === "Escape" && opened.value) {
    e.preventDefault();
    e.stopPropagation();
    close();
  } else if (e.key === "Tab") close();
  else if (
    opened.value &&
    e.key.length === 1 &&
    !e.ctrlKey &&
    !e.metaKey &&
    !e.isComposing
  ) {
    search += e.key.toLocaleLowerCase();
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => (search = ""), 650);
    const index = props.options.findIndex(
      (o) => !o.disabled && o.label.toLocaleLowerCase().startsWith(search),
    );
    if (index >= 0) {
      active.value = index;
      reveal();
    }
  }
}
function onToggle(e: Event) {
  if ((e as ToggleEvent).newState === "closed") {
    opened.value = false;
    removeListeners();
  }
}
watch(
  () => props.disabled,
  (v) => {
    if (v) close();
  },
);
onBeforeUnmount(() => {
  close();
  clearTimeout(searchTimer);
});
</script>
<template>
  <span class="app-select" :class="attrs.class" :style="attrs.style as any">
    <button
      ref="trigger"
      v-bind="forwarded"
      type="button"
      role="combobox"
      class="select-trigger"
      :disabled="disabled"
      aria-haspopup="listbox"
      :aria-expanded="opened"
      :aria-controls="id + '-list'"
      :aria-activedescendant="
        opened && active >= 0 ? id + '-option-' + active : undefined
      "
      @click.stop="toggle"
      @keydown="keydown"
    >
      <span>{{ current?.label || "请选择" }}</span
      ><ChevronDown :size="15" :class="{ rotated: opened }" />
    </button>
    <div
      ref="menu"
      :id="id + '-list'"
      role="listbox"
      :aria-label="(attrs['aria-label'] as string) || '选项'"
      :popover="supportsPopover ? 'auto' : undefined"
      class="select-menu"
      :class="{ fallback: !supportsPopover, 'is-open': opened }"
      :style="position"
      @toggle="onToggle"
    >
      <div
        v-for="(option, index) in options"
        :id="id + '-option-' + index"
        :key="option.value"
        role="option"
        class="select-option"
        :data-active="index === active"
        :aria-selected="option.value === String(modelValue)"
        :aria-disabled="option.disabled || undefined"
        :title="option.label"
        @pointerenter="active = index"
        @pointerdown.prevent
        @click.stop.prevent="choose(index)"
      >
        <span>{{ option.label }}</span
        ><Check v-if="option.value === String(modelValue)" :size="15" />
      </div>
    </div>
  </span>
</template>
<style>
.app-select {
  display: inline-flex;
  min-width: 0;
  vertical-align: middle;
}
.select-trigger {
  width: 100%;
  justify-content: space-between;
  gap: 12px;
  padding: 7px 10px;
  border-radius: 6px;
  font: inherit;
  line-height: 20px;
  text-align: left;
}
.select-trigger > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.select-trigger > svg {
  flex-shrink: 0;
  color: var(--muted);
  transition: transform 0.12s;
}
.select-trigger > svg.rotated {
  transform: rotate(180deg);
}
.select-menu {
  display: none;
  position: fixed;
  inset: auto;
  margin: 0;
  padding: 4px;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-width: thin;
  box-sizing: border-box;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--canvas);
  color: var(--text);
  box-shadow: 0 8px 28px #0003;
  font:
    13px/20px -apple-system,
    BlinkMacSystemFont,
    "PingFang SC",
    "Microsoft YaHei",
    sans-serif;
  z-index: 1000;
}
.select-menu:popover-open,
.select-menu.fallback.is-open {
  display: block;
}
.select-menu::backdrop {
  background: transparent;
}
.select-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 34px;
  padding: 7px 9px;
  border-radius: 4px;
  cursor: pointer;
}
.select-option > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.select-option > svg {
  flex-shrink: 0;
  color: var(--blue);
}
.select-option[data-active="true"] {
  background: var(--surface);
}
.select-option[aria-selected="true"] {
  color: var(--blue);
}
.select-option[aria-disabled="true"] {
  opacity: 0.45;
  cursor: default;
}
.dialog-body label > .app-select {
  width: 100%;
}
.setting-row > .app-select {
  min-width: 140px;
  flex-shrink: 0;
}
.desktop-tools .app-select {
  max-width: 130px;
  font-size: 11px;
}
.desktop-tools .select-trigger {
  padding: 5px 8px;
}
.app-select.mobile-group {
  display: none;
}
@media (max-width: 600px) {
  .app-select.mobile-group {
    display: inline-flex;
  }
}
</style>
