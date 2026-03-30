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
	  const response = await ApiService.getMyProjects();
	  const payload = response?.data;
    projects.value = Array.isArray(payload)
		? payload
		: (Array.isArray(payload?.results) ? payload.results : []);
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
    <div class="mb-4 flex flex-row items-center gap-3">
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
             class="surface-card">
          <RouterLink :to="`/aprons/projects/${project.id}`">
            <h2 class="text-xl font-semibold text-gray-700">{{ project.name }}</h2>
            <p class="text-sm text-gray-600 mt-2">描述: {{ project.description || '无描述' }}</p>
            <div class="flex flex-wrap items-center gap-4 mt-4">
              <span class="tag-pill">语言: {{
                  project.language
                }}</span>
              <span class="tag-pill">模式: {{
                  project.mode
                }}</span>
            </div>
            <p class="text-sm text-gray-600 mt-2">命令: <code
                class="rounded bg-gray-100 px-2 py-1 text-gray-800">{{ project.command }}</code></p>
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
