<!--
  - Copyright (c)   2025.2  Henry Zhao. All rights reserved.
  - From CA.
  -->

<script lang="ts" setup>
import {computed} from 'vue';
import {useRoute} from 'vue-router';
import Tabs from '@/components/Tabs.vue';

const tabsList = computed(() => {
	const route = useRoute();
	const id = route.params.id || 'default-id';  // 如果没有ID，使用默认值
	const tabs = [
		{name: '程序项目', url: `/aprons/projects/${id}/repo`},
		{name: '创建项目', url: `/aprons/projects/${id}/repo/create`},
	];
	if (route.params.versionId) {
		tabs.push({name: '版本设置', url: `/aprons/projects/${id}/versions/${route.params.versionId}/settings`});
	}
	return tabs;
});
</script>


<template>
	<h1 class="text-gray-500 text-3xl m-4">
		项目仓库
	</h1>
	<Tabs :tabs="tabsList"/>
	<slot>
	</slot>
</template>