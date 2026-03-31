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

interface Team {
  id: number;
  name: string;
  description: string;
  role?: string | null;
}

const teams = ref<Team[]>([]);
const notice = ref('');

const fetchTeams = async () => {
  try {
	  const response = await ApiService.getMyTeams();
	  teams.value = Array.isArray(response?.data?.results) ? response.data.results : [];
	  notice.value = '';
  } catch (error) {
		console.error('Error fetching teams:', error);
		notice.value = '加载团队失败';
  }
};

onMounted(() => {
  fetchTeams();
});
</script>

<template>
	<Aprons>
		<div class="mb-6 flex items-center">
			<h1 class="m-2 text-3xl text-gray-500">团队</h1>
			<RouterLink
				v-if="teams.length"
				class="btn-primary ml-auto"
				to="/aprons/teams/create">创建团队
			</RouterLink>
		</div>
		<div class="mb-2 text-sm text-slate-500">{{ notice }}</div>
		<div v-if="teams.length" class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
			<div
				v-for="team in teams"
				:key="team.id"
				class="team-card transition-all hover:bg-[#e9eff6]"
			>
				<RouterLink :to="`/aprons/teams/${team.id}`" class="flex h-full flex-col">
					<div class="flex items-center justify-between">
						<h2 class="text-xl font-bold text-gray-800">{{ team.name || '未命名团队' }}</h2>
						<span
							class="tag-pill"
						>
              {{ team.role || 'member' }}
            </span>
					</div>
					<p class="mt-2 flex-1 text-gray-600">
						{{ team.description || '暂无描述' }}
					</p>
					<div class="mt-4 flex items-center justify-between text-sm text-gray-500">
						<span>ID: {{ team.id }}</span>
						<RouterLink class="btn-muted" :to="`/aprons/teams/${team.id}/settings`">设置</RouterLink>
					</div>
				</RouterLink>
			</div>
		</div>
		<div v-else class="flex h-full flex-col items-center justify-center">
			<p class="text-gray-500 text-lg">还没有团队</p>
			<RouterLink
				class="btn-primary mt-4"
				to="/aprons/teams/create"
			>
				创建团队↗
			</RouterLink>
		</div>
	</Aprons>
</template>


<style scoped>

</style>
