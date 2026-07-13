<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Project.vue
  - Last Modified: 2025-05-24 14:58:05  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<!-- App.vue -->
<template>
	<div class="flex h-screen w-full flex-col overflow-x-hidden">
		<Header @toggleSidebar="toggleSidebar"/>
		<div class="flex flex-1 flex-row overflow-x-hidden">
			<Sidebar :isLargeScreen="isLargeScreen" :isSidebarCollapsed="!isSidebarOpen" :links="links"
			         @toggleSidebar="toggleSidebar"/>
			<main class="min-w-0 flex-1 overflow-x-hidden p-4">
				<slot></slot>
			</main>
			<RightSidebar :isLargeScreen="isLargeScreen" :isRightSidebarCollapsed="!isSidebarOpen" :links="right_links"
			              @toggleRightSidebar="toggleSidebar"/>
		</div>
	</div>
</template>


<script lang="ts" setup>import { useI18n } from '@/i18n';
const { t } = useI18n();

import {computed, defineComponent, onBeforeUnmount, onMounted, ref} from 'vue';
import Header from '../../components/Header.vue';
import Sidebar from '../../components/Sidebar.vue';
import RightSidebar from "@/components/RightSidebar.vue";
import {useRoute} from "vue-router";

defineComponent({
	components: {
		Header,
		Sidebar
	},
	created() {
		console.log(Header, Sidebar);
	}
});

const isSidebarOpen = ref(false); // 默认关闭
const isLargeScreen = ref(window.innerWidth >= 768);

// 定义链接列表
// 定义链接列表，并动态填入 ID
const links = computed(() => {
	let route = useRoute();
	const id = route.params.id || 'default-id';  // 如果没有ID，使用默认值
	return [
		{name: t('返回'), url: `/aprons/workplaces`},
		{name: t('概览'), url: `/aprons/projects/${id}`},
		{name: t('代码仓库'), url: `/aprons/projects/${id}/repo`},
		{name: t('项目分发'), url: `/aprons/projects/${id}/distribute`},
		{name: t('项目设置'), url: `/aprons/projects/${id}/settings`},
	];
});

const right_links = computed(() => {
	let route = useRoute();
	const id = route.params.id || 'default-id';  // 如果没有ID，使用默认值
	return [
		{name: t('概览'), url: `/aprons/projects`},
		{name: t('个人资料'), url: `/aprons/profile`},
		{name: t('注销'), url: `/logout`},
	];
});

//console.log('isSidebarOpen', isSidebarOpen);

const toggleSidebar = () => {
	console.log('toggleSidebar triggered');
	isSidebarOpen.value = !isSidebarOpen.value;
};

const checkScreenSize = () => {
	isLargeScreen.value = window.innerWidth >= 768;
	console.log('isLargeScreen', isLargeScreen);
	isSidebarOpen.value = !!isLargeScreen.value;
};

onMounted(() => {
	checkScreenSize();
	window.addEventListener('resize', checkScreenSize);
});

onBeforeUnmount(() => {
	window.removeEventListener('resize', checkScreenSize);
});
</script>

<style>
body {
	margin: 0;
}
</style>
