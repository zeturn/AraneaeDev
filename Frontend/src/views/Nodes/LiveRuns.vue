<template>
	<Aprons>
		<div class="space-y-5">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<div>
					<h1 class="text-2xl font-semibold text-slate-800">{{ $t('节点进行中任务') }}</h1>
					<p class="text-sm text-slate-500">{{ $t('实时查看所有节点当前 queued/running 的任务。') }}</p>
				</div>
				<div class="flex items-center gap-2">
					<span class="text-xs text-slate-500">最后刷新: {{ lastUpdatedText }}</span>
					<button class="btn-muted px-3 py-1.5 text-sm" :disabled="loading" @click="fetchRuns">
						{{ loading ? '刷新中...' : '手动刷新' }}
					</button>
				</div>
			</div>

			<div v-if="error" class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
				{{ error }}
			</div>

			<div v-if="loading && runs.length === 0" class="rounded-lg bg-slate-50 p-8 text-center text-sm text-slate-500">
				{{ $t('正在加载节点运行任务...') }}
			</div>

			<div v-else-if="runs.length === 0" class="rounded-lg bg-slate-50 p-8 text-center text-sm text-slate-500">
				{{ $t('当前没有进行中的任务。') }}
			</div>

			<div v-else class="grid gap-4 lg:grid-cols-[2fr,3fr]">
				<div class="max-h-[560px] space-y-2 overflow-auto rounded-xl border border-slate-200 p-3">
					<button
						v-for="run in runs"
						:key="run.id"
						class="w-full rounded-lg border px-3 py-2 text-left transition"
						:class="selectedRun?.id === run.id ? 'border-blue-500 bg-blue-50' : 'border-slate-200 bg-white hover:bg-slate-50'"
						@click="selectRun(run.id)"
					>
						<div class="flex items-center justify-between gap-2">
							<p class="text-sm font-medium text-slate-900">{{ run.node_name }}</p>
							<span class="tag-pill">
								{{ run.status }}
							</span>
						</div>
						<p class="mt-1 text-xs text-slate-500">任务: {{ run.task_name || run.task_id }}</p>
						<p class="mt-1 text-xs text-slate-500">队列: {{ run.node_queue }}</p>
						<p class="mt-1 text-xs text-slate-500">Run ID: {{ run.id }}</p>
						<p class="mt-1 text-xs text-slate-500">触发: {{ run.trigger_source || '-' }}</p>
						<p class="mt-1 text-xs text-slate-500">创建: {{ formatDate(run.created_at) }}</p>
					</button>
				</div>

				<div class="space-y-3">
					<div class="rounded-xl bg-slate-50 p-4 text-sm text-slate-700">
						<div class="grid gap-2 md:grid-cols-2">
							<p><span class="font-medium">{{ $t('节点:') }}</span> {{ selectedRun?.node_name || '-' }}</p>
							<p><span class="font-medium">{{ $t('状态:') }}</span> {{ selectedRun?.status || '-' }}</p>
							<p><span class="font-medium">{{ $t('任务名:') }}</span> {{ selectedRun?.task_name || '-' }}</p>
							<p><span class="font-medium">{{ $t('任务 ID:') }}</span> {{ selectedRun?.task_id || '-' }}</p>
							<p><span class="font-medium">Run ID:</span> {{ selectedRun?.id || '-' }}</p>
							<p><span class="font-medium">{{ $t('队列:') }}</span> {{ selectedRun?.node_queue || '-' }}</p>
							<p><span class="font-medium">{{ $t('触发:') }}</span> {{ selectedRun?.trigger_source || '-' }}</p>
							<p><span class="font-medium">{{ $t('开始:') }}</span> {{ formatDate(selectedRun?.started_at) }}</p>
						</div>
					</div>

					<div>
						<p class="mb-2 text-sm font-medium text-slate-700">{{ $t('终端输出（实时）') }}</p>
						<pre class="min-h-[340px] overflow-auto rounded-xl bg-slate-950 p-4 text-xs text-slate-100 whitespace-pre-wrap">{{ selectedRunOutput }}</pre>
					</div>
				</div>
			</div>
		</div>
	</Aprons>
</template>

<script setup>import { useI18n } from '@/i18n';
const { t } = useI18n();

import {computed, onBeforeUnmount, onMounted, ref} from 'vue';
import Aprons from '@/views/Aprons/Aprons.vue';
import ApiService from '@/services/ApiService';

const runs = ref([]);
const loading = ref(false);
const error = ref('');
const selectedRunId = ref('');
const lastUpdatedAt = ref(null);
const pollIntervalMs = 5000;
let pollTimer = null;

const selectedRun = computed(() => runs.value.find(item => item.id === selectedRunId.value) || null);

const selectedRunOutput = computed(() => {
	const out = selectedRun.value?.output;
	if (typeof out !== 'string' || out.trim() === '') {
		return t('暂无输出');
	}
	return out;
});

const lastUpdatedText = computed(() => {
	if (!lastUpdatedAt.value) {
		return '-';
	}
	return new Date(lastUpdatedAt.value).toLocaleTimeString();
});

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

function selectRun(runId) {
	selectedRunId.value = runId;
}

async function fetchRuns() {
	if (loading.value) {
		return;
	}
	loading.value = true;
	error.value = '';
	try {
		const response = await ApiService.getLiveNodeRuns();
		const list = Array.isArray(response?.data?.records) ? response.data.records : [];
		runs.value = list;
		lastUpdatedAt.value = Date.now();
		if (!selectedRunId.value && list.length > 0) {
			selectedRunId.value = list[0].id;
		}
		if (selectedRunId.value && !list.find(item => item.id === selectedRunId.value)) {
			selectedRunId.value = list[0]?.id || '';
		}
	} catch (err) {
		error.value = err?.response?.data?.message || err?.message || t('加载进行中任务失败');
	} finally {
		loading.value = false;
	}
}

onMounted(() => {
	fetchRuns();
	pollTimer = window.setInterval(fetchRuns, pollIntervalMs);
});

onBeforeUnmount(() => {
	if (pollTimer) {
		window.clearInterval(pollTimer);
		pollTimer = null;
	}
});
</script>
