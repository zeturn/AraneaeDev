<template>
	<Workplace>
		<Task>
			<div class="mx-auto max-w-3xl px-4 pb-10">
				<div class="surface-panel space-y-6">
					<header class="space-y-2">
						<p class="text-xs uppercase tracking-wider text-slate-500">Task Settings</p>
						<h1 class="text-2xl font-semibold text-slate-900">{{ form.name || '任务设置' }}</h1>
						<p class="text-sm text-slate-500">修改任务名称、命令、队列与触发配置。</p>
					</header>

					<div class="grid gap-4 md:grid-cols-2">
						<div class="md:col-span-2">
							<label class="mb-2 block text-sm font-medium text-slate-700">任务名称</label>
							<input v-model="form.name" type="text" class="field-input" placeholder="输入任务名称" />
						</div>
						<div>
							<label class="mb-2 block text-sm font-medium text-slate-700">Project ID</label>
							<input v-model="form.project_id" type="text" class="field-input" placeholder="project id" />
						</div>
						<div>
							<label class="mb-2 block text-sm font-medium text-slate-700">Version ID</label>
							<input v-model="form.version_id" type="text" class="field-input" placeholder="version id" />
						</div>
						<div class="md:col-span-2">
							<label class="mb-2 block text-sm font-medium text-slate-700">执行命令</label>
							<input v-model="form.entry_command" type="text" class="field-input" placeholder="例如: python app.py" />
						</div>
						<div>
							<label class="mb-2 block text-sm font-medium text-slate-700">Cron 表达式</label>
							<input v-model="form.cron_expr" type="text" class="field-input" placeholder="留空表示仅手动/API触发" />
						</div>
						<div>
							<label class="mb-2 block text-sm font-medium text-slate-700">节点队列</label>
							<input v-model="form.node_queue" type="text" class="field-input" placeholder="default" />
						</div>
						<div class="md:col-span-2">
							<label class="inline-flex items-center gap-2 text-sm text-slate-700">
								<input v-model="form.enabled" type="checkbox" />
								启用任务
							</label>
						</div>
					</div>

					<div class="flex flex-wrap items-center gap-3">
						<button class="btn-primary" :disabled="loading" @click="saveTask">
							{{ loading ? '保存中...' : '保存设置' }}
						</button>
						<button class="btn-danger" :disabled="loading" @click="deleteTask">
							删除任务
						</button>
						<span class="text-sm text-slate-500">{{ notice }}</span>
					</div>
				</div>
			</div>
		</Task>
	</Workplace>
</template>

<script setup>
import {onMounted, reactive, ref} from 'vue';
import {useRoute, useRouter} from 'vue-router';
import ApiService from '@/services/ApiService.js';
import Workplace from '@/views/Workplaces/Workplace.vue';
import Task from '@/views/Workplaces/Tasks/Tasks.vue';

const route = useRoute();
const router = useRouter();

const loading = ref(false);
const notice = ref('');
const form = reactive({
	id: '',
	name: '',
	project_id: '',
	version_id: '',
	entry_command: '',
	cron_expr: '',
	node_queue: 'default',
	enabled: true,
});

const taskId = () => String(route.params.taskId || '');
const workplaceId = () => String(route.params.id || '');

const fetchTask = async () => {
	try {
		const response = await ApiService.getTask(taskId());
		const data = response?.data || {};
		form.id = data.id || taskId();
		form.name = data.name || '';
		form.project_id = data.project_id || '';
		form.version_id = data.version_id || '';
		form.entry_command = data.entry_command || '';
		form.cron_expr = data.cron_expr || '';
		form.node_queue = data.node_queue || 'default';
		form.enabled = data.enabled !== false;
		notice.value = '';
	} catch (error) {
		console.error('fetch task failed:', error);
		notice.value = '加载任务信息失败';
	}
};

const saveTask = async () => {
	if (!String(form.name || '').trim()) {
		notice.value = '任务名称不能为空';
		return;
	}
	if (!String(form.entry_command || '').trim()) {
		notice.value = '执行命令不能为空';
		return;
	}
	loading.value = true;
	notice.value = '';
	try {
		await ApiService.updateTask(taskId(), {
			name: String(form.name).trim(),
			project_id: String(form.project_id).trim(),
			version_id: String(form.version_id).trim(),
			entry_command: String(form.entry_command).trim(),
			cron_expr: String(form.cron_expr || '').trim(),
			node_queue: String(form.node_queue || 'default').trim() || 'default',
			enabled: !!form.enabled,
		});
		notice.value = '任务设置已保存';
	} catch (error) {
		console.error('update task failed:', error);
		notice.value = error?.response?.data?.detail || '保存失败';
	} finally {
		loading.value = false;
	}
};

const deleteTask = async () => {
	if (!window.confirm('确认删除当前任务？此操作不可撤销。')) {
		return;
	}
	loading.value = true;
	notice.value = '';
	try {
		await ApiService.deleteTask(taskId());
		await router.push(`/aprons/workplaces/${workplaceId()}/tasks`);
	} catch (error) {
		console.error('delete task failed:', error);
		notice.value = error?.response?.data?.detail || '删除失败';
	} finally {
		loading.value = false;
	}
};

onMounted(fetchTask);
</script>
