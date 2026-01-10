<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - ApronsTeams.vue
  - Last Modified: 2025-05-22 20:50:58  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<script lang="ts" setup>
import {ref, onMounted} from 'vue';
import ApiService from '@/services/ApiService.js';
import Aprons from "@/views/Aprons/Aprons.vue";

const teams = ref([]);

const fetchTeams = async () => {
  try {
	  const response = await ApiService.getMyTeams();
	  teams.value = response.data.results;
  } catch (error) {
    console.error('Error fetching workplaces:', error);
  }
};

onMounted(() => {
  fetchTeams();
});
</script>

<template>
	<Aprons>
		<div class="flex items-center mb-6">
			<h1 class="text-gray-500 text-3xl m-2">团队</h1>
			<RouterLink
				v-if="teams.length"
				class="ml-auto rounded text-green-600 hover:bg-gray-200 p-2"
				to="/aprons/teams/create">创建团队
			</RouterLink>
		</div>
		<div v-if="teams.length" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
			<div
				v-for="team in teams"
				:key="team.id"
				class="p-6 rounded-lg bg-[#F9FAFB] transition-all hover:bg-gray-200"
			>
				<RouterLink
					:to="`/aprons/teams/${team.id}`"
					class="flex flex-col h-full"
				>
					<div class="flex items-center justify-between">
						<h2 class="text-xl font-bold text-gray-800">{{ team.name }}</h2>
						<span
							class="inline-block px-3 py-1 text-xs font-mono font-semibold text-yellow-600 bg-yellow-300 rounded-lg"
						>
              {{ team.role }}
            </span>
					</div>
					<p class="mt-2 text-gray-600 flex-1">
						{{ team.description }}
					</p>
					<div class="mt-4 flex items-center justify-between text-sm text-gray-500">
						<span>ID: {{ team.id }}</span>
					</div>
				</RouterLink>
			</div>
		</div>
		<div v-else class="flex flex-col items-center justify-center h-full">
			<p class="text-gray-500 text-lg">还没有团队</p>
			<RouterLink
				class="mt-4 rounded text-green-600 hover:bg-gray-200 p-2"
				to="/aprons/teams/create"
			>
				创建团队↗
			</RouterLink>
		</div>
	</Aprons>
</template>


<style scoped>

</style>