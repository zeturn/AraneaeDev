<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Tabs.vue
  - Last Modified: 2025-05-19 17:41:19  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
  <div class="overflow-x-auto whitespace-nowrap">
    <div class="flex">
      <router-link
          v-for="link in links"
          :key="link.name"
          :class="{
          'bg-gray-100 rounded-lg': link.url === $route.path,
          'text-gray-500': link.url !== $route.path,
        }"
          :to="link.url"
          class="px-4 py-2 inline-block text-center cursor-pointer"
      >
        {{ link.name }}
      </router-link>
    </div>
  </div>

  <div class="mt-4">
    <!-- Tab Content Goes Here -->
    <router-view></router-view>
  </div>
</template>

<script setup>
import {computed} from 'vue';
import {useRoute} from 'vue-router';
import {defineProps} from 'vue';

const props = defineProps({
  tabs: {
    type: Array,
    required: true
  }
});

const route = useRoute();
const links = computed(() => {
  return props.tabs.map(link => {
    // 使用当前路由参数动态生成URL
    const id = route.params.id || 'default-id';
    return {
      ...link,
      url: link.url.replace(':id', id) // 替换URL中的占位符 :id
    };
  });
});
</script>
