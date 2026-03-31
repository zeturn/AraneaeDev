<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-05-19 00:16:13  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->
<script lang="ts" setup>
/**
 * 删除与更新节点视图组件
 * Delete and Update Node View Component
 */
import {ref, reactive, onMounted} from 'vue';
import {useRoute, useRouter} from 'vue-router';
import ApiService from '@/services/ApiService';
import CheckboxSquareField from '@/components/BeansDesign/Checkbox/CheckboxSquareField.vue';
import Node from '@/views/Nodes/Node.vue';

const route = useRoute();
const router = useRouter();
const nodeId = route.params.id as string;

// 表单数据模型，不包含 CPU Info 和 Memory Info
// Form model, excluding CPU Info and Memory Info
const form = reactive({
	name: '',
	description: '',
	status: '',
	ip_address: '',
	port: 0,
	rpc_url: '',
	celery_queue: '',
	is_enabled: false,
});

const message = ref('');

/**
 * 获取节点信息并填充表单
 * Fetch node data and populate form
 */
const fetchNode = async () => {
	try {
		const res = await ApiService.getNode(nodeId);
		const data = res.data;
		form.name = data.name;
		form.description = data.description;
		form.status = data.status;
		form.ip_address = data.ip_address;
		form.port = data.port;
		form.rpc_url = data.rpc_url;
		form.celery_queue = data.celery_queue;
		form.is_enabled = data.is_enabled;
	} catch (error) {
		console.error('Error fetching node:', error);
		message.value = 'Failed to load node.';
	}
};

/**
 * 更新节点信息
 * Update node information
 */
const updateNode = async () => {
	try {
		await ApiService.updateNode(nodeId, {
			name: form.name,
			description: form.description,
			status: form.status,
			ip_address: form.ip_address,
			port: form.port,
			rpc_url: form.rpc_url,
			celery_queue: form.celery_queue,
			is_enabled: form.is_enabled,
		});
		message.value = 'Node updated successfully!';
	} catch (error) {
		console.error('Error updating node:', error);
		message.value = 'Failed to update node.';
	}
};

/**
 * 删除节点
 * Delete node
 */
const deleteNode = async () => {
	try {
		await ApiService.deleteNode(nodeId);
		message.value = 'Node deleted successfully!';
		router.push({name: 'NodesList'}); // 跳转回节点列表
	} catch (error) {
		console.error('Error deleting node:', error);
		message.value = 'Failed to delete node.';
	}
};

onMounted(() => {
	fetchNode();
});
</script>

<template>
	<Node>
		<div class="container">
			<h1 class="text-3xl font-semibold mb-6 text-gray-500">节点设置</h1>
			<!-- 节点设置（更新/删除） -->
			<div v-if="form" class="max-w-4xl p-6 bg-white">
				<p v-if="message"
				   :class="{
						'text-green-500': message.includes('successfully'),
						'text-red-500': message.includes('Failed')
					}"
				   class="mb-6"
				>
					{{ message }}
				</p>
				<form class="grid grid-cols-1 gap-6" @submit.prevent="updateNode">
					<div>
						<label class="block mb-2 text-gray-700 text-sm font-medium">名称</label>
						<input v-model="form.name"
						       class="field-input"
						       type="text"
						       required
						       placeholder="请输入节点名称"
						/>
					</div>
					<div>
						<label class="block mb-2 text-gray-700 text-sm font-medium">描述</label>
						<textarea v-model="form.description"
						          class="field-input"
						          rows="3"
						          placeholder="请输入描述（可选）"
						></textarea>
					</div>
					<div>
						<label class="block mb-2 text-gray-700 text-sm font-medium">状态</label>
						<select v-model="form.status"
						        class="field-input"
						>
							<option value="active">启用</option>
							<option value="inactive">停用</option>
						</select>
					</div>
					<div>
						<label class="block mb-2 text-gray-700 text-sm font-medium">IP 地址</label>
						<input v-model="form.ip_address"
						       class="field-input"
						       type="text"
						       placeholder="请输入节点 IP"
						/>
					</div>
					<div>
						<label class="block mb-2 text-gray-700 text-sm font-medium">端口</label>
						<input v-model="form.port"
						       class="field-input"
						       type="number"
						       placeholder="请输入端口"
						/>
					</div>
					<div>
						<label class="block mb-2 text-gray-700 text-sm font-medium">RPC URL</label>
						<input v-model="form.rpc_url"
						       class="field-input break-all"
						       type="text"
						       placeholder="请输入 RPC URL"
						/>
					</div>
					<div>
						<label class="block mb-2 text-gray-700 text-sm font-medium">Celery 队列</label>
						<input v-model="form.celery_queue"
						       class="field-input"
						       type="text"
						       placeholder="请输入 Celery 队列"
						/>
					</div>
					<div class="flex items-center">
						<CheckboxSquareField id="enabled" v-model="form.is_enabled">启用</CheckboxSquareField>
					</div>
					<div class="flex justify-end space-x-4 mt-4">
						<button
							class="btn-primary px-8 py-3"
							type="submit"
						>
							保存
						</button>
					</div>
				</form>
				<!-- 删除节点 -->
				<div class="flex justify-end mt-4">
					<button
						@click="deleteNode"
						class="btn-danger px-8 py-3"
					>
						删除节点
					</button>
				</div>
			</div>
			<!-- 加载状态提示 -->
			<div v-else class="flex justify-center items-center h-64 text-gray-400">
				<p>正在加载节点设置...</p>
			</div>
		</div>
	</Node>
</template>

