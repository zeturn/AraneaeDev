<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - ApronsProjects.vue
  - Last Modified: 2025-05-19 21:17:11  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<script lang="ts" setup>
import {ref, onMounted} from 'vue';
import ApiService from '@/services/ApiService';
import Aprons from "@/views/Aprons/Aprons.vue";

// 定义项目数据的类型接口
interface Project {
  id: number; // 假设每个项目都有一个唯一的ID
  name: string;
  description: string | null;
  language: string;
  command: string;
  mode: string;
  created_at: string;
  updated_at: string;
}

// 定义项目列表和加载状态
const projects = ref<Project[]>([]);
const isLoading = ref(false);
const errorMessage = ref<string | null>(null);

// 获取项目的函数
const fetchProjects = async () => {
  isLoading.value = true;
  errorMessage.value = null;
  try {
	  const response = await ApiService.getMyProjects(); // 假设这个方法返回项目数据
    projects.value = response.data; // 将响应的数据赋值给projects
  } catch (error: any) {
    errorMessage.value = error.message || '无法加载项目';
  } finally {
    isLoading.value = false;
  }
};

// 组件挂载时调用fetchProjects
onMounted(() => {
  fetchProjects();
});
</script>

<template>
  <Aprons>
    <div class="flex flex-row flex-1 mb-4">
      <h1 class="text-3xl font-semibold text-gray-500">项目</h1>
    </div>
    <div v-if="isLoading" class="text-center py-4">
      <div class="animate-spin rounded-full h-10 w-10 border-t-2 border-b-2 border-gray-800"></div>
      <p class="mt-2 text-gray-500">正在加载项目...</p>
    </div>

    <div v-else-if="errorMessage" class="text-center text-red-500 text-lg font-medium">
      {{ errorMessage }}
    </div>

    <div v-else>
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <div v-for="project in projects" :key="project.id"
             class="p-6 bg-white rounded-lg shadow-md border border-gray-200">
          <RouterLink :to="`/aprons/projects/${project.id}`">
            <h2 class="text-xl font-semibold text-gray-700">{{ project.name }}</h2>
            <p class="text-sm text-gray-600 mt-2">描述: {{ project.description || '无描述' }}</p>
            <div class="flex flex-wrap items-center gap-4 mt-4">
              <span class="px-3 py-1 bg-blue-100 text-blue-800 text-sm font-medium rounded-full">语言: {{
                  project.language
                }}</span>
              <span class="px-3 py-1 bg-green-100 text-green-800 text-sm font-medium rounded-full">模式: {{
                  project.mode
                }}</span>
            </div>
            <p class="text-sm text-gray-600 mt-2">命令: <code
                class="bg-gray-100 text-gray-800 px-2 py-1 rounded">{{ project.command }}</code></p>
            <div class="mt-4 text-sm text-gray-500">
              <p>创建于: {{ project.created_at }}</p>
              <p>更新于: {{ project.updated_at }}</p>
            </div>

          </RouterLink>
        </div>
      </div>
    </div>
  </Aprons>
</template>

<style scoped>
/* 你可以在这里添加额外的样式 */
</style>
