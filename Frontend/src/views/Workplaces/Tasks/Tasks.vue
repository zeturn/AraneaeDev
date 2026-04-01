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
	const tabs = [
		{name: '程序任务', url: `/aprons/workplaces/${id}/tasks`},
		{name: '创建任务', url: `/aprons/workplaces/${id}/tasks/create`},
	];
	if (route.params.taskId) {
		tabs.push({name: '任务设置', url: `/aprons/workplaces/${id}/tasks/${route.params.taskId}/settings`});
		tabs.push({name: '运行记录', url: `/aprons/workplaces/${id}/tasks/${route.params.taskId}/runs`});
	}
	return tabs;
});
</script>


<template>
	<section class="w-full min-w-0 overflow-x-hidden px-4">
		<h1 class="text-gray-500 text-3xl py-2">
			任务
		</h1>
		<p class="text-gray-500 text-sm py-2">
			任务是一次性的实时自动化操作，如果需要定时或链式操作，请使用计划。
		</p>
		<Tabs :tabs="tabsList" class="py-2"/>
		<slot>
		</slot>
	</section>
</template>