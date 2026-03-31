<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-05-22 21:18:56  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->
<template>
	<Schedules>
		<div>
			<div v-if="loading" class="text-center text-gray-500 text-lg">加载中...</div>
			<div v-else>
				<div v-if="error" class="text-red-600 mb-4">{{ error }}</div>
				<div v-if="actionError" class="text-red-600 mb-4">{{ actionError }}</div>
				<div v-else class="space-y-6">
					<div class="bg-white shadow rounded-lg p-6 space-y-6">
						<h2 class="text-2xl font-semibold border-b pb-2">调度详情</h2>
						<div class="flex flex-wrap items-center gap-3">
							<button
								class="btn-ghost px-4 py-2 text-sm font-medium"
								:style="{ color: schedule.enabled ? '#c2410c' : '#15803d' }"
								:disabled="actionLoading"
								@click="toggleScheduleEnabled"
							>
								{{ actionLoading ? '处理中...' : (schedule.enabled ? '停用计划' : '启用计划') }}
							</button>
							<button
								class="btn-danger px-4 py-2 text-sm font-medium"
								:disabled="actionLoading"
								@click="deleteSchedule"
							>
								删除计划
							</button>
						</div>
						<dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-4">
							<div>
								<dt class="text-sm font-medium text-gray-500">ID</dt>
								<dd class="mt-1 text-gray-700">{{ schedule.id }}</dd>
							</div>
							<div>
								<dt class="text-sm font-medium text-gray-500">名称</dt>
								<dd class="mt-1 text-gray-700">{{ schedule.name }}</dd>
							</div>
							<div>
								<dt class="text-sm font-medium text-gray-500">描述</dt>
								<dd class="mt-1 text-gray-700">{{ schedule.description }}</dd>
							</div>
							<div>
								<dt class="text-sm font-medium text-gray-500">启用</dt>
								<dd class="mt-1 text-gray-700">{{ schedule.enabled ? '是' : '否' }}</dd>
							</div>
							<div>
								<dt class="text-sm font-medium text-gray-500">工作区</dt>
								<dd class="mt-1 text-gray-700">{{ schedule.workplace }}</dd>
							</div>
							<div class="sm:col-span-2">
								<dt class="text-sm font-medium text-gray-500">顺序</dt>
								<dd class="mt-1">
									<pre class="bg-gray-100 p-2 rounded text-sm overflow-auto">{{ formattedOrder }}</pre>
								</dd>
							</div>
							<div>
								<dt class="text-sm font-medium text-gray-500">创建时间</dt>
								<dd class="mt-1 text-gray-700">{{ formattedCreatedAt }}</dd>
							</div>
							<div>
								<dt class="text-sm font-medium text-gray-500">更新时间</dt>
								<dd class="mt-1 text-gray-700">{{ formattedUpdatedAt }}</dd>
							</div>
						</dl>
					</div>

					<div class="bg-white shadow rounded-lg p-6 space-y-5">
						<div class="flex flex-wrap items-center justify-between gap-3 border-b pb-2">
							<h3 class="text-xl font-semibold">运行记录与终端输出</h3>
							<button class="btn-muted px-3 py-1.5 text-sm font-medium" :disabled="runsLoading" @click="fetchScheduleRuns">
								{{ runsLoading ? '刷新中...' : '刷新' }}
							</button>
						</div>

						<div v-if="runsError" class="text-red-600 text-sm">{{ runsError }}</div>
						<div v-if="runsLoading" class="text-gray-500 text-sm">正在加载运行记录...</div>
						<div v-else-if="scheduleRuns.length === 0" class="text-gray-500 text-sm">当前计划还没有运行记录。</div>

						<div v-else class="grid gap-4 lg:grid-cols-[2fr,3fr]">
							<div class="max-h-[420px] space-y-2 overflow-auto rounded-lg border border-gray-200 p-3">
								<button
									v-for="run in scheduleRuns"
									:key="run.id"
									class="w-full rounded-lg border px-3 py-2 text-left transition"
									:class="selectedRun?.id === run.id ? 'border-blue-500 bg-blue-50' : 'border-gray-200 bg-white hover:bg-gray-50'"
									@click="selectRun(run.id)"
								>
									<p class="text-sm font-medium text-gray-800">{{ run.status || '-' }}</p>
									<p class="mt-1 text-xs text-gray-500">Run ID: {{ run.id }}</p>
									<p class="mt-1 text-xs text-gray-500">任务ID: {{ run.task_id || '-' }}</p>
									<p class="mt-1 text-xs text-gray-500">触发方式: {{ run.trigger_source || '-' }}</p>
									<p class="mt-1 text-xs text-gray-500">创建时间: {{ formatDate(run.created_at) }}</p>
								</button>
							</div>

							<div class="space-y-3">
								<div class="rounded-lg bg-gray-50 p-4 text-sm text-gray-700">
									<div class="grid gap-2 md:grid-cols-2">
										<p><span class="font-medium">Run ID:</span> {{ selectedRun?.id || '-' }}</p>
										<p><span class="font-medium">状态:</span> {{ selectedRun?.status || '-' }}</p>
										<p><span class="font-medium">任务ID:</span> {{ selectedRun?.task_id || '-' }}</p>
										<p><span class="font-medium">退出码:</span> {{ normalizeExitCode(selectedRun?.exit_code) }}</p>
										<p><span class="font-medium">开始时间:</span> {{ formatDate(selectedRun?.started_at) }}</p>
										<p><span class="font-medium">结束时间:</span> {{ formatDate(selectedRun?.finished_at) }}</p>
									</div>
								</div>
								<div>
									<p class="mb-2 text-sm font-medium text-gray-700">终端输出</p>
									<pre class="min-h-[300px] overflow-auto rounded-lg bg-slate-950 p-4 text-xs text-slate-100 whitespace-pre-wrap">{{ selectedRunOutput }}</pre>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	</Schedules>
</template>


<script setup>
import {ref, onMounted, computed} from 'vue';
import {useRoute} from 'vue-router';
import {useRouter} from 'vue-router';
import ApiService from '@/services/ApiService';
import Schedules from "@/views/Schedules/Schedules.vue";

/**
 * 中文: 调度详情页面
 * English: Schedule Detail Page
 */
const route = useRoute();
const router = useRouter();

/**
 * 中文: 从路由参数获取调度ID
 * English: Get schedule ID from route params
 */
const scheduleId = String(route.params.id || '');

const schedule = ref({});
const scheduleRuns = ref([]);
const runsLoading = ref(false);
const runsError = ref('');
const selectedRunId = ref('');
const loading = ref(false);
const error = ref(null);
const actionLoading = ref(false);
const actionError = ref('');

/**
 * 中文: 调用 API 获取指定 ID 的调度信息
 * English: Fetch schedule by ID from API
 */
async function fetchSchedule() {
	loading.value = true;
	error.value = null;
	try {
		// 直接调用 getSchedule 获取单个调度对象
		const response = await ApiService.getSchedule(scheduleId);
		schedule.value = response.data;
	} catch (err) {
		error.value = err.response?.data?.message || '获取调度失败';
	} finally {
		loading.value = false;
	}
}

async function fetchScheduleRuns() {
	runsLoading.value = true;
	runsError.value = '';
	try {
		const response = await ApiService.getScheduleRuns(scheduleId);
		const list = Array.isArray(response?.data) ? response.data : [];
		scheduleRuns.value = list;
		if (!selectedRunId.value && list.length > 0) {
			selectedRunId.value = list[0].id;
		}
		if (selectedRunId.value && !list.find(item => item.id === selectedRunId.value)) {
			selectedRunId.value = list[0]?.id || '';
		}
	} catch (err) {
		runsError.value = err.response?.data?.message || '获取运行记录失败';
	} finally {
		runsLoading.value = false;
	}
}

function selectRun(runId) {
	selectedRunId.value = runId;
}

function formatDate(value) {
	if (!value) {
		return '-';
	}
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) {
		return '-';
	}
	return date.toLocaleString();
}

function normalizeExitCode(value) {
	if (value === null || value === undefined || value === '') {
		return '-';
	}
	return String(value);
}

const selectedRun = computed(() => {
	return scheduleRuns.value.find(item => item.id === selectedRunId.value) || null;
});

const selectedRunOutput = computed(() => {
	const output = selectedRun.value?.output;
	if (typeof output !== 'string' || output.trim() === '') {
		return '暂无输出';
	}
	return output;
});

onMounted(() => {
	fetchSchedule();
	fetchScheduleRuns();
});

async function toggleScheduleEnabled() {
	if (!schedule.value?.id || actionLoading.value) {
		return;
	}
	actionError.value = '';
	actionLoading.value = true;
	const targetEnabled = !schedule.value.enabled;
	try {
		const response = targetEnabled
			? await ApiService.enableSchedule(schedule.value.id)
			: await ApiService.disableSchedule(schedule.value.id);
		schedule.value = {
			...schedule.value,
			...response.data,
		};
	} catch (err) {
		actionError.value = err.response?.data?.message || '更新计划状态失败';
	} finally {
		actionLoading.value = false;
	}
}

async function deleteSchedule() {
	if (!schedule.value?.id || actionLoading.value) {
		return;
	}
	if (!window.confirm(`确定删除计划 ${schedule.value.name || schedule.value.id} 吗？`)) {
		return;
	}
	actionError.value = '';
	actionLoading.value = true;
	try {
		await ApiService.deleteSchedule(schedule.value.id);
		const workplace = schedule.value.workplace || 'go-workspace';
		await router.push(`/aprons/workplaces/${workplace}/schedules`);
	} catch (err) {
		actionError.value = err.response?.data?.message || '删除计划失败';
		actionLoading.value = false;
	}
}

/**
 * 中文: 将 order 对象格式化为可读的 JSON 字符串
 * English: Format the order object to readable JSON string
 */
const formattedOrder = computed(() =>
	schedule.value.order
		? JSON.stringify(schedule.value.order, null, 2)
		: ''
);

/**
 * 中文: 将 ISO 时间字符串格式化为本地时间
 * English: Format ISO timestamp to local string
 */
const formattedCreatedAt = computed(() =>
	schedule.value.created_at
		? new Date(schedule.value.created_at).toLocaleString()
		: ''
);
const formattedUpdatedAt = computed(() =>
	schedule.value.updated_at
		? new Date(schedule.value.updated_at).toLocaleString()
		: ''
);
</script>

# === 以下功能：样式定义 ===
<style scoped>
</style>
