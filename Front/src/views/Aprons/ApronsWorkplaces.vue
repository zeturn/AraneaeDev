<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - ApronsWorkplaces.vue
  - Last Modified: 2025-05-22 20:47:41  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<script lang="ts" setup>
import {ref, onMounted} from 'vue';
import ApiService from '@/services/ApiService.js';
import Aprons from "@/views/Aprons/Aprons.vue";

interface Workplace {
  id: number;
  name: string;
  description: string;
  status: string;
}

const workplaces = ref<Workplace[]>([]);

const fetchWorkplaces = async () => {
  try {
	  const response = await ApiService.getMyWorkplaces();
	  const payload = response?.data;
	  workplaces.value = Array.isArray(payload)
		  ? payload
		  : (Array.isArray(payload?.results) ? payload.results : []);
  } catch (error) {
    console.error('Error fetching workplaces:', error);
  }
};

onMounted(() => {
  fetchWorkplaces();
});
</script>

<template>
	<Aprons>
		<div class="mb-6 flex items-center">
			<h1 class="m-2 text-3xl text-gray-500">工作区</h1>
			<RouterLink
				v-if="workplaces.length"
				class="btn-primary ml-auto"
				to="/aprons/workplaces/create"
			>
				创建工作区
			</RouterLink>
		</div>
		<div v-if="workplaces.length" class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
			<div
				v-for="workplace in workplaces"
				:key="workplace.id"
				class="rounded-lg bg-[#F9FAFB] p-6 transition-all hover:bg-gray-200"
			>
				<RouterLink :to="`/aprons/workplaces/${workplace.id}`" class="block">
					<h2 class="mb-2 text-2xl font-semibold text-gray-900">{{ workplace.name }}</h2>
					<div class="mb-2 flex flex-wrap gap-2">
						<span class="tag-pill">ID: {{ workplace.id }}</span>
						<span class="tag-pill">{{ workplace.status || 'active' }}</span>
					</div>
					<h3 class="m-1 text-gray-500">{{ workplace.description || '暂无描述' }}</h3>
				</RouterLink>
			</div>
		</div>
		<div v-else class="flex h-full flex-col items-center justify-center gap-3 py-8">
			<p class="text-gray-500 text-lg">还没有工作区</p>
			<RouterLink
				class="btn-primary"
				to="/aprons/workplaces/create"
			>
				创建工作区↗
			</RouterLink>
		</div>
	</Aprons>
</template>

<style scoped>
</style>
