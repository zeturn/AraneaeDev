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
		{name: '程序任务', url: `/aprons/workplaces/${id}/tasks`},
		{name: '创建任务', url: `/aprons/workplaces/${id}/tasks/create`},
	];
});
</script>


<template>
	<h1 class="text-gray-500 text-3xl m-4">
		任务
	</h1>
	<p class="text-gray-500 text-sm m-4">
		任务是一次性的实时自动化操作，如果需要定时或链式操作，请使用计划。
	</p>
	<Tabs :tabs="tabsList" class="m-4"/>
	<slot>
	</slot>
</template>