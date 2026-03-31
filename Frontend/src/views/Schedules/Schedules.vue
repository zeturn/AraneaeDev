<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Schedules.vue
  - Last Modified: 2025-05-19 21:08:52  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<!-- App.vue -->
<template>
	<div class="flex flex-col h-screen w-screen">
		<Header @toggleSidebar="toggleSidebar"/>
		<div class="flex flex-row flex-1">
			<Sidebar :isLargeScreen="isLargeScreen" :isSidebarCollapsed="!isSidebarOpen" :links="links"
			         @toggleSidebar="toggleSidebar"/>
			<main class="flex-1 p-4">
				<slot></slot>
			</main>
			<RightSidebar :isLargeScreen="isLargeScreen" :isRightSidebarCollapsed="!isSidebarOpen" :links="right_links"
			              @toggleRightSidebar="toggleSidebar"/>
		</div>
	</div>
</template>


<script lang="ts" setup>
import {computed, defineComponent, onBeforeUnmount, onMounted, ref} from 'vue';
import Header from '../../components/Header.vue';
import Sidebar from '../../components/Sidebar.vue';
import RightSidebar from '@/components/RightSidebar.vue';
import {useRoute} from 'vue-router';
import ApiService from '@/services/ApiService';

defineComponent({
	components: {
		Header,
		Sidebar,
	},
	created() {
		console.log(Header, Sidebar);
	}
});

const route = useRoute();
const scheduleId = String(route.params.id || '');

const schedule = ref<Record<string, any>>({});
// === 以下功能：响应式工作区ID定义 ===
// Define reactive workplace ID
const workplaceId = ref<number>(0);

const loading = ref(false);
const error = ref<string | null>(null);

const isSidebarOpen = ref(false);
const isLargeScreen = ref(window.innerWidth >= 768);

// === 以下功能：链接列表计算 ===
// 中文: 根据响应式 workplaceId 生成导航链接
// English: Generate nav links based on reactive workplaceId
const links = computed(() => {
	const id = route.params.id || 'default-id';
	return [
		{name: '返回', url: `/aprons/workplaces/${workplaceId.value}/schedules`},
		{name: '概览', url: `/aprons/schedule/${id}`},
		{name: '编辑', url: `/aprons/schedule/${id}/edit`},
		{name: '计划设置', url: `/aprons/schedule/${id}/settings`},
	];
});

const right_links = computed(() => {
	const id = route.params.id || 'default-id';
	return [
		{name: '概览', url: `/aprons/projects`},
		{name: '个人资料', url: `/profile`},
		{name: '注销', url: `/logout`},
	];
});

const toggleSidebar = () => {
	isSidebarOpen.value = !isSidebarOpen.value;
};

const checkScreenSize = () => {
	isLargeScreen.value = window.innerWidth >= 768;
	isSidebarOpen.value = isLargeScreen.value;
};

/**
 * 中文: 调用 API 获取调度并设置工作区ID
 * English: Fetch schedule and set workplace ID
 */
async function fetchSchedule() {
	loading.value = true;
	error.value = null;
	try {
		const response = await ApiService.getSchedule(scheduleId);
		schedule.value = response.data;
		// 设置响应式工作区ID
		workplaceId.value = schedule.value.workplace;
		console.log('调度详情，工作区ID：', workplaceId.value);
	} catch (err: any) {
		error.value = err.response?.data?.message || '获取调度失败';
	} finally {
		loading.value = false;
	}
}

onMounted(() => {
	checkScreenSize();
	window.addEventListener('resize', checkScreenSize);
	fetchSchedule();
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
