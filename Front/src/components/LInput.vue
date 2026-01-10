<!--
  - Copyright (c)   2024.11  Henry Zhao. All rights reserved.
  - From CA.
  -->

<template>
  <div class="flex flex-col mb-4">
    <label :for="inputId" class="mb-1 text-gray-700">{{ label }}</label>
    <div
        :style="{ borderColor: active ? activeBorderColor : borderColor }"
        class="relative border border-gray-300 rounded p-2 transition duration-300"
        @mouseenter="active = true"
        @mouseleave="active = false"
    >
      <input
          :id="inputId"
          :class="{'ring ring-offset-2': active}"
          :placeholder="placeholder"
          :style="{ borderColor: active ? activeBorderColor : borderColor }"
          :value="value"
          class="w-full outline-none"
          type="text"
          @blur="active = false"
          @focus="active = true"
          @input="$emit('input', $event.target.value)"
      />
    </div>
  </div>
</template>

<script>
export default {
  props: {
    label: {
      type: String,
      required: true
    },
    placeholder: {
      type: String,
      default: '请输入内容'
    },
    inputId: {
      type: String,
      default: 'input-box'
    },
    borderColor: {
      type: String,
      default: '#D1D5DB' // Tailwind's gray-300
    },
    activeBorderColor: {
      type: String,
      default: '#2563EB' // Tailwind's blue-600
    },
    value: {
      type: String,
      default: ''
    }
  },
  data() {
    return {
      active: false
    };
  }
};
</script>

<style scoped>
/* Optional: Add any additional styles here */
</style>
