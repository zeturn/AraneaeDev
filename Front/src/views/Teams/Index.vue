<!--
  - Copyright (c)  2025.4.29
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-04-29 00:36:39  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Team>
		<div class="mx-auto max-w-5xl px-4 pb-10">
			<div class="team-panel space-y-6">
				<header class="space-y-2">
					<p class="text-xs uppercase tracking-wide text-slate-500">Team Overview</p>
					<h1 class="text-2xl font-semibold text-slate-900">{{ team.name || '团队概况' }}</h1>
					<p class="text-sm text-slate-500">{{ team.description || '暂无描述' }}</p>
				</header>

				<div v-if="loading" class="text-sm text-slate-500">加载中...</div>
				<div v-else class="grid gap-4 md:grid-cols-3">
					<div class="team-card">
						<p class="text-xs uppercase tracking-wide text-slate-500">成员数量</p>
						<p class="mt-2 text-2xl font-semibold text-slate-900">{{ members.length }}</p>
					</div>
					<div class="team-card">
						<p class="text-xs uppercase tracking-wide text-slate-500">我的角色</p>
						<p class="mt-2 text-2xl font-semibold text-slate-900">{{ team.role || 'member' }}</p>
					</div>
					<div class="team-card">
						<p class="text-xs uppercase tracking-wide text-slate-500">加入策略</p>
						<p class="mt-2 text-2xl font-semibold text-slate-900">{{ team.join_able ? '开放' : '受限' }}</p>
					</div>
				</div>

				<div class="team-card space-y-3">
					<div class="flex flex-wrap items-center justify-between gap-2">
						<h2 class="text-base font-semibold text-slate-900">成员预览</h2>
						<router-link class="btn-primary" :to="`/aprons/teams/${teamId}/members`">进入添加成员</router-link>
					</div>
					<div v-if="members.length === 0" class="text-sm text-slate-500">暂无成员</div>
					<ul v-else class="space-y-2">
						<li v-for="item in members.slice(0, 6)" :key="item.user?.id" class="flex items-center justify-between rounded-lg bg-white px-3 py-2">
							<span class="text-sm text-slate-700">{{ item.user?.username || '未知用户' }}</span>
							<span class="tag-pill">{{ item.role }}</span>
						</li>
					</ul>
				</div>
			</div>
		</div>
	</Team>
</template>

<script setup>
import {computed, onMounted, ref} from 'vue';
import {useRoute} from 'vue-router';
import ApiService from '@/services/ApiService.js';
import Team from '@/views/Teams/Team.vue';

const route = useRoute();
const teamId = computed(() => String(route.params.id || ''));

const loading = ref(false);
const team = ref({
	id: '',
	name: '',
	description: '',
	join_able: false,
	role: '',
});
const members = ref([]);

const fetchTeam = async () => {
	const response = await ApiService.getTeam(teamId.value);
	team.value = response?.data || team.value;
};

const fetchMembers = async () => {
	const response = await ApiService.getTeamMembers(teamId.value);
	members.value = Array.isArray(response?.data?.members) ? response.data.members : [];
};

const loadAll = async () => {
	loading.value = true;
	try {
		await Promise.all([fetchTeam(), fetchMembers()]);
	} catch (error) {
		console.error('load team data failed:', error);
	} finally {
		loading.value = false;
	}
};

onMounted(loadAll);
</script>
