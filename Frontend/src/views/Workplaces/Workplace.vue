<!--
  - Copyright (c)   2025.2  Henry Zhao. All rights reserved.
  - From CA.
  -->

<!-- App.vue -->
<template>
  <div class="flex min-h-screen w-full flex-col overflow-x-hidden">
    <Header @toggleSidebar="toggleSidebar"/>
    <div class="flex flex-row flex-1 overflow-x-hidden">
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


<script lang="ts" setup>
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
		{name: '返回', url: `/aprons/workplaces`},
		{name: '概览', url: `/aprons/workplaces/${id}`},
    {name: '项目', url: `/aprons/workplaces/${id}/projects`},
    {name: '任务', url: `/aprons/workplaces/${id}/tasks`},
    {name: '计划', url: `/aprons/workplaces/${id}/schedules`},
    {name: '分析和日志', url: `/aprons/workplaces/${id}/AnalyticsAndLogging`},
    {name: '设置', url: `/aprons/workplaces/${id}/settings`},
	];
});

const right_links = computed(() => {
  let route = useRoute();
  const id = route.params.id || 'default-id';  // 如果没有ID，使用默认值
  return [
    {name: '概览', url: `/aprons/workplaces`},
    {name: '个人资料', url: `/aprons/profile`},
    {name: '注销', url: `/logout`},

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
