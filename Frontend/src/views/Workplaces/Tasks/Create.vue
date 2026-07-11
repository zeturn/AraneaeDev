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
			<div class="mx-auto my-6 w-full max-w-xl overflow-x-hidden rounded-2xl bg-white p-4 sm:p-6">
				<form @submit.prevent="submitForm" class="space-y-5">
					<!-- 名称 -->
					<div>
						<label for="name" class="block mb-2 text-gray-700 text-sm font-medium">名称</label>
						<input
							v-model="form.name"
							id="name"
							type="text"
							required
							class="field-input"
							placeholder="请输入任务名称"
						/>
					</div>

					<!-- 模式 -->
					<div v-if="!isGoApi">
						<label for="mode" class="block mb-2 text-gray-700 text-sm font-medium">模式</label>
						<el-select
							v-model="form.mode"
							id="mode"
							class="w-full"
						>
							<el-option label="一次性" value="once" />
							<el-option label="循环" value="recurring" />
						</el-select>
					</div>
					<div v-if="isGoApi" class="grid grid-cols-1 gap-5 md:grid-cols-2">
						<div>
							<label for="project_id" class="block mb-2 text-gray-700 text-sm font-medium">项目</label>
							<el-select
								v-model="goForm.project_id"
								id="project_id"
								class="w-full"
								placeholder="请选择项目"
								:loading="goProjectLoading"
							>
								<el-option
									v-for="project in goProjects"
									:key="project.id"
									:label="formatProjectLabel(project)"
									:value="project.id"
								/>
							</el-select>
							<p v-if="!goProjectLoading && goProjects.length === 0" class="mt-2 text-xs text-slate-500">暂无可用项目，请先创建项目。</p>
						</div>
						<div>
							<label for="version_id" class="block mb-2 text-gray-700 text-sm font-medium">版本</label>
							<el-select
								v-model="goForm.version_id"
								id="version_id"
								:disabled="!goForm.project_id || goVersionLoading"
								class="w-full"
								:placeholder="goForm.project_id ? '请选择版本' : '请先选择项目'"
								:loading="goVersionLoading"
							>
								<el-option
									v-for="version in goVersions"
									:key="version.id"
									:label="formatVersionLabel(version)"
									:value="version.id"
								/>
							</el-select>
							<p v-if="goForm.project_id && !goVersionLoading && goVersions.length === 0" class="mt-2 text-xs text-slate-500">该项目暂无可用版本，请先上传版本。</p>
						</div>
					</div>
					<div v-if="isGoApi" class="grid grid-cols-1 gap-5 md:grid-cols-2">
						<div>
							<label for="entry_command" class="block mb-2 text-gray-700 text-sm font-medium">执行命令</label>
							<input
								v-model="goForm.entry_command"
								id="entry_command"
								type="text"
								required
								class="field-input"
								placeholder="例如: bash run.sh"
							/>
						</div>
						<div>
							<label for="node_queue" class="block mb-2 text-gray-700 text-sm font-medium">节点队列</label>
							<input
								v-model="goForm.node_queue"
								id="node_queue"
								type="text"
								class="field-input"
								placeholder="默认 default"
							/>
						</div>
					</div>

					<!-- 描述 -->
					<div>
						<label for="description" class="block mb-2 text-gray-700 text-sm font-medium">描述</label>
						<textarea
							v-model="form.description"
							id="description"
							rows="3"
							class="field-input resize-none"
							placeholder="请输入任务描述"
						></textarea>
					</div>
					<!-- 启用 -->
					<div v-if="!isGoApi" class="mb-2">
						<CheckboxSquareField id="enabled" v-model="form.enabled">启用</CheckboxSquareField>
					</div>
					<!-- 提交按钮 -->
					<div>
						<button
							type="submit"
							:disabled="loading"
							class="btn-primary w-full disabled:opacity-50"
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
import {onMounted, reactive, ref, watch} from 'vue';
import {useRoute, useRouter} from 'vue-router';
import ApiService from '@/services/ApiService.js';
import CheckboxSquareField from '@/components/BeansDesign/Checkbox/CheckboxSquareField.vue';
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
const goProjectLoading = ref(false);
const goVersionLoading = ref(false);
const goProjects = ref<Array<{id: string; name?: string}>>([]);
const goVersions = ref<Array<{id: string; file_name?: string; version_hash?: string; created_at?: string; release_date?: string}>>([]);
const goForm = reactive({
	project_id: '',
	version_id: '',
	entry_command: 'bash run.sh',
	node_queue: 'default',
});

function formatProjectLabel(project: {id: string; name?: string}) {
	const projectName = String(project?.name || '').trim() || 'untitled-project';
	const shortId = String(project?.id || '').slice(0, 8);
	return shortId ? `${projectName} (${shortId})` : projectName;
}

function formatVersionLabel(version: {id: string; file_name?: string; version_hash?: string; created_at?: string; release_date?: string}) {
	const fileName = String(version?.file_name || '').trim() || 'artifact';
	const hash = String(version?.version_hash || version?.id || '').slice(0, 8);
	return hash ? `${fileName} (${hash})` : fileName;
}

async function loadGoProjects() {
	if (!isGoApi) {
		return;
	}
	goProjectLoading.value = true;
	try {
		const response = await ApiService.getWorkplaceProjects(workplaceId);
		const payload = response?.data;
		goProjects.value = Array.isArray(payload)
			? payload
			: (Array.isArray(payload?.results) ? payload.results : []);
	} catch (e) {
		goProjects.value = [];
	} finally {
		goProjectLoading.value = false;
	}
}

async function loadGoVersions(projectId: string) {
	if (!isGoApi || !projectId) {
		goVersions.value = [];
		return;
	}
	goVersionLoading.value = true;
	try {
		const response = await ApiService.getVersionsFromProject(projectId);
		const payload = response?.data;
		goVersions.value = Array.isArray(payload)
			? payload
			: (Array.isArray(payload?.versions) ? payload.versions : []);
	} catch (e) {
		goVersions.value = [];
	} finally {
		goVersionLoading.value = false;
	}
}

watch(
	() => goForm.project_id,
	async (projectId, prevId) => {
		if (!isGoApi) {
			return;
		}
		if (!projectId) {
			goForm.version_id = '';
			goVersions.value = [];
			return;
		}
		if (projectId !== prevId) {
			goForm.version_id = '';
		}
		await loadGoVersions(projectId);
	}
);

onMounted(async () => {
	if (!isGoApi) {
		return;
	}
	await loadGoProjects();
});

/**
 * 中文: 提交任务创建表单
 * English: Submit task creation form
 */
async function submitForm() {
	loading.value = true;
	error.value = null;
	if (!isGoApi && !form.mode) {
		error.value = '请选择模式';
		loading.value = false;
		return;
	}
	if (isGoApi && !goForm.project_id) {
		error.value = '请选择项目';
		loading.value = false;
		return;
	}
	if (isGoApi && !goForm.version_id) {
		error.value = '请选择版本';
		loading.value = false;
		return;
	}
	try {
		const taskPayload = isGoApi
			? {
				name: form.name,
				project_id: goForm.project_id,
				version_id: goForm.version_id,
				entry_command: goForm.entry_command,
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



