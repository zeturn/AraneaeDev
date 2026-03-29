<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Create.vue
  - Last Modified: 2025-05-19 22:06:28  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Workplace>
		<Task>
			<div class="container max-w-lg mx-auto p-6 bg-white rounded-2xl my-8">
				<form @submit.prevent="submitForm" class="space-y-5">
					<!-- 名称 -->
					<div>
						<label for="name" class="block mb-2 text-gray-700 text-sm font-medium">名称</label>
						<input
							v-model="form.name"
							id="name"
							type="text"
							required
							class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
							placeholder="请输入任务名称"
						/>
					</div>
					<!-- 描述 -->
					<div>
						<label for="description" class="block mb-2 text-gray-700 text-sm font-medium">描述</label>
						<textarea
							v-model="form.description"
							id="description"
							rows="3"
							class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400 resize-none"
							placeholder="请输入任务描述"
						></textarea>
					</div>
					<!-- 模式 -->
					<div v-if="!isGoApi">
						<label for="mode" class="block mb-2 text-gray-700 text-sm font-medium">模式</label>
						<select
							v-model="form.mode"
							id="mode"
							required
							class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
						>
							<option value="once">一次性</option>
							<option value="recurring">循环</option>
						</select>
					</div>
					<div v-if="isGoApi">
						<label for="project_id" class="block mb-2 text-gray-700 text-sm font-medium">项目 ID</label>
						<input
							v-model="goForm.project_id"
							id="project_id"
							type="text"
							required
							class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
							placeholder="请输入项目 ID"
						/>
					</div>
					<div v-if="isGoApi">
						<label for="version_id" class="block mb-2 text-gray-700 text-sm font-medium">版本 ID</label>
						<input
							v-model="goForm.version_id"
							id="version_id"
							type="text"
							required
							class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
							placeholder="请输入版本 ID"
						/>
					</div>
					<div v-if="isGoApi">
						<label for="entry_command" class="block mb-2 text-gray-700 text-sm font-medium">执行命令</label>
						<input
							v-model="goForm.entry_command"
							id="entry_command"
							type="text"
							required
							class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
							placeholder="例如: bash run.sh"
						/>
					</div>
					<div v-if="isGoApi">
						<label for="cron_expr" class="block mb-2 text-gray-700 text-sm font-medium">Cron 表达式</label>
						<input
							v-model="goForm.cron_expr"
							id="cron_expr"
							type="text"
							class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
							placeholder="例如: */30 * * * * *（留空表示只支持手动/API触发）"
						/>
					</div>
					<div v-if="isGoApi">
						<label for="node_queue" class="block mb-2 text-gray-700 text-sm font-medium">节点队列</label>
						<input
							v-model="goForm.node_queue"
							id="node_queue"
							type="text"
							class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
							placeholder="默认 default"
						/>
					</div>
					<!-- 启用 -->
					<div v-if="!isGoApi" class="flex items-center mb-2">
						<input
							v-model="form.enabled"
							id="enabled"
							type="checkbox"
							class="h-4 w-4 text-blue-600 focus:ring-blue-400 border-gray-300 rounded"
						/>
						<label for="enabled" class="ml-2 text-gray-700 text-sm font-medium">启用</label>
					</div>
					<!-- 提交按钮 -->
					<div>
						<button
							type="submit"
							:disabled="loading"
							class="w-full py-3 bg-gray-800 text-white rounded-lg hover:bg-gray-900 transition-colors font-medium font-medium disabled:opacity-50"
						>
							{{ loading ? '提交中...' : '创建' }}
						</button>
					</div>
					<div v-if="error" class="text-red-500 text-sm mt-2">{{ error }}</div>
				</form>
			</div>
		</Task>
	</Workplace>
</template>


<script setup lang="ts">
import {reactive, ref} from 'vue';
import {useRoute, useRouter} from 'vue-router';
import ApiService from '@/services/ApiService.js';
import Workplace from '@/views/Workplaces/Workplace.vue';
import Task from '@/views/Workplaces/Tasks/Tasks.vue';

/**
 * 中文: 任务创建页面组件
 * English: Task Creation Page Component
 */
const route = useRoute();
const router = useRouter();

// 中文: 获取工作区ID
// English: Get workplace ID
const workplaceId = Number(route.params.id);

// 中文: 定义表单数据模型
// English: Define form data model
const form = reactive({
	name: '',
	description: '',
	mode: 'once',
	enabled: false
});

const loading = ref(false);
const error = ref<string | null>(null);
const isGoApi = ((import.meta.env.VITE_API_FLAVOR || 'django').toLowerCase() === 'go');
const goForm = reactive({
	project_id: '',
	version_id: '',
	entry_command: 'bash run.sh',
	cron_expr: '*/30 * * * * *',
	node_queue: 'default',
});

/**
 * 中文: 提交任务创建表单
 * English: Submit task creation form
 */
async function submitForm() {
	loading.value = true;
	error.value = null;
	try {
		const taskPayload = isGoApi
			? {
				name: form.name,
				project_id: goForm.project_id,
				version_id: goForm.version_id,
				entry_command: goForm.entry_command,
				cron_expr: goForm.cron_expr,
				node_queue: goForm.node_queue || 'default',
			}
			: {
				workplace: workplaceId,
				celery_label: 'schedule_task_execution',
				name: form.name,
				description: form.description,
				mode: form.mode,
				enabled: form.enabled
			};
		await ApiService.createTask(taskPayload);
		// 中文: 创建成功后跳转到任务列表
		// English: Redirect to task list on success
		await router.push(`/aprons/workplaces/${workplaceId}/tasks`);
	} catch (err: any) {
		error.value = err.response?.data?.message || '创建任务失败';
	} finally {
		loading.value = false;
	}
}
</script>



