<template>
	<Aprons>
		<section class="mx-auto flex max-w-5xl flex-col gap-6">
			<header class="flex flex-col gap-2">
				<h1 class="text-2xl font-semibold text-gray-900">RSS 订阅</h1>
				<p class="text-sm text-gray-500">输入 RSS 链接后，Araneae 会抓取 feed 并把条目保存到本地。</p>
			</header>

			<form class="flex flex-col gap-3 rounded border border-gray-200 bg-white p-4 md:flex-row" @submit.prevent="createSubscription">
				<select
					v-model="selectedWorkplaceId"
					class="rounded border border-gray-300 px-3 py-2 text-sm outline-none focus:border-blue-500 md:w-56"
					required
				>
					<option disabled value="">选择工作区</option>
					<option v-for="workplace in workplaces" :key="workplace.id" :value="String(workplace.id)">
						{{ workplace.name }}
					</option>
				</select>
				<input
					v-model.trim="rssUrl"
					class="min-w-0 flex-1 rounded border border-gray-300 px-3 py-2 text-sm outline-none focus:border-blue-500"
					placeholder="https://example.com/feed.xml"
					type="url"
					required
				/>
				<button class="btn-primary px-4 py-2 text-sm font-medium" :disabled="submitting">
					{{ submitting ? '抓取中...' : '添加订阅' }}
				</button>
			</form>

			<div v-if="error" class="rounded border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
				{{ error }}
			</div>
			<div v-if="message" class="rounded border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-700">
				{{ message }}
			</div>

			<div class="overflow-hidden rounded border border-gray-200 bg-white">
				<div class="flex items-center justify-between border-b border-gray-200 px-4 py-3">
					<h2 class="text-base font-semibold text-gray-900">已保存订阅</h2>
					<button class="btn-muted px-3 py-1.5 text-sm" :disabled="loading" @click="loadSubscriptions">刷新</button>
				</div>
				<div v-if="loading" class="px-4 py-6 text-sm text-gray-500">加载中...</div>
				<div v-else-if="subscriptions.length === 0" class="px-4 py-6 text-sm text-gray-500">还没有 RSS 订阅。</div>
				<ul v-else class="divide-y divide-gray-200">
					<li v-for="subscription in subscriptions" :key="subscription.id" class="flex flex-col gap-3 px-4 py-4">
						<div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
							<div class="min-w-0">
								<h3 class="truncate text-sm font-semibold text-gray-900">{{ subscription.title || subscription.url }}</h3>
								<a class="break-all text-xs text-blue-600" :href="subscription.url" target="_blank" rel="noreferrer">{{ subscription.url }}</a>
								<p class="mt-1 text-xs text-gray-500">
									本地目录：{{ subscription.storage_dir }}
								</p>
							</div>
							<div class="flex shrink-0 gap-2">
								<button class="btn-muted px-3 py-1.5 text-sm" @click="loadItems(subscription.id)">查看条目</button>
								<button class="btn-muted px-3 py-1.5 text-sm" @click="refreshSubscription(subscription.id)">重新抓取</button>
								<button class="btn-danger px-3 py-1.5 text-sm" @click="deleteSubscription(subscription.id)">删除</button>
							</div>
						</div>
						<div v-if="activeSubscriptionId === subscription.id" class="rounded bg-gray-50 p-3">
							<div v-if="itemsLoading" class="text-sm text-gray-500">条目加载中...</div>
							<ul v-else class="flex flex-col gap-2">
								<li v-for="item in items" :key="item.id" class="text-sm">
									<a v-if="item.link" class="font-medium text-blue-700" :href="item.link" target="_blank" rel="noreferrer">{{ item.title || item.link }}</a>
									<span v-else class="font-medium text-gray-800">{{ item.title || item.guid }}</span>
									<p class="mt-1 line-clamp-2 text-xs text-gray-500">{{ item.summary || item.content_path }}</p>
								</li>
								<li v-if="items.length === 0" class="text-sm text-gray-500">没有条目。</li>
							</ul>
						</div>
					</li>
				</ul>
			</div>
		</section>
	</Aprons>
</template>

<script setup>
import {onMounted, ref, watch} from 'vue';
import Aprons from './Aprons.vue';
import ApiService from '@/services/ApiService.js';

const rssUrl = ref('');
const subscriptions = ref([]);
const workplaces = ref([]);
const selectedWorkplaceId = ref('');
const items = ref([]);
const activeSubscriptionId = ref('');
const loading = ref(false);
const itemsLoading = ref(false);
const submitting = ref(false);
const error = ref('');
const message = ref('');

const setError = err => {
	error.value = err?.response?.data?.message || err?.response?.data || err?.message || '操作失败';
};

const loadSubscriptions = async () => {
	if (!selectedWorkplaceId.value) {
		subscriptions.value = [];
		return;
	}
	loading.value = true;
	error.value = '';
	try {
		const response = await ApiService.getRSSSubscriptions(selectedWorkplaceId.value);
		subscriptions.value = Array.isArray(response.data) ? response.data : [];
	} catch (err) {
		setError(err);
	} finally {
		loading.value = false;
	}
};

const createSubscription = async () => {
	if (!rssUrl.value) {
		return;
	}
	if (!selectedWorkplaceId.value) {
		setError(new Error('请先选择工作区'));
		return;
	}
	submitting.value = true;
	error.value = '';
	message.value = '';
	try {
		const response = await ApiService.createRSSSubscription(rssUrl.value, selectedWorkplaceId.value);
		const created = response?.data?.created || 0;
		const updated = response?.data?.updated || 0;
		message.value = `抓取完成：新增 ${created} 条，更新 ${updated} 条。`;
		rssUrl.value = '';
		await loadSubscriptions();
	} catch (err) {
		setError(err);
	} finally {
		submitting.value = false;
	}
};

const refreshSubscription = async id => {
	error.value = '';
	message.value = '';
	try {
		const response = await ApiService.refreshRSSSubscription(id);
		message.value = `抓取完成：新增 ${response?.data?.created || 0} 条，更新 ${response?.data?.updated || 0} 条。`;
		await loadSubscriptions();
		if (activeSubscriptionId.value === id) {
			await loadItems(id);
		}
	} catch (err) {
		setError(err);
	}
};

const loadItems = async id => {
	activeSubscriptionId.value = id;
	itemsLoading.value = true;
	error.value = '';
	try {
		const response = await ApiService.getRSSItems(id);
		items.value = Array.isArray(response.data) ? response.data : [];
	} catch (err) {
		setError(err);
	} finally {
		itemsLoading.value = false;
	}
};

const deleteSubscription = async id => {
	error.value = '';
	message.value = '';
	try {
		await ApiService.deleteRSSSubscription(id);
		message.value = '订阅已删除。';
		if (activeSubscriptionId.value === id) {
			activeSubscriptionId.value = '';
			items.value = [];
		}
		await loadSubscriptions();
	} catch (err) {
		setError(err);
	}
};

const loadWorkplaces = async () => {
	try {
		const response = await ApiService.getMyWorkplaces();
		const list = Array.isArray(response?.data?.results) ? response.data.results : [];
		workplaces.value = list;
		if (!selectedWorkplaceId.value && list.length > 0) {
			selectedWorkplaceId.value = String(list[0].id);
		}
	} catch (err) {
		setError(err);
	}
};

onMounted(async () => {
	await loadWorkplaces();
	await loadSubscriptions();
});

watch(selectedWorkplaceId, async () => {
	activeSubscriptionId.value = '';
	items.value = [];
	await loadSubscriptions();
});
</script>
