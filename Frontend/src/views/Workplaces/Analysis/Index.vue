<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-05-19 17:29:03  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Workplace>
		<div class="p-6 bg-white rounded-2xl workplace-logs space-y-4">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<h2 class="text-2xl font-semibold">工作区日志记录 Task Records</h2>
				<div class="text-sm text-gray-500">共 {{ count }} 条</div>
			</div>
			<div class="overflow-y-auto">
				<table class="min-w-full divide-gray-200 table-auto">
					<thead class="bg-gray-50">
					<tr>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">运行ID / Run ID</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">任务ID / Task ID</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">任务状态 / Status</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">任务结果 / Result</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">创建时间 / Created At</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">更新时间 / Updated At</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">节点 / Node</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">项目 / Project</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">版本 / Version</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">调度 / Schedule</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">操作 / Actions</th>
					</tr>
					</thead>
					<tbody class="bg-white divide-y divide-gray-200">
					<tr v-for="(record, idx) in records" :key="record.id || `${record.task_id}-${idx}`" class="hover:bg-gray-50">
						<td class="px-4 py-3 text-sm text-gray-700">{{ record.id || '-' }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ record.task_id || '-' }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ record.task_status || '-' }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ summarizeResult(record.task_result) }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ formatDate(record.task_created_at) }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ formatDate(record.task_updated_at) }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ record.node || '-' }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ record.project || '-' }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ record.version || '-' }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ record.schedule || '-' }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">
							<div class="flex flex-wrap gap-2">
								<button class="rounded bg-slate-900 px-3 py-1 text-xs text-white" :disabled="!hasOutput(record)" @click="openOutput(record)">
									查看输出
								</button>
								<button class="rounded border border-slate-300 px-3 py-1 text-xs text-slate-700" :disabled="!record.task_id" @click="openTaskRuns(record.task_id)">
									历史页面
								</button>
							</div>
						</td>
					</tr>
					<tr v-if="records.length === 0">
						<td colspan="11" class="px-4 py-6 text-center text-sm text-gray-500">暂无运行记录</td>
					</tr>
					</tbody>
				</table>
			</div>

			<div v-if="selectedRecord" class="rounded-xl border border-slate-200 bg-slate-50 p-4 space-y-3">
				<div class="flex items-center justify-between">
					<div class="text-sm text-slate-700">
						<span class="font-medium">Run ID:</span> {{ selectedRecord.id || '-' }}
						<span class="ml-4 font-medium">状态:</span> {{ selectedRecord.task_status || '-' }}
					</div>
					<button class="rounded border border-slate-300 px-2 py-1 text-xs text-slate-700" @click="closeOutput">关闭</button>
				</div>
				<pre class="max-h-[360px] overflow-auto rounded bg-slate-950 p-4 text-xs text-slate-100 whitespace-pre-wrap">{{ selectedOutput }}</pre>
			</div>
		</div>
	</Workplace>
</template>

<script>
import ApiService from '@/services/ApiService.js'; // 引入 ApiService
import Workplace from '@/views/Workplaces/Workplace.vue'; // 引入 Workplace 模板

/**
 * 工作区任务日志视图组件
 * Workplace Task Records View Component
 */
export default {
	name: 'WorkplaceTaskRecords',
	components: {Workplace},
	data() {
		return {
			records: [], // 日志记录列表 / Task records list
			count: 0, // 日志总数 / Total count of records
			selectedRecord: null,
		};
	},
	computed: {
		selectedOutput() {
			if (!this.selectedRecord) {
				return '';
			}
			const out = this.selectedRecord.run_output || this.selectedRecord.task_result || '';
			return typeof out === 'string' && out.trim() !== '' ? out : '暂无输出';
		}
	},
	created() {
		this.fetchTaskRecords();
	},
	methods: {
		// 获取工作区ID
		// Get workplace ID from route parameters
		getWorkplaceIdFromURL() {
			return this.$route.params.id;
		},

		// 获取日志数据
		// Fetch log data for the workplace
		async fetchTaskRecords() {
			const workplaceId = this.getWorkplaceIdFromURL();
			try {
				const response = await ApiService.getWorkplaceTaskRecords(workplaceId);
				this.records = Array.isArray(response?.data?.records) ? response.data.records : [];
				this.count = Number(response?.data?.count) || this.records.length;
				if (this.selectedRecord) {
					const next = this.records.find(item => item.id === this.selectedRecord.id);
					this.selectedRecord = next || null;
				}
			} catch (error) {
				console.error('获取工作区日志数据时出错:', error);
			}
		},
		hasOutput(record) {
			const out = record?.run_output || record?.task_result || '';
			return typeof out === 'string' && out.trim() !== '';
		},
		summarizeResult(result) {
			if (!result) {
				return '-';
			}
			const text = String(result).replace(/\s+/g, ' ').trim();
			if (text.length <= 80) {
				return text;
			}
			return `${text.slice(0, 80)}...`;
		},
		openOutput(record) {
			this.selectedRecord = record;
		},
		closeOutput() {
			this.selectedRecord = null;
		},
		openTaskRuns(taskId) {
			if (!taskId) {
				return;
			}
			const workplaceId = this.getWorkplaceIdFromURL();
			this.$router.push(`/aprons/workplaces/${workplaceId}/tasks/${taskId}/runs`);
		},

		/**
		 * 格式化日期
		 * Format date string to readable format
		 * @param {string} dateString ISO 格式日期字符串 / ISO date string
		 * @returns {string} 本地化日期字符串 / Localized date string
		 */
		formatDate(dateString) {
			if (!dateString) {
				return '-';
			}
			const options = {year: 'numeric', month: 'long', day: 'numeric', hour: 'numeric', minute: 'numeric'};
			const date = new Date(dateString);
			if (Number.isNaN(date.getTime())) {
				return '-';
			}
			return date.toLocaleString(undefined, options);
		}
	}
};
</script>
