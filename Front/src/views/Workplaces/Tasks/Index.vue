<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-05-22 21:16:32  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Workplace>
		<Task>
			<div>
				<div v-if="tasks.length === 0" class="text-gray-500">没有可用的项目。</div>
				<div v-else>
					<!-- 排序控制 -->
					<div class="flex items-center mb-4 space-x-2">
						<label for="sort" class="font-medium">排序：</label>
						<select id="sort" v-model="sortKey" class="border rounded p-1">
							<option value="name">名称</option>
							<option value="mode">模式</option>
							<option value="created_at">创建时间</option>
							<option value="updated_at">更新时间</option>
							<option value="enabled">状态</option>
						</select>
						<button @click="toggleSortOrder" class="border rounded p-1">
							{{ sortOrderLabel }}
						</button>
					</div>

					<!-- 卡片网格 -->
					<div class="grid gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
						<div
							v-for="task in sortedTasks"
							:key="task.id"
							class="group rounded-xl bg-[#F9FAFB] p-6 hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
						>
							<h3 class="font-bold text-lg">{{ task.name }}</h3>
							<p class="text-sm">{{ task.description }}</p>
							<div class="mt-4 flex flex-wrap gap-2">
								<p class="rounded-lg bg-blue-100 px-3 py-1 text-xs font-semibold text-blue-600">模式: {{
									task.mode }}</p>
								<p class="rounded-lg bg-yellow-100 px-3 py-1 text-xs font-semibold text-yellow-600">状态:
									{{ task.enabled ? '启用' : '禁用' }}</p>
								<p class="text-xs text-gray-500 mt-2">创建: {{ task.created_at }}</p>
								<p class="text-xs text-gray-500">更新: {{ task.updated_at }}</p>
							</div>
						</div>
					</div>
				</div>
			</div>
		</Task>
	</Workplace>
</template>

<script>
import ApiService from "@/services/ApiService.js";
import Task from "@/views/Workplaces/Tasks/Tasks.vue";
import Workplace from "@/views/Workplaces/Workplace.vue";

export default {
	components: {Workplace, Task},
	data() {
		return {
			tasks: [],
			sortKey: 'name',
			sortOrder: 'asc',
		};
	},
	computed: {
		sortedTasks() {
			return [...this.tasks].sort((a, b) => {
				const key = this.sortKey;
				let aVal = a[key];
				let bVal = b[key];

				if (key === 'created_at' || key === 'updated_at') {
					aVal = new Date(aVal);
					bVal = new Date(bVal);
					return this.sortOrder === 'asc' ? aVal - bVal : bVal - aVal;
				}

				if (key === 'enabled') {
					aVal = aVal ? 1 : 0;
					bVal = bVal ? 1 : 0;
					return this.sortOrder === 'asc' ? aVal - bVal : bVal - aVal;
				}

				return this.sortOrder === 'asc'
					? String(aVal).localeCompare(String(bVal))
					: String(bVal).localeCompare(String(aVal));
			});
		},
		sortOrderLabel() {
			return this.sortOrder === 'asc' ? '升序' : '降序';
		},
	},
	methods: {
		getWorkplaceIdFromURL() {
			return this.$route.params.id;
		},
		async fetchWorkplaceTask() {
			const taskId = this.getWorkplaceIdFromURL();
			try {
				const response = await ApiService.getWorkplaceTasks(taskId);
				this.tasks = response.data.tasks;
			} catch (error) {
				console.error("Error fetching workplace task:", error);
			}
		},
		toggleSortOrder() {
			this.sortOrder = this.sortOrder === 'asc' ? 'desc' : 'asc';
		},
	},
	mounted() {
		this.fetchWorkplaceTask();
	},
};
</script>

