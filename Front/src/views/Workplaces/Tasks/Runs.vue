<template>
	<Workplace>
		<Task>
			<div class="mx-auto max-w-6xl px-4 pb-10">
				<div class="surface-panel space-y-6">
					<header class="flex flex-wrap items-center justify-between gap-3">
						<div>
							<p class="text-xs uppercase tracking-wider text-slate-500">Task Runs</p>
							<h1 class="text-2xl font-semibold text-slate-900">运行记录与输出</h1>
							<p class="text-sm text-slate-500">查看任务每次执行状态、退出码和标准输出。</p>
						</div>
						<div class="flex items-center gap-2">
							<button class="btn-muted" :disabled="loading" @click="refreshData">
								{{ loading ? '刷新中...' : '刷新' }}
							</button>
							<button class="btn-primary" :disabled="loading" @click="runTaskOnce">
								手动运行一次
							</button>
						</div>
					</header>

					<div class="grid gap-4 lg:grid-cols-[2fr,3fr]">
						<section class="space-y-3">
							<div v-if="runs.length === 0" class="rounded-xl bg-slate-50 p-4 text-sm text-slate-500">
								当前还没有运行记录。
							</div>
							<div v-else class="space-y-2">
								<button
									v-for="run in runs"
									:key="run.id"
									class="btn-run-item"
									:class="selectedRun?.id === run.id ? 'btn-run-item-selected' : 'btn-run-item-idle'"
									@click="selectRun(run.id)"
								>
									<p class="text-sm font-medium text-slate-900">{{ run.status || 'unknown' }}</p>
									<p class="mt-1 text-xs text-slate-500">Run ID: {{ run.id }}</p>
									<p class="mt-1 text-xs text-slate-500">触发方式: {{ run.trigger_source || '-' }}</p>
									<p class="mt-1 text-xs text-slate-500">退出码: {{ normalizeExitCode(run.exit_code) }}</p>
									<p class="mt-1 text-xs text-slate-500">创建时间: {{ formatDate(run.created_at) }}</p>
								</button>
							</div>
						</section>

						<section class="space-y-3">
							<div class="rounded-xl bg-slate-50 p-4 text-sm text-slate-700">
								<div class="grid gap-2 md:grid-cols-2">
									<p><span class="font-medium">Run ID:</span> {{ selectedRun?.id || '-' }}</p>
									<p><span class="font-medium">状态:</span> {{ selectedRun?.status || '-' }}</p>
									<p><span class="font-medium">触发方式:</span> {{ selectedRun?.trigger_source || '-' }}</p>
									<p><span class="font-medium">退出码:</span> {{ normalizeExitCode(selectedRun?.exit_code) }}</p>
									<p><span class="font-medium">开始时间:</span> {{ formatDate(selectedRun?.started_at) }}</p>
									<p><span class="font-medium">结束时间:</span> {{ formatDate(selectedRun?.finished_at) }}</p>
								</div>
							</div>

							<div class="space-y-2">
								<p class="text-sm font-medium text-slate-700">运行输出</p>
								<pre class="min-h-[320px] overflow-auto rounded-xl bg-slate-950 p-4 text-xs text-slate-100 whitespace-pre-wrap">{{ selectedRunOutput }}</pre>
							</div>
						</section>
					</div>

					<p class="text-sm text-slate-500">{{ notice }}</p>
				</div>
			</div>
		</Task>
	</Workplace>
</template>

<script setup>
import {computed, onMounted, ref} from 'vue';
import {useRoute} from 'vue-router';
import ApiService from '@/services/ApiService.js';
import Workplace from '@/views/Workplaces/Workplace.vue';
import Task from '@/views/Workplaces/Tasks/Tasks.vue';

const route = useRoute();

const loading = ref(false);
const notice = ref('');
const runs = ref([]);
const selectedRunId = ref('');

const taskId = computed(() => String(route.params.taskId || ''));

const selectedRun = computed(() => {
	return runs.value.find(item => item.id === selectedRunId.value) || null;
});

const selectedRunOutput = computed(() => {
	const output = selectedRun.value?.output;
	if (typeof output !== 'string' || output.trim() === '') {
		return '暂无输出';
	}
	return output;
});

function formatDate(value) {
	if (!value) {
		return '-';
	}
	return new Date(value).toLocaleString();
}

function normalizeExitCode(value) {
	if (value === null || value === undefined || value === '') {
		return '-';
	}
	return String(value);
}

function selectRun(runId) {
	selectedRunId.value = runId;
}

async function fetchRuns() {
	if (!taskId.value) {
		notice.value = '无效任务 ID';
		return;
	}
	loading.value = true;
	try {
		const response = await ApiService.getTaskRuns(taskId.value);
		const list = Array.isArray(response?.data) ? response.data : [];
		runs.value = list;
		if (!selectedRunId.value && list.length > 0) {
			selectedRunId.value = list[0].id;
		}
		if (selectedRunId.value && !list.find(item => item.id === selectedRunId.value)) {
			selectedRunId.value = list[0]?.id || '';
		}
		notice.value = `共 ${list.length} 条运行记录`;
	} catch (error) {
		console.error('fetch task runs failed:', error);
		notice.value = error?.response?.data?.detail || '加载运行记录失败';
	} finally {
		loading.value = false;
	}
}

async function runTaskOnce() {
	if (!taskId.value) {
		notice.value = '无效任务 ID';
		return;
	}
	loading.value = true;
	try {
		const response = await ApiService.triggerTask(taskId.value);
		const runId = response?.data?.id;
		notice.value = runId ? `任务已触发，运行 ID: ${runId}` : '任务已触发';
		await fetchRuns();
		if (runId) {
			selectedRunId.value = runId;
		}
	} catch (error) {
		console.error('trigger task failed:', error);
		notice.value = error?.response?.data?.detail || '触发任务失败';
	} finally {
		loading.value = false;
	}
}

async function refreshData() {
	await fetchRuns();
}

onMounted(fetchRuns);
</script>
