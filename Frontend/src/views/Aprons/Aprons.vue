<!--
  - Copyright (c)  2025.3.22
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Aprons.vue
  - Last Modified: 2025-03-22 00:17:26  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<!-- App.vue -->
<template>
	<div class="flex h-screen w-full flex-col overflow-x-hidden">
		<Header @toggleSidebar="toggleSidebar"/>
		<div class="flex flex-1 flex-row overflow-x-hidden">
			<Sidebar
				:isLargeScreen="isLargeScreen"
				:isSidebarCollapsed="!isSidebarOpen"
				:links="links"
				@toggleSidebar="toggleSidebar"
			/>
			<main class="min-w-0 flex-1 overflow-y-auto overflow-x-hidden p-4">
				<!-- Wrap slot in a max-width container -->
				<div class="min-w-0 max-w-full">
					<slot></slot>
				</div>
			</main>
		</div>
	</div>
</template>

<script lang="ts" setup>
import {ref, onMounted, onBeforeUnmount} from 'vue';
import Header from '../../components/Header.vue';
import Sidebar from '../../components/Sidebar.vue';

const isSidebarOpen = ref(false);
const isLargeScreen = ref(window.innerWidth >= 768);

const links = [
	{name: '工作区', url: '/aprons/workplaces'},
	{name: '节点', url: '/aprons/nodes'},
	{name: 'RSS', url: '/aprons/rss'},
	{name: '团队', url: '/aprons/teams'},
	{name: '设置', url: '/aprons/settings'},
	{name: '帮助', url: '/aprons/help'},
	{name: '关于', url: '/aprons/about'},
];

const toggleSidebar = () => {
	isSidebarOpen.value = !isSidebarOpen.value;
};

const checkScreenSize = () => {
	isLargeScreen.value = window.innerWidth >= 768;
	isSidebarOpen.value = isLargeScreen.value;
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
/* Ensure the browser content box never overflows horizontally */
html, body {
	margin: 0;
	padding: 0;
	width: 100%;
	overflow-x: hidden;
	box-sizing: border-box;
}

/* Optional: force all elements to respect their container’s width */
*, *::before, *::after {
	box-sizing: inherit;
}
</style>
