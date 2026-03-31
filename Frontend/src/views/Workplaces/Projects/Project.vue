<!--
  - Copyright (c)   2024.8  Henry Zhao. All rights reserved.
  -->

<script lang="ts" setup>
import {computed} from 'vue';
import {useRoute} from 'vue-router';
import Tabs from '@/components/Tabs.vue';

const tabsList = computed(() => {
  const route = useRoute();
  const id = route.params.id || 'default-id';  // 如果没有ID，使用默认值
  return [
    {name: '程序项目', url: `/aprons/workplaces/${id}/projects`},
    {name: '创建项目', url: `/aprons/workplaces/${id}/projects/create`},
  ];
});
</script>


<template>
  <h1 class="text-gray-500 text-3xl m-4">
    项目
  </h1>
	<Tabs :tabs="tabsList" class="m-4"/>
  <slot>
  </slot>
</template>