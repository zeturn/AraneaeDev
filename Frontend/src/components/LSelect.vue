<template>
  <div ref="rootRef" class="l-select">
    <button
      type="button"
      class="btn-ghost btn-ghost-muted l-select-trigger"
      :class="{ 'is-open': isOpen }"
      :aria-expanded="isOpen"
      aria-haspopup="listbox"
      @click="toggleOpen"
      @keydown.enter.prevent="toggleOpen"
      @keydown.space.prevent="toggleOpen"
      @keydown.esc.prevent="close"
    >
      <span class="l-select-label" :class="{ 'is-placeholder': !selectedOption }">
        {{ selectedOption ? selectedOption.label : placeholder }}
      </span>
      <svg
        class="l-select-arrow"
        :class="{ 'is-open': isOpen }"
        viewBox="0 0 20 20"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path d="M6 8L10 12L14 8" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    </button>

    <transition name="l-select-pop">
      <ul v-if="isOpen" class="l-select-menu" role="listbox">
        <li v-for="option in normalizedOptions" :key="String(option.value)">
          <button
            type="button"
            class="btn-ghost btn-ghost-muted l-select-option"
            :class="{
              'is-selected': isSelected(option.value),
              'is-disabled': option.disabled,
            }"
            :disabled="option.disabled"
            @click="selectOption(option)"
          >
            {{ option.label }}
          </button>
        </li>
      </ul>
    </transition>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';

const props = defineProps({
  modelValue: {
    type: [String, Number, null],
    default: '',
  },
  options: {
    type: Array,
    default: () => [],
  },
  placeholder: {
    type: String,
    default: '请选择',
  },
  disabled: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(['update:modelValue', 'change']);

const rootRef = ref(null);
const isOpen = ref(false);

const normalizedOptions = computed(() => {
  return (Array.isArray(props.options) ? props.options : []).map((item) => {
    if (item && typeof item === 'object') {
      return {
        label: item.label ?? String(item.value ?? ''),
        value: item.value ?? '',
        disabled: Boolean(item.disabled),
      };
    }
    return {
      label: String(item ?? ''),
      value: item ?? '',
      disabled: false,
    };
  });
});

const selectedOption = computed(() => {
  return normalizedOptions.value.find((option) => isSelected(option.value));
});

const isSelected = (value) => {
  return String(props.modelValue ?? '') === String(value ?? '');
};

const close = () => {
  isOpen.value = false;
};

const toggleOpen = () => {
  if (props.disabled) {
    return;
  }
  isOpen.value = !isOpen.value;
};

const selectOption = (option) => {
  if (option.disabled) {
    return;
  }
  emit('update:modelValue', option.value);
  emit('change', option.value);
  close();
};

const onDocumentClick = (event) => {
  if (!rootRef.value) {
    return;
  }
  if (!rootRef.value.contains(event.target)) {
    close();
  }
};

onMounted(() => {
  document.addEventListener('click', onDocumentClick);
});

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocumentClick);
});
</script>

<style scoped>
.l-select {
  position: relative;
  width: 100%;
}

.l-select-trigger {
  width: 100%;
  min-height: 46px;
  border: none;
  border-radius: 12px;
  background: #f3f4f6;
  color: #1f2a37;
  display: inline-flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem 0.95rem;
  font-size: 0.875rem;
  line-height: 1.25;
  transition: background-color 0.2s ease, color 0.2s ease;
}

.l-select-trigger:hover,
.l-select-trigger.is-open {
  background: #eef2f7;
}

.l-select-trigger:focus-visible {
  outline: 2px solid #2dd4bf;
  outline-offset: 1px;
}

.l-select-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.l-select-label.is-placeholder {
  color: #64748b;
}

.l-select-arrow {
  width: 1rem;
  height: 1rem;
  color: #64748b;
  flex-shrink: 0;
  transition: transform 0.2s ease, color 0.2s ease;
}

.l-select-arrow.is-open {
  transform: rotate(180deg);
  color: #475569;
}

.l-select-menu {
  position: absolute;
  top: calc(100% + 0.45rem);
  left: 0;
  right: 0;
  border: none;
  border-radius: 14px;
  background: #f8fafc;
  box-shadow: none;
  padding: 0.45rem;
  margin: 0;
  list-style: none;
  z-index: 40;
}

.l-select-option {
  display: block;
  width: 100%;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: #334155;
  text-align: left;
  font-size: 0.875rem;
  line-height: 1.3;
  padding: 0.62rem 0.78rem;
  transition: background-color 0.2s ease, color 0.2s ease;
}

.l-select-option:hover {
  background: rgba(20, 184, 166, 0.14);
  color: #0f766e;
}

.l-select-option.is-selected {
  background: rgba(20, 184, 166, 0.2);
  color: #0f766e;
}

.l-select-option.is-disabled {
  cursor: not-allowed;
  color: #64748b;
}

.l-select-pop-enter-active,
.l-select-pop-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
  transform-origin: top center;
}

.l-select-pop-enter-from,
.l-select-pop-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.98);
}
</style>