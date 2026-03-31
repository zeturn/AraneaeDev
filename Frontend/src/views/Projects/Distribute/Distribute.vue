<!--
  - Copyright (c)  2025.3.23
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Distribute.vue
  - Last Modified: 2025-03-23 23:46:27  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<script lang="ts" setup>
import {computed} from 'vue';
import {useRoute} from 'vue-router';
import Tabs from '@/components/Tabs.vue';

const tabsList = computed(() => {
	const route = useRoute();
	const id = route.params.id || 'default-id';  // 如果没有ID，使用默认值
	return [
		{name: '项目分发状态', url: `/aprons/projects/${id}/distribute`},
		{name: '创建项目分发', url: `/aprons/projects/${id}/distribute/order`},
	];
});
</script>


<template>
	<h1 class="text-gray-500 text-3xl m-4">
		项目分发
	</h1>
	<p class="text-gray-400 text-sm m-4">
		分发您的项目到各个节点
	</p>
	<Tabs :tabs="tabsList"/>
	<slot>
	</slot>
</template>