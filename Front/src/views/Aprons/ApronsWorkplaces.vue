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

const workplaces = ref([]);

const fetchWorkplaces = async () => {
  try {
	  const response = await ApiService.getMyWorkplaces();
	  workplaces.value = response.data.results;
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
		<div class="flex items-center mb-6">
			<h1 class="text-gray-500 text-3xl m-2">工作区</h1>
			<RouterLink
				v-if="workplaces.length"
				class="ml-auto rounded text-green-600 hover:bg-gray-200 p-2"
				to="/aprons/workplaces/create"
			>
				创建工作区
			</RouterLink>
		</div>
		<div v-if="workplaces.length" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
			<div
				v-for="workplace in workplaces"
				:key="workplace.id"
				class="p-6  rounded-lg bg-[#F9FAFB] transition-all hover:bg-gray-200"
			>
				<RouterLink :to="`/aprons/workplaces/${workplace.id}`" class="block">
					<h2 class="text-2xl font-semibold text-gray-900 mb-1">{{ workplace.name }}</h2>
					<span
						class="inline-block mx-1 px-3 py-1 text-xs font-mono font-semibold text-blue-600 bg-blue-100 rounded-lg">
					  ID: {{ workplace.id }}
					</span>
					<p class="inline-block mx-1 px-3 py-1 text-xs font-mono font-semibold text-green-600 bg-green-100 rounded-lg">
						{{ workplace.status }}
					</p>
					<h3 class="text-gray-500 mb-2 m-2">{{ workplace.description }}</h3>
				</RouterLink>
			</div>
		</div>
		<div v-else class="flex flex-col items-center justify-center h-full">
			<p class="text-gray-500 text-lg">还没有工作区</p>
			<RouterLink
				class="mt-4 rounded text-green-600 hover:bg-gray-200 p-2"
				to="/aprons/workplaces/create"
			>
				创建工作区↗
			</RouterLink>
		</div>
	</Aprons>
</template>

<style scoped>
</style>