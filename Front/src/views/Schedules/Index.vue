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
		<div class="">
			<div v-if="loading" class="text-center text-gray-500 text-lg">加载中...</div>
			<div v-else>
				<div v-if="error" class="text-red-600 mb-4">{{ error }}</div>
				<div v-else class="bg-white shadow rounded-lg p-6 space-y-6">
					<h2 class="text-2xl font-semibold border-b pb-2">调度详情</h2>
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
							<dt class="text-sm font-medium text-gray-500">模式</dt>
							<dd class="mt-1 text-gray-700">{{ schedule.mode }}</dd>
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
			</div>
		</div>
	</Schedules>
</template>


<script setup>
import {ref, onMounted, computed} from 'vue';
import {useRoute} from 'vue-router';
import ApiService from '@/services/ApiService';
import Schedules from "@/views/Schedules/Schedules.vue";

/**
 * 中文: 调度详情页面
 * English: Schedule Detail Page
 */
const route = useRoute();

/**
 * 中文: 从路由参数获取调度ID
 * English: Get schedule ID from route params
 */
const scheduleId = Number(route.params.id);

const schedule = ref({});
const loading = ref(false);
const error = ref(null);

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

onMounted(() => {
	fetchSchedule();
});

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
