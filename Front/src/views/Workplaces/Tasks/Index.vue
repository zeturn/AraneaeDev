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
			<section class="space-y-4">
				<div class="surface-panel">
					<div class="mb-4 flex flex-wrap items-center gap-3">
						<label for="sort" class="text-sm font-medium text-slate-700">排序</label>
						<select id="sort" v-model="sortKey" class="field-input max-w-[220px]">
							<option value="name">名称</option>
							<option value="node_queue">节点队列</option>
							<option value="created_at">创建时间</option>
							<option value="enabled">状态</option>
						</select>
						<button class="btn-muted" @click="toggleSortOrder">{{ sortOrderLabel }}</button>
						<span class="text-sm text-slate-500">{{ notice }}</span>
					</div>

					<div v-if="tasks.length === 0" class="py-6 text-sm text-slate-500">没有可用任务。</div>
					<div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
						<article v-for="task in sortedTasks" :key="task.id" class="surface-card space-y-3">
							<header class="space-y-1">
								<h3 class="text-base font-semibold text-slate-900">{{ task.name || 'untitled-task' }}</h3>
								<p class="text-xs text-slate-500">ID: {{ task.id }}</p>
							</header>
							<div class="flex flex-wrap gap-2 text-xs">
								<span class="tag-pill">队列: {{ task.node_queue || 'default' }}</span>
								<span class="tag-pill">{{ task.enabled ? '已启用' : '已禁用' }}</span>
							</div>
							<p class="text-xs text-slate-500">创建时间: {{ formatDate(task.created_at) }}</p>
							<div class="flex flex-wrap gap-2 pt-1">
								<button class="btn-primary" @click="runTaskOnce(task)">手动运行一次</button>
								<button class="btn-muted" @click="renameTask(task)">重命名</button>
								<button class="btn-muted" @click="openRuns(task)">运行记录</button>
								<button class="btn-muted" @click="openSettings(task)">设置</button>
								<button class="btn-danger" @click="removeTask(task)">删除</button>
							</div>
						</article>
					</div>
				</div>
			</section>
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
			notice: '',
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

				if (key === 'created_at') {
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
		formatDate(value) {
			if (!value) {
				return '-';
			}
			return new Date(value).toLocaleString();
		},
		getWorkplaceIdFromURL() {
			return this.$route.params.id;
		},
		async fetchWorkplaceTask() {
			const taskId = this.getWorkplaceIdFromURL();
			try {
				const response = await ApiService.getWorkplaceTasks(taskId);
				this.tasks = response.data.tasks;
				this.notice = '';
			} catch (error) {
				console.error("Error fetching workplace task:", error);
				this.notice = '加载任务失败';
			}
		},
		toggleSortOrder() {
			this.sortOrder = this.sortOrder === 'asc' ? 'desc' : 'asc';
		},
		openSettings(task) {
			const workplaceId = this.getWorkplaceIdFromURL();
			this.$router.push(`/aprons/workplaces/${workplaceId}/tasks/${task.id}/settings`);
		},
		openRuns(task) {
			const workplaceId = this.getWorkplaceIdFromURL();
			this.$router.push(`/aprons/workplaces/${workplaceId}/tasks/${task.id}/runs`);
		},
		async renameTask(task) {
			const nextName = window.prompt('输入新任务名称', task.name || '');
			if (nextName === null) {
				return;
			}
			const name = nextName.trim();
			if (!name) {
				this.notice = '任务名称不能为空';
				return;
			}
			try {
				await ApiService.updateTask(task.id, {name});
				this.notice = '任务名称已更新';
				await this.fetchWorkplaceTask();
			} catch (error) {
				console.error('rename task failed:', error);
				this.notice = error?.response?.data?.detail || '更新任务失败';
			}
		},
		async removeTask(task) {
			if (!window.confirm(`确认删除任务 ${task.name || task.id} ?`)) {
				return;
			}
			try {
				await ApiService.deleteTask(task.id);
				this.notice = '任务已删除';
				await this.fetchWorkplaceTask();
			} catch (error) {
				console.error('delete task failed:', error);
				this.notice = error?.response?.data?.detail || '删除任务失败';
			}
		},
		async runTaskOnce(task) {
			try {
				const response = await ApiService.triggerTask(task.id);
				const runId = response?.data?.id;
				this.notice = runId ? `任务已触发，运行ID: ${runId}` : '任务已触发';
			} catch (error) {
				console.error('trigger task failed:', error);
				this.notice = error?.response?.data?.detail || '触发任务失败';
			}
		},
	},
	mounted() {
		this.fetchWorkplaceTask();
	},
};
</script>

