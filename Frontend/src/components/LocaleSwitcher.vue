<!--
  - Language switcher for Araneae.
  - Provides a compact EN / 中 toggle that persists the choice in localStorage.
  -->
<template>
  <el-select
    :model-value="locale"
    size="small"
    style="width: 110px"
    @update:model-value="onChange"
  >
    <el-option
      v-for="item in availableLocales"
      :key="item.code"
      :label="item.label"
      :value="item.code"
    />
  </el-select>
</template>

<script lang="ts" setup>

const { locale, setLocale, availableLocales } = useI18n();

function onChange(value) {
  setLocale(value);
}
</script>
