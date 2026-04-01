<!--
  - Copyright (c)   2024.11  Henry Zhao. All rights reserved.
  - From CA.
  -->

<template>
	<Workplace>
		<div class="mx-auto w-full max-w-4xl px-4 pb-10">
			<div class="surface-panel workplace-settings-panel space-y-6">
				<header class="space-y-2">
					<p class="text-xs uppercase tracking-wider text-slate-500">Workplace Settings</p>
					<h1 class="text-2xl font-semibold text-slate-900">{{ form.name || '工作区设置' }}</h1>
					<p class="text-sm text-slate-500">重命名、修改描述与状态，或删除该工作区。</p>
				</header>

				<div class="grid gap-4 md:grid-cols-2">
					<div class="md:col-span-2">
						<label class="mb-2 block text-sm font-medium text-slate-700">工作区名称</label>
						<input v-model="form.name" type="text" class="field-input" placeholder="输入新名称" />
					</div>
					<div class="md:col-span-2">
						<label class="mb-2 block text-sm font-medium text-slate-700">描述</label>
						<textarea v-model="form.description" rows="4" class="field-input" placeholder="输入工作区描述"></textarea>
					</div>
					<div class="md:col-span-2">
						<label class="mb-2 block text-sm font-medium text-slate-700">状态</label>
						<select v-model="form.status" class="field-input">
							<option value="active">active</option>
							<option value="inactive">inactive</option>
						</select>
					</div>
				</div>

				<div class="flex flex-wrap items-center gap-3">
					<button class="btn-primary" :disabled="loading" @click="saveWorkplace">
						{{ loading ? '保存中...' : '保存设置' }}
					</button>
					<button class="btn-danger" :disabled="loading" @click="confirmDelete">
						删除工作区
					</button>
					<span class="text-sm text-slate-500 settings-notice">{{ notice }}</span>
				</div>

				<div class="grid gap-3 text-sm text-slate-500 md:grid-cols-2">
					<p>创建时间: {{ formatDate(createdAt) }}</p>
					<p>更新时间: {{ formatDate(updatedAt) }}</p>
				</div>
			</div>
		</div>
	</Workplace>
</template>

<script setup>
import {onMounted, reactive, ref} from 'vue';
import {useRoute, useRouter} from 'vue-router';
import ApiService from '@/services/ApiService.js';
import Workplace from '@/views/Workplaces/Workplace.vue';

const route = useRoute();
const router = useRouter();

const loading = ref(false);
const notice = ref('');
const createdAt = ref('');
const updatedAt = ref('');
const form = reactive({
	id: '',
	name: '',
	description: '',
	status: 'active',
});

const formatDate = (value) => {
	if (!value) {
		return '-';
	}
	return new Date(value).toLocaleString();
};

const fetchWorkplace = async () => {
	try {
		const workplaceId = route.params.id;
		const response = await ApiService.getWorkplace(workplaceId);
		const data = response?.data || {};
		form.id = data.id;
		form.name = data.name || '';
		form.description = data.description || '';
		if (data.status) {
			form.status = data.status;
		} else {
			form.status = data.enabled === false ? 'inactive' : 'active';
		}
		createdAt.value = data.created_at || '';
		updatedAt.value = data.updated_at || '';
	} catch (error) {
		console.error('Error fetching workplace data:', error);
		notice.value = '加载工作区信息失败';
	}
};

const saveWorkplace = async () => {
	const name = String(form.name || '').trim();
	if (!name) {
		notice.value = '名称不能为空';
		return;
	}
	loading.value = true;
	notice.value = '';
	try {
		const enabled = form.status === 'active';
		await ApiService.updateWorkplace(form.id || route.params.id, {
			name,
			description: form.description,
			status: form.status,
			enabled,
		});
		notice.value = '工作区设置已保存';
		updatedAt.value = new Date().toISOString();
	} catch (error) {
		console.error('Error updating workplace:', error);
		notice.value = error?.response?.data?.detail || '保存失败';
	} finally {
		loading.value = false;
	}
};

const confirmDelete = async () => {
	if (!window.confirm('确认删除该工作区？此操作不可撤销。')) {
		return;
	}
	loading.value = true;
	notice.value = '';
	try {
		await ApiService.deleteWorkplace(form.id || route.params.id);
		await router.push('/aprons/workplaces');
	} catch (error) {
		console.error('Error deleting workplace:', error);
		notice.value = error?.response?.data?.detail || '删除失败';
	} finally {
		loading.value = false;
	}
};

onMounted(fetchWorkplace);
</script>

<style scoped>
.workplace-settings-panel {
	background: #f8fafc;
	border: 1px solid #e2e8f0;
	box-shadow: none;
}

.settings-notice {
	min-height: 1.25rem;
}
</style>

