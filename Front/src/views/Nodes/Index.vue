<!--
  - Copyright (c)  2025.5.18
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-05-05 20:00:11  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<!-- src/views/Nodes/Index.vue -->
<!-- src/views/Nodes/Index.vue -->
<template>
	<Node>
		<div v-if="node" class="max-w-4xl mx-auto p-6 bg-white rounded-2xl border border-gray-200">
			<!-- …节点基本信息… -->

			<!-- 实时资源环形图 -->
			<div class="grid grid-cols-1 md:grid-cols-2 gap-6 mt-8">
				<div>
					<h2 class="text-xl font-semibold mb-2 text-gray-800">CPU 使用率</h2>
					<div ref="cpuChartRef" style="width:100%;height:250px;"></div>
				</div>
				<div>
					<h2 class="text-xl font-semibold mb-2 text-gray-800">内存使用率</h2>
					<div ref="memChartRef" style="width:100%;height:250px;"></div>
				</div>
			</div>
		</div>
		<div v-else class="flex justify-center items-center h-64 text-gray-500">
			<p>Loading node details...</p>
		</div>
	</Node>
</template>

<script lang="ts" setup>
import {ref, onMounted, onUnmounted, nextTick} from 'vue'
import {useRoute} from 'vue-router'
import * as echarts from 'echarts'
import ApiService from '@/services/ApiService'
import Node from '@/views/Nodes/Node.vue'

const route = useRoute()
const nodeId = route.params.id as string
const node = ref<any>(null)

const cpuChartRef = ref<HTMLElement | null>(null)
const memChartRef = ref<HTMLElement | null>(null)
let cpuChart: echarts.ECharts | null = null
let memChart: echarts.ECharts | null = null
let poller: number | null = null

// 获取节点基本信息
const fetchNode = async () => {
	try {
		const res = await ApiService.getNode(nodeId)
		node.value = res.data
	} catch (err) {
		console.error('Error fetching node:', err)
	}
}

// 初始化环形图：让 “Used” 扇区在中心显示，隐藏 “Free” 的标签
const initCharts = () => {
	if (cpuChartRef.value) {
		cpuChart = echarts.init(cpuChartRef.value)
		cpuChart.setOption({
			tooltip: {trigger: 'item'},
			series: [{
				type: 'pie',
				radius: ['50%', '70%'],
				center: ['50%', '50%'],
				label: {show: false},  // 默认隐藏
				data: [
					{
						value: 0,
						name: 'Used',
						label: {
							show: true,
							position: 'center',
							formatter: '{d}%',
							fontSize: 20,
							fontWeight: 'bold'
						}
					},
					{
						value: 100,
						name: 'Free',
						label: {show: false}
					}
				]
			}]
		})
	}

	if (memChartRef.value) {
		memChart = echarts.init(memChartRef.value)
		memChart.setOption({
			tooltip: {trigger: 'item'},
			series: [{
				type: 'pie',
				radius: ['50%', '70%'],
				center: ['50%', '50%'],
				label: {show: false},
				data: [
					{
						value: 0,
						name: 'Used',
						label: {
							show: true,
							position: 'center',
							formatter: '{d}%',
							fontSize: 20,
							fontWeight: 'bold'
						}
					},
					{
						value: 100,
						name: 'Free',
						label: {show: false}
					}
				]
			}]
		})
	}
}

// 拉取并更新数据：保持 “Used” 在 data[0]，并在更新时覆盖成带标签的对象
const fetchResourceStatus = async () => {
	try {
		const res = await ApiService.getNodeStatus(nodeId)
		const {cpu_percent, memory_used, memory_total} = res.data
		const memPercent = Number(((memory_used / memory_total) * 100).toFixed(1))

		cpuChart?.setOption({
			series: [{
				data: [
					{
						value: cpu_percent,
						name: 'Used',
						label: {
							show: true,
							position: 'center',
							formatter: `${cpu_percent}%`,
							fontSize: 20,
							fontWeight: 'bold'
						}
					},
					{value: 100 - cpu_percent, name: 'Free', label: {show: false}}
				]
			}]
		})

		memChart?.setOption({
			series: [{
				data: [
					{
						value: memPercent,
						name: 'Used',
						label: {
							show: true,
							position: 'center',
							formatter: `${memPercent}%`,
							fontSize: 20,
							fontWeight: 'bold'
						}
					},
					{value: 100 - memPercent, name: 'Free', label: {show: false}}
				]
			}]
		})
	} catch (err) {
		console.error('Error fetching resource status:', err)
	}
}

// 组件生命周期
onMounted(async () => {
	await nextTick()
	await fetchNode()
	initCharts()
	await fetchResourceStatus()
	poller = window.setInterval(fetchResourceStatus, 5000)
	window.addEventListener('resize', () => {
		cpuChart?.resize()
		memChart?.resize()
	})
})

onUnmounted(() => {
	if (poller !== null) clearInterval(poller)
	cpuChart?.dispose()
	memChart?.dispose()
})
</script>
