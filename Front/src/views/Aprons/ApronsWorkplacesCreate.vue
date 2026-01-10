<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - ApronsWorkplacesCreate.vue
  - Last Modified: 2025-05-19 21:17:11  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->
<script lang="ts" setup>
import {ref, onMounted} from 'vue';
import ApiService from '@/services/ApiService';
import Aprons from "@/views/Aprons/Aprons.vue";
import {useRouter} from 'vue-router';
import EventBus from '@/utils/event-bus'

const router = useRouter();

interface Team {
	id: number;
	name: string;
	role: string;
}

interface WorkplaceInput {
	name: string;
	description: string;
	status: 'active' | 'inactive';
	team_id: number | null;
}

const workplaces = ref<any[]>([]);
const teams = ref<Team[]>([]);

const newWorkplace = ref<WorkplaceInput>({
	name: '',
	description: '',
	status: 'active',
	team_id: null,
});

const fetchTeams = async () => {
	try {
		const res = await ApiService.getMyTeams();
		// 分页接口返回 { results: [...] }
		teams.value = res.data.results;
	} catch (err) {
		console.error('Error fetching teams:', err);
	}
};

const createWorkplace = async () => {
	if (!newWorkplace.value.team_id) {
		EventBus.emit('notify', {
			type: 'warning',
			title: '缺少团队',
			message: '请选择一个团队'
		});
		return;
	}
	try {
		const res = await ApiService.createWorkplace(newWorkplace.value);
		const newId = res.data.id; // 拿到新建的workplace id
		newWorkplace.value = {name: '', description: '', status: 'active', team_id: null};

		EventBus.emit('notify', {
			type: 'success',
			title: '创建成功',
			message: '工作区已成功创建'
		});
		// 跳转到详情页
		await router.push({name: 'workplace', params: {id: newId}});
	} catch (err) {
		console.error('Error creating workplace:', err);

		EventBus.emit('notify', {
			type: 'error',
			title: '创建失败',
			message: (err as Error).message || '网络错误，请稍后重试'
		});
	}
};


onMounted(async () => {
	await Promise.all([fetchTeams()]);
});
</script>

<template>
	<Aprons>
		<div class="container">
			<h1 class="text-3xl font-semibold text-gray-500">创建工作区</h1>

			<form class="mb-8 p-6 bg-white  rounded-2xl my-4" @submit.prevent="createWorkplace">
				<!-- 团队下拉 -->
				<div class="mb-5">
					<label class="block mb-2 text-gray-700 text-sm font-medium">所属团队</label>
					<select
						v-model="newWorkplace.team_id"
						class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-600 focus:border-blue-600"
					>
						<option disabled value="">请选择团队</option>
						<option
							v-for="team in teams"
							:key="team.id"
							:value="team.id"
						>
							{{ team.name }}（角色: {{ team.role }}）
						</option>
					</select>
				</div>

				<!-- 名称输入 -->
				<div class="mb-5">
					<label class="block mb-2 text-gray-700 text-sm font-medium">名称</label>
					<input
						v-model="newWorkplace.name"
						class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
						placeholder="请输入工作区名称"
						required
						type="text"
					/>
				</div>

				<!-- 描述输入 -->
				<div class="mb-5">
					<label class="block mb-2 text-gray-700 text-sm font-medium">描述</label>
					<input
						v-model="newWorkplace.description"
						class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
						placeholder="请输入描述（可选）"
						type="text"
					/>
				</div>

				<!-- 状态选择 -->
				<div class="mb-6">
					<label class="block mb-2 text-gray-700 text-sm font-medium">状态</label>
					<select
						v-model="newWorkplace.status"
						class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
					>
						<option value="active">启用</option>
						<option value="inactive">停用</option>
					</select>
				</div>

				<button
					class="w-full py-3 ring-green-400 text-green-600 rounded-lg hover:bg-green-200 transition-colors"
					type="submit"
				>
					创建工作区
				</button>
			</form>
		</div>
	</Aprons>
</template>


<style scoped>
/* ... */
</style>
