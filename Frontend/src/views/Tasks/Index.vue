<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue (Global Tasks List)
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Tasks>
		<div class="p-6">
			<!-- Loading state -->
			<div v-if="loading" class="text-center py-12 text-gray-400">加载中...</div>

			<!-- Empty state -->
			<div v-else-if="tasks.length === 0" class="text-center py-12 text-gray-400">
				暂无任务。
			</div>

			<!-- Task list -->
			<div v-else>
				<!-- Sort controls -->
				<div class="flex items-center mb-4 space-x-2">
					<label for="sort" class="text-sm font-medium text-gray-600">排序：</label>
					<select
						id="sort"
						v-model="sortKey"
						class="field-input w-auto px-2 py-1"
					>
						<option value="name">名称</option>
						<option value="mode">模式</option>
						<option value="created_at">创建时间</option>
						<option value="updated_at">更新时间</option>
						<option value="enabled">状态</option>
					</select>
					<button
						class="btn-muted px-2 py-1 text-sm"
						@click="toggleSortOrder"
					>
						{{ sortOrderLabel }}
					</button>
				</div>

				<!-- Cards grid -->
				<div class="grid gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
					<div
						v-for="task in sortedTasks"
						:key="task.id"
						class="group rounded-xl bg-[#F9FAFB] p-6 hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500 cursor-default"
					>
						<h3 class="font-bold text-lg text-gray-800">{{ task.name }}</h3>
						<p class="text-sm text-gray-500 mt-1">{{ task.description }}</p>
						<div class="mt-4 flex flex-wrap gap-2">
							<span class="tag-pill">
								模式: {{ task.mode }}
							</span>
							<span class="tag-pill">
								{{ task.enabled ? '启用' : '禁用' }}
							</span>
						</div>
						<div class="mt-3 text-xs text-gray-400 space-y-0.5">
							<p>创建: {{ task.created_at }}</p>
							<p>更新: {{ task.updated_at }}</p>
						</div>
					</div>
				</div>
			</div>

			<div v-if="error" class="mt-4 text-red-500 text-sm">{{ error }}</div>
		</div>
	</Tasks>
</template>

<script>
import ApiService from '@/services/ApiService.js';
import Tasks from '@/views/Tasks/Tasks.vue';

export default {
	name: 'GlobalTasksIndex',
	components: {Tasks},
	data() {
		return {
			tasks: [],
			loading: false,
			error: null,
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
		async fetchTasks() {
			this.loading = true;
			this.error = null;
			try {
				const response = await ApiService.getTasks();
				// Handle paginated or flat list response
				this.tasks = response.data.results || response.data.tasks || response.data || [];
			} catch (e) {
				console.error('Error fetching tasks:', e);
				this.error = '获取任务列表失败';
			} finally {
				this.loading = false;
			}
		},
		toggleSortOrder() {
			this.sortOrder = this.sortOrder === 'asc' ? 'desc' : 'asc';
		},
	},
	mounted() {
		this.fetchTasks();
	},
};
</script>