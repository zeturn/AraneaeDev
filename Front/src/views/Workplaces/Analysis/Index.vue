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
		<div class="p-6 bg-white rounded-2xl workplace-logs">
			<h2 class="text-2xl font-semibold mb-4">工作区日志记录 Task Records</h2>
			<div class="overflow-y-auto">
				<table class="min-w-full divide-gray-200 table-auto">
					<thead class="bg-gray-50">
					<tr>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">任务ID
							/ Task ID
						</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">任务状态
							/ Status
						</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">任务结果
							/ Result
						</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">创建时间
							/ Created At
						</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">更新时间
							/ Updated At
						</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">节点
							/ Node
						</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">项目
							/ Project
						</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">版本
							/ Version
						</th>
						<th class="px-4 py-2 text-left text-sm font-medium text-gray-500 uppercase tracking-wider">调度
							/ Schedule
						</th>
					</tr>
					</thead>
					<tbody class="bg-white divide-y divide-gray-200">
					<tr v-for="record in records" :key="record.id" class="hover:bg-gray-50">
						<td class="px-4 py-3 text-sm text-gray-700">{{ record.id }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ record.task_id }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ record.task_status }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ record.task_result }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ formatDate(record.task_created_at) }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ formatDate(record.task_updated_at) }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ record.node }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ record.project }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ record.version }}</td>
						<td class="px-4 py-3 text-sm text-gray-700">{{ record.schedule }}</td>
					</tr>
					</tbody>
				</table>
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
			count: 0 // 日志总数 / Total count of records
		};
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
				this.records = response.data.records;
				this.count = response.data.count;
			} catch (error) {
				console.error('获取工作区日志数据时出错:', error);
			}
		},

		/**
		 * 格式化日期
		 * Format date string to readable format
		 * @param {string} dateString ISO 格式日期字符串 / ISO date string
		 * @returns {string} 本地化日期字符串 / Localized date string
		 */
		formatDate(dateString) {
			const options = {year: 'numeric', month: 'long', day: 'numeric', hour: 'numeric', minute: 'numeric'};
			return new Date(dateString).toLocaleDateString(undefined, options);
		}
	}
};
</script>
