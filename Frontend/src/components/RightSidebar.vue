<!--
  - Copyright (c)   2025.2  Henry Zhao. All rights reserved.
  - From CA.
  -->

<!-- RightSidebar.vue -->
<template>
  <div :class="rightSidebarClasses">
    <div class="p-4">
      <div :class="smallRightSidebarClasses" class="flex items-center">
        <SmallAvatar/>
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
import SmallAvatar from "@/components/SmallAvatar.vue";


const props = defineProps<{
  links: { name: string; url: string }[];
  isRightSidebarCollapsed: boolean;
  isLargeScreen: boolean;
}>();

const rightSidebarClasses = computed(() => {
  return [
    'fixed md:relative inset-y-0 right-0 transform transition-transform duration-200 ease-in-out bg-white',
    !props.isRightSidebarCollapsed || props.isLargeScreen ? 'translate-x-0' : 'translate-x-full',
    'text-white h-full',
    props.isLargeScreen ? (props.isRightSidebarCollapsed ? 'w-24' : 'w-64') : 'w-64'
  ];
});

const menuItemClasses = computed(() => {
  return [
    props.isRightSidebarCollapsed && props.isLargeScreen ? 'p-2 hover:bg-gray-200 text-xs' : 'p-2 hover:bg-gray-200',
    'text-blue-700 rounded'
  ];
});

const smallRightSidebarClasses = computed(() => {
  return [
    'flex items-center',
    props.isLargeScreen ? 'hidden' : 'justify-end'
  ];
});


</script>

<style scoped>
/* Additional styles */
</style>

