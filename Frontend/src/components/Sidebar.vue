<!--
  - Copyright (c)   2024.8  Henry Zhao. All rights reserved.
  -->

<!-- Sidebar.vue -->
<template>
  <div :class="sidebarClasses">
    <div class="p-4">
      <div :class="smallSidebarClasses" class="flex items-center">
        <button class="btn-muted mr-4 px-2 py-1" @click="$emit('toggleSidebar')">
          ☰
        </button>
        <h1 class="text-lg text-blue-600">Araneae</h1>
      </div>
      <ul>
        <li v-for="(link, index) in links" :key="index" :class="menuItemClasses">
          <router-link :to="link.url">{{ link.name }}</router-link>
        </li>
      </ul>
    </div>
  </div>
</template>

<script lang="ts" setup>
import {computed} from 'vue';

const props = defineProps<{
  links: { name: string; url: string }[];
  isSidebarCollapsed: boolean;
  isLargeScreen: boolean;
}>();

const sidebarClasses = computed(() => {
  return [
    'fixed md:relative inset-y-0 left-0 transform transition-transform duration-200 ease-in-out bg-white',
    !props.isSidebarCollapsed || props.isLargeScreen ? 'translate-x-0' : '-translate-x-full',
    'text-white h-full',
    props.isLargeScreen ? (props.isSidebarCollapsed ? 'w-24' : 'w-64') : 'w-64'
  ];
});

const menuItemClasses = computed(() => {
  return [
    props.isSidebarCollapsed && props.isLargeScreen ? 'p-2 hover:bg-gray-200 text-xs' : 'p-2 hover:bg-gray-200',
    'text-blue-700 rounded'
  ];
});

const smallSidebarClasses = computed(() => {
  return [
    'flex items-center',
    props.isLargeScreen ? 'hidden' : 'justify-start'
  ];
});
</script>

<style scoped>
/* Additional styles */
</style>
