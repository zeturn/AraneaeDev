<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - ApronsNodesCreate.vue
  - Last Modified: 2025-05-19 00:04:32  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<script lang="ts" setup>
import {onMounted, ref} from "vue";
import {useRouter} from "vue-router";
import Aprons from "@/views/Aprons/Aprons.vue";
import ApiService from '@/services/ApiService';
import EventBus from "@/utils/event-bus";

interface DiscoveredNode {
	ip: string;
	name: string;
	port: number;
	grpc_port: number;
	already_registered: boolean;
	registered_node_id: number | null;
	machine?: string | null;
	os?: string | null;
}

// 定义输入框绑定的变量
const nodeName = ref("");
const nodeIp = ref("");
const discoverLoading = ref(false);
const discoverError = ref("");
const discoveredNodes = ref<DiscoveredNode[]>([]);
const customCidr = ref("");
const router = useRouter(); // 获取 Vue Router 实例

const applyCandidate = (candidate: DiscoveredNode) => {
	nodeIp.value = candidate.ip;
	if (!nodeName.value) {
		nodeName.value = candidate.name || `node-${candidate.ip}`;
	}
};

const discoverNodes = async (scope: 'local' | 'custom' = 'local') => {
	discoverLoading.value = true;
	discoverError.value = "";
	try {
		const params: Record<string, string> = {scope};
		if (scope === 'custom') {
			if (!customCidr.value.trim()) {
				discoverError.value = "请输入 CIDR，例如 192.168.1.0/24";
				discoverLoading.value = false;
				return;
			}
			params.cidr = customCidr.value.trim();
		}
		const response = await ApiService.discoverNodes(params);
		discoveredNodes.value = response?.data?.candidates || [];
	} catch (error: any) {
		discoveredNodes.value = [];
		discoverError.value = error?.response?.data?.error || "扫描失败，请稍后重试";
	}
	discoverLoading.value = false;
};

// 处理节点创建
const createNode = async () => {
	if (!nodeIp.value || !nodeName.value) {
		alert("请填写完整的节点信息！");
		return;
	}

	try {
		const response = await ApiService.registerNodes(nodeIp.value, nodeName.value);
		console.log("Node created:", response);
		const newId = response.data.id; // 假设返回的数据中包含新节点的 ID
		EventBus.emit('notify', {
			type: 'success',
			title: '创建成功',
			message: '节点已成功注册'
		});
		// 跳转回节点列表页面
		await router.push({name: 'node', params: {id: newId}});
	} catch (error) {
		console.error("Failed to create node:", error);
		EventBus.emit('notify', {
			type: 'error',
			title: '创建失败',
			message: '节点注册失败'
		});
	}
};

onMounted(() => {
	discoverNodes('local');
});
</script>

<template>
	<Aprons>
		<div class="container">
			<div class="flex flex-row items-center mb-6">
				<h1 class="text-3xl font-semibold text-gray-500">创建节点</h1>
				<RouterLink
					class="ml-auto text-blue-500 hover:underline text-base"
					to="/aprons/nodes"
				>
					返回节点列表
				</RouterLink>
			</div>

			<form class="p-6 bg-white rounded-2xl mx-auto my-4" @submit.prevent="createNode">
				<div class="mb-5 rounded-xl border border-gray-200 bg-gray-50 p-4">
					<div class="flex flex-wrap items-center gap-2">
						<button
							class="rounded bg-blue-600 px-3 py-2 text-sm text-white hover:bg-blue-700 disabled:opacity-60"
							type="button"
							:disabled="discoverLoading"
							@click="discoverNodes('local')"
						>
							扫描本地/内网
						</button>
						<input
							v-model="customCidr"
							class="min-w-[220px] flex-1 rounded border border-gray-300 bg-white px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
							type="text"
							placeholder="自定义网段 CIDR，如 192.168.1.0/24"
						/>
						<button
							class="rounded bg-gray-800 px-3 py-2 text-sm text-white hover:bg-black disabled:opacity-60"
							type="button"
							:disabled="discoverLoading"
							@click="discoverNodes('custom')"
						>
							扫描自定义网段
						</button>
					</div>
					<p v-if="discoverLoading" class="mt-3 text-sm text-gray-500">正在扫描可用 worknode...</p>
					<p v-if="discoverError" class="mt-3 text-sm text-red-600">{{ discoverError }}</p>
					<div v-if="discoveredNodes.length" class="mt-3 space-y-2">
						<button
							v-for="candidate in discoveredNodes"
							:key="candidate.ip"
							class="flex w-full items-center justify-between rounded border border-gray-200 bg-white px-3 py-2 text-left hover:border-blue-300 hover:bg-blue-50"
							type="button"
							@click="applyCandidate(candidate)"
						>
							<span class="text-sm text-gray-700">
								{{ candidate.name }} · {{ candidate.ip }}:{{ candidate.port }}
								<span v-if="candidate.os"> · {{ candidate.os }}</span>
							</span>
							<span
								v-if="candidate.already_registered"
								class="rounded bg-amber-100 px-2 py-1 text-xs text-amber-700"
							>
								已注册
							</span>
							<span
								v-else
								class="rounded bg-green-100 px-2 py-1 text-xs text-green-700"
							>
								匹配
							</span>
						</button>
					</div>
				</div>

				<!-- 节点名称 -->
				<div class="mb-5">
					<label class="block mb-2 text-gray-700 text-sm font-medium">节点名称</label>
					<input
						v-model="nodeName"
						class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
						placeholder="请输入节点名称"
						type="text"
						required
					/>
				</div>

				<!-- 节点 IP -->
				<div class="mb-5">
					<label class="block mb-2 text-gray-700 text-sm font-medium">节点 IP 地址</label>
					<input
						v-model="nodeIp"
						class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
						placeholder="请输入节点 IP"
						type="text"
						required
					/>
				</div>

				<button
					class="w-full py-3 ring-green-400 text-green-600 rounded-lg hover:bg-green-200 transition-colors"
					type="submit"
				>
					创建节点
				</button>
			</form>
		</div>
	</Aprons>
</template>


<style scoped>
</style>
