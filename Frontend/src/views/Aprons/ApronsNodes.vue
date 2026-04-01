<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - ApronsNodes.vue
  - Last Modified: 2025-05-22 20:47:09  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

# === 以下功能：优化节点视图 ===
<script lang="ts" setup>
import {ref, onMounted} from "vue";
import Aprons from "@/views/Aprons/Aprons.vue";
import ApiService from "@/services/ApiService";

interface CPUInfo {
	cpu_count: number | string;
	cpu_frequency: { current: number | string; max: number | string; min: number | string; };
	cpu_physical_cores: number | string;
}

interface MemoryInfo {
	available_memory: number | string;
	memory_percentage: number | string;
	total_memory: number | string;
	used_memory: number | string;
}

interface Node {
	id: number;
	name: string;
	description: string;
	status: string;
	ip_address: string;
	port: number;
	last_active_time: string;
	cpu_info: CPUInfo;
	HDID: string;
	memory_info: MemoryInfo;
}

const nodes = ref<Node[]>([]);

/**
 * 获取节点列表
 * Fetch the list of nodes
 */
// === 以下功能：获取并解析节点列表 ===
const fetchNodes = async () => {
	try {
		const response = await ApiService.getNodesList();
		console.log("Raw API Response:", response);

		if (response.data?.results && Array.isArray(response.data.results)) {
			nodes.value = response.data.results.map((node: any) => {
				// 解析 CPU 和内存信息
				// Parse CPU and memory info
				let cpu: CPUInfo | null = null;
				let mem: MemoryInfo | null = null;
				try {
					cpu = node.cpu_info ? JSON.parse(node.cpu_info) : null;
				} catch (e) {
					console.error("Failed to parse CPU Info:", e);
				}
				try {
					mem = node.memory_info ? JSON.parse(node.memory_info) : null;
				} catch (e) {
					console.error("Failed to parse Memory Info:", e);
				}

				// 重写 null 值
				// Replace null with placeholders
				const cpuInfo = cpu
					? {
						cpu_count: cpu.cpu_count ?? "-",
						cpu_frequency: {
							current: cpu.cpu_frequency.current ?? "-",
							max: cpu.cpu_frequency.max ?? "-",
							min: cpu.cpu_frequency.min ?? "-"
						},
						cpu_physical_cores: cpu.cpu_physical_cores ?? "-"
					}
					: {cpu_count: "-", cpu_frequency: {current: "-", max: "-", min: "-"}, cpu_physical_cores: "-"};
				const memInfo = mem
					? {
						available_memory: mem.available_memory ?? "-",
						memory_percentage: mem.memory_percentage ?? "-",
						total_memory: mem.total_memory ?? "-",
						used_memory: mem.used_memory ?? "-"
					}
					: {available_memory: "-", memory_percentage: "-", total_memory: "-", used_memory: "-"};

				return {
					id: node.id,
					name: node.name,
					description: node.description,
					status: node.status,
					ip_address: node.ip_address,
					port: node.port,
					last_active_time: node.last_active_time,
					HDID: node.HDID,
					cpu_info: cpuInfo,
					memory_info: memInfo,
				} as Node;
			});
		} else {
			console.error("Invalid response structure:", response);
			nodes.value = [];
		}
	} catch (error) {
		console.error("Failed to fetch nodes:", error);
		nodes.value = [];
	}
};

onMounted(fetchNodes);
</script>

<template>
	<Aprons>
		<div class="flex items-center mb-6">
			<h1 class="text-gray-500 text-3xl m-2">节点管理</h1>
			<RouterLink
				class="ml-auto rounded text-blue-600 hover:bg-gray-200 p-2"
				to="/aprons/nodes/running"
			>
				进行中任务
			</RouterLink>
			<RouterLink
				v-if="nodes.length"
				class="rounded text-green-600 hover:bg-gray-200 p-2"
				to="/aprons/node/create"
			>
				创建节点
			</RouterLink>
		</div>

		<div v-if="nodes.length" class="grid gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
			<div
				v-for="node in nodes"
				:key="node.id"
				class="p-2  rounded-lg bg-[#F9FAFB] transition-all hover:bg-gray-200"
			>
				<RouterLink :to="`/aprons/nodes/${node.id}`" class="block p-6">
					<div class="flex justify-between items-start">
						<h2 class="text-xl font-semibold text-gray-800 truncate">{{ node.name }}</h2>
						<span class="tag-pill">
              {{ node.status }}
            </span>
					</div>
					<p class="mt-2 text-sm text-gray-500 truncate">{{ node.description }}</p>

					<ul class="mt-4 space-y-2 text-sm text-gray-600">
						<li><strong>ID:</strong> {{ node.id }}</li>
						<li><strong>IP:</strong> {{ node.ip_address }}:{{ node.port }}</li>
						<li><strong>Last Active:</strong> {{ new Date(node.last_active_time).toLocaleString() }}</li>
					</ul>

					<div class="mt-4 border-t pt-4 space-y-3">
						<div>
							<p class="text-sm font-semibold text-gray-700">CPU 信息 (HDID: {{ node.HDID }})</p>
							<p class="text-sm text-gray-600">核心数: {{ node.cpu_info.cpu_count }}</p>
							<p class="text-sm text-gray-600">频率: {{ node.cpu_info.cpu_frequency.current }} MHz</p>
						</div>
						<div>
							<p class="text-sm font-semibold text-gray-700">内存信息</p>
							<p class="text-sm text-gray-600">
								已用: {{
									node.memory_info.used_memory !== '-' ? (Number(node.memory_info.used_memory) / (1024 * 1024 * 1024)).toFixed(2) + ' GB' : '-'
								}}
							</p>
							<p class="text-sm text-gray-600">
								总计: {{
									node.memory_info.total_memory !== '-' ? (Number(node.memory_info.total_memory) / (1024 * 1024 * 1024)).toFixed(2) + ' GB' : '-'
								}}
							</p>
							<p class="text-sm text-gray-600">使用率: {{ node.memory_info.memory_percentage }}%</p>
						</div>
					</div>
				</RouterLink>
			</div>
		</div>
		<div v-else class="flex flex-col items-center justify-center h-full">
			<p class="text-gray-500 text-lg">还没有节点</p>
			<RouterLink
				class="mt-4 rounded text-green-600 hover:bg-gray-200 p-2"
				to="/aprons/node/create"
			>
				创建节点↗
			</RouterLink>
		</div>
	</Aprons>
</template>

<style scoped>
/* 自定义滚动条 */
::-webkit-scrollbar {
	width: 8px;
	height: 8px;
}

::-webkit-scrollbar-track {
	background: transparent;
}

::-webkit-scrollbar-thumb {
	background-color: rgba(107, 114, 128, 0.5);
	border-radius: 4px;
}
</style>
