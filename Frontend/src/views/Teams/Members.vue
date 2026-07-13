<template>
	<Team>
		<div class="mx-auto max-w-5xl px-4 pb-10">
			<div class="team-panel space-y-6">
				<header class="space-y-2">
					<p class="text-xs uppercase tracking-wide text-slate-500">Team Members</p>
					<h1 class="text-2xl font-semibold text-slate-900">{{ $t('添加成员') }}</h1>
					<p class="text-sm text-slate-500">{{ $t('通过用户列表选择，或直接输入用户名/用户ID进行批量添加。') }}</p>
				</header>

				<section class="team-card space-y-3">
					<div class="grid gap-3 md:grid-cols-2">
						<LSelect v-model="selectedUserId" :options="userOptions" :placeholder="$t('从用户列表选择')" />
						<input v-model="manualIdentifiers" class="field-input" :placeholder="$t('输入用户名/ID，多个用逗号分隔')" type="text" />
					</div>
					<div class="flex items-center gap-3">
						<button class="btn-primary" :disabled="submitting" @click="addMembers">{{ submitting ? '处理中...' : '添加成员' }}</button>
						<span class="text-sm text-slate-500">{{ notice }}</span>
					</div>
				</section>

				<section class="space-y-3">
					<h2 class="text-base font-semibold text-slate-900">{{ $t('成员列表') }}</h2>
					<div v-if="loading" class="text-sm text-slate-500">{{ $t('加载中...') }}</div>
					<div v-else-if="members.length === 0" class="text-sm text-slate-500">{{ $t('暂无成员') }}</div>
					<div v-else class="grid gap-3 md:grid-cols-2">
						<article v-for="item in members" :key="item.user?.id" class="team-card flex items-center justify-between gap-3 bg-white">
							<div>
								<p class="text-sm font-semibold text-slate-900">{{ item.user?.username || '未知用户' }}</p>
								<p class="text-xs text-slate-500">ID: {{ item.user?.id || '-' }}</p>
							</div>
							<div class="flex items-center gap-2">
								<span class="tag-pill">{{ item.role }}</span>
								<button
									v-if="item.role !== 'owner'"
									class="btn-danger"
									@click="removeMember(item)"
								>
									{{ $t('移除') }}
								</button>
							</div>
						</article>
					</div>
				</section>
			</div>
		</div>
	</Team>
</template>

<script setup>import { useI18n } from '@/i18n';
const { t } = useI18n();

import {computed, onMounted, ref} from 'vue';
import {useRoute} from 'vue-router';
import ApiService from '@/services/ApiService.js';
import LSelect from '@/components/LSelect.vue';
import Team from '@/views/Teams/Team.vue';

const route = useRoute();
const teamId = computed(() => String(route.params.id || ''));

const loading = ref(false);
const submitting = ref(false);
const notice = ref('');
const members = ref([]);
const users = ref([]);
const selectedUserId = ref('');
const manualIdentifiers = ref('');

const userOptions = computed(() => {
	return users.value.map((u) => ({
		label: `${u.username} (${u.role})`,
		value: String(u.id),
	}));
});

const fetchMembers = async () => {
	const response = await ApiService.getTeamMembers(teamId.value);
	members.value = Array.isArray(response?.data?.members) ? response.data.members : [];
};

const fetchUsers = async () => {
	try {
		const response = await ApiService.getUsers();
		users.value = Array.isArray(response?.data?.results)
			? response.data.results
			: (Array.isArray(response?.data) ? response.data : []);
	} catch (_) {
		users.value = [];
	}
};

const loadAll = async () => {
	loading.value = true;
	notice.value = '';
	try {
		await Promise.all([fetchMembers(), fetchUsers()]);
	} catch (error) {
		console.error('load team members failed:', error);
		notice.value = t('加载成员失败');
	} finally {
		loading.value = false;
	}
};

const addMembers = async () => {
	const manual = String(manualIdentifiers.value || '')
		.split(',')
		.map(v => v.trim())
		.filter(Boolean);
	const ids = [...new Set([selectedUserId.value, ...manual].filter(Boolean))];
	if (ids.length === 0) {
		notice.value = t('请先选择或输入成员');
		return;
	}

	submitting.value = true;
	notice.value = '';
	try {
		const response = await ApiService.addTeamMembers(teamId.value, ids);
		const added = Number(response?.data?.added || 0);
		const skipped = Number(response?.data?.skipped || 0);
		const notFound = Array.isArray(response?.data?.not_found) ? response.data.not_found : [];
		notice.value = `添加完成：新增 ${added}，跳过 ${skipped}${notFound.length ? `，未找到 ${notFound.join(', ')}` : ''}`;
		selectedUserId.value = '';
		manualIdentifiers.value = '';
		await fetchMembers();
	} catch (error) {
		console.error('add team members failed:', error);
		notice.value = error?.response?.data?.detail || t('添加成员失败');
	} finally {
		submitting.value = false;
	}
};

const removeMember = async (item) => {
	if (!item?.user?.id) {
		return;
	}
	if (!window.confirm(`确认移除成员 ${item.user.username || item.user.id} ?`)) {
		return;
	}
	try {
		await ApiService.removeTeamMember(teamId.value, item.user.id);
		notice.value = t('成员已移除');
		await fetchMembers();
	} catch (error) {
		console.error('remove team member failed:', error);
		notice.value = error?.response?.data?.detail || t('移除成员失败');
	}
};

onMounted(loadAll);
</script>
