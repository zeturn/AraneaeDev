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
import {ref} from "vue";
import {useRouter} from "vue-router";
import Aprons from "@/views/Aprons/Aprons.vue";
import ApiService from '@/services/ApiService';
import EventBus from "@/utils/event-bus";

// 定义输入框绑定的变量
const nodeName = ref("");
const nodeIp = ref("");
const router = useRouter(); // 获取 Vue Router 实例

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
