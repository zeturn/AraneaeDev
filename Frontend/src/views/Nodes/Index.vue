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

			<!-- 运行时环境能力 -->
			<div class="mt-8">
				<div class="flex items-center justify-between mb-3">
					<h2 class="text-xl font-semibold text-gray-800">运行时环境</h2>
					<button
						:disabled="capLoading"
						class="btn-primary px-4 py-1.5 text-sm disabled:opacity-50"
						@click="doRefreshCapabilities"
					>
						<span v-if="capLoading">检测中...</span>
						<span v-else>🔍 刷新检测</span>
					</button>
				</div>

				<div v-if="capError" class="text-sm text-red-500 mb-2">{{ capError }}</div>

				<div v-if="capabilities.length === 0 && !capLoading" class="text-sm text-gray-400">
					暂无数据，点击"刷新检测"从执行节点获取运行时列表。
				</div>

				<!-- Badge grid -->
				<div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3">
					<div
						v-for="cap in capabilities"
						:key="cap.key"
						:class="[
							'rounded-xl px-4 py-3 border transition-colors',
							installJobs[cap.key]?.status === 'running' || installJobs[cap.key]?.status === 'pending'
								? 'bg-yellow-50 border-yellow-300'
								: cap.available
									? 'bg-green-50 border-green-200'
									: 'bg-gray-50 border-gray-200'
						]"
					>
						<!-- Name + status dot -->
						<div class="flex items-center gap-2 mb-1">
							<span
								:class="{
									'bg-green-500': cap.available && !isInstalling(cap.key),
									'bg-gray-300': !cap.available && !isInstalling(cap.key),
									'bg-yellow-400 animate-pulse': isInstalling(cap.key),
								}"
								class="inline-block w-2 h-2 rounded-full flex-shrink-0"
							></span>
							<span class="font-semibold text-sm text-gray-800 truncate">{{ cap.name }}</span>
						</div>

						<!-- Status text / version -->
						<p v-if="isInstalling(cap.key)" class="text-xs text-yellow-600">安装中...</p>
						<p v-else-if="installJobs[cap.key]?.status === 'success'" class="text-xs text-green-600">安装成功 ✓</p>
						<p v-else-if="installJobs[cap.key]?.status === 'failed'" class="text-xs text-red-500">安装失败</p>
						<p v-else-if="cap.available" class="text-xs text-gray-500 truncate" :title="cap.version">{{ cap.version || '已安装' }}</p>
						<p v-else class="text-xs text-gray-400">未安装</p>

						<!-- Install button (show only when not installed and not currently installing) -->
						<button
							v-if="!cap.available && !isInstalling(cap.key) && installJobs[cap.key]?.status !== 'success'"
							class="btn-primary mt-2 w-full px-2 py-1 text-xs"
							@click="doInstall(cap.key)"
						>
							📦 安装
						</button>

						<!-- View log button (show when a job exists) -->
						<button
							v-if="installJobs[cap.key]"
							class="btn-muted mt-1 w-full px-2 py-1 text-xs"
							@click="openLog(cap.key)"
						>
							📋 查看日志
						</button>
					</div>
				</div>

				<!-- 安装日志面板 -->
				<div v-if="logPanelKey" class="mt-6 rounded-xl border border-gray-200 overflow-hidden">
					<div class="flex items-center justify-between bg-gray-800 px-4 py-2">
						<span class="text-sm font-medium text-white">📋 安装日志 — {{ logPanelKey }}</span>
						<button class="btn-muted px-2 py-1 text-xs text-gray-200" @click="logPanelKey = null">✕ 关闭</button>
					</div>
					<pre
						ref="logPanelRef"
						class="bg-gray-900 text-green-300 text-xs p-4 max-h-72 overflow-y-auto whitespace-pre-wrap font-mono"
					>{{ installJobs[logPanelKey]?.log || '（暂无日志）' }}</pre>
					<div class="bg-gray-800 px-4 py-1.5 flex gap-3 items-center">
						<span
							:class="{
								'text-yellow-400': isInstalling(logPanelKey),
								'text-green-400': installJobs[logPanelKey]?.status === 'success',
								'text-red-400': installJobs[logPanelKey]?.status === 'failed',
								'text-gray-400': installJobs[logPanelKey]?.status === 'pending',
							}"
							class="text-xs"
						>
							状态：{{ installJobs[logPanelKey]?.status }}
						</span>
						<button
							v-if="!isInstalling(logPanelKey)"
							class="btn-primary ml-auto px-3 py-1 text-xs"
							@click="doInstall(logPanelKey)"
						>
							重试安装
						</button>
					</div>
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
import { PieChart } from 'echarts/charts'
import { TooltipComponent } from 'echarts/components'
import { use, init, type ECharts } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import ApiService from '@/services/ApiService'
import Node from '@/views/Nodes/Node.vue'

use([PieChart, TooltipComponent, CanvasRenderer])

const route = useRoute()
const nodeId = route.params.id as string
const node = ref<any>(null)

const cpuChartRef = ref<HTMLElement | null>(null)
const memChartRef = ref<HTMLElement | null>(null)
let cpuChart: ECharts | null = null
let memChart: ECharts | null = null
let poller: number | null = null
const onWindowResize = () => {
	cpuChart?.resize()
	memChart?.resize()
}

// 运行时能力
const capabilities = ref<any[]>([])
const capLoading = ref(false)
const capError = ref<string | null>(null)

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
		cpuChart = init(cpuChartRef.value)
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
		memChart = init(memChartRef.value)
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
		const safeMemoryTotal = Number(memory_total) > 0 ? Number(memory_total) : 0
		const rawMemPercent = safeMemoryTotal > 0 ? (Number(memory_used) / safeMemoryTotal) * 100 : 0
		const memPercent = Number(Math.max(0, Math.min(100, rawMemPercent)).toFixed(1))
		const safeCpuPercent = Number(Math.max(0, Math.min(100, Number(cpu_percent) || 0)).toFixed(1))

		cpuChart?.setOption({
			series: [{
				data: [
					{
						value: safeCpuPercent,
						name: 'Used',
						label: {
							show: true,
							position: 'center',
							formatter: `${safeCpuPercent}%`,
							fontSize: 20,
							fontWeight: 'bold'
						}
					},
					{value: 100 - safeCpuPercent, name: 'Free', label: {show: false}}
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

/**
 * 从控制节点读取已存储的运行时能力（不主动探测节点）
 */
const fetchCapabilities = async () => {
	try {
		const res = await ApiService.getNodeCapabilities(nodeId)
		capabilities.value = res.data.capabilities || []
	} catch (err) {
		console.error('Error fetching stored capabilities:', err)
	}
}

/**
 * 主动触发执行节点探测，更新运行时能力列表
 */
const doRefreshCapabilities = async () => {
	capLoading.value = true
	capError.value = null
	try {
		const res = await ApiService.refreshNodeCapabilities(nodeId)
		capabilities.value = res.data.capabilities || []
	} catch (err: any) {
		console.error('Error refreshing capabilities:', err)
		capError.value = err?.response?.data?.error || '刷新失败，请检查节点是否在线'
	} finally {
		capLoading.value = false
	}
}

// ---------------------------------------------------------------------------
// 软件安装器状态
// ---------------------------------------------------------------------------
// job 字典：key → {job_id, status, log, ...}  （运行时 key 作为索引）
const installJobs = ref<Record<string, any>>({})
const logPanelKey = ref<string | null>(null)
const logPanelRef = ref<HTMLElement | null>(null)
let installPoller: number | null = null

/** 判断某个运行时是否正在安装中 */
const isInstalling = (key: string | null) => {
	if (!key) return false
	const st = installJobs.value[key]?.status
	return st === 'pending' || st === 'running'
}

/** 轮询所有活跃安装任务的进度 */
const pollInstallJobs = async () => {
	const activeKeys = Object.keys(installJobs.value).filter(k => isInstalling(k))
	if (activeKeys.length === 0) return
	for (const key of activeKeys) {
		const jobId = installJobs.value[key]?.job_id
		if (!jobId) continue
		try {
			const res = await ApiService.getInstallStatus(nodeId, jobId)
			installJobs.value[key] = {...res.data, job_id: jobId}
			// 自动滚动日志面板到底部
			if (logPanelRef.value && logPanelKey.value === key) {
				await nextTick()
				logPanelRef.value.scrollTop = logPanelRef.value.scrollHeight
			}
			// 安装完成后自动刷新能力列表
			const finalStatus = res.data.status
			if (finalStatus === 'success' || finalStatus === 'failed') {
				if (finalStatus === 'success') {
					await fetchCapabilities()
				}
			}
		} catch (err) {
			console.error('Error polling install status:', err)
		}
	}

	const hasActiveJobs = Object.keys(installJobs.value).some(k => isInstalling(k))
	if (!hasActiveJobs && installPoller !== null) {
		clearInterval(installPoller)
		installPoller = null
	}
}

/**
 * 发起安装任务
 * @param key 运行时 key (e.g. 'node', 'python')
 */
const doInstall = async (key: string) => {
	if (isInstalling(key)) return
	// 乐观更新：立即显示「安装中」状态
	installJobs.value[key] = {status: 'pending', log: '', job_id: null}
	logPanelKey.value = key
	try {
		const res = await ApiService.installRuntime(nodeId, key)
		const jobId = res.data.job_id
		installJobs.value[key] = {status: 'pending', log: '', job_id: jobId}
		// 启动轮询（如果还没启动）
		if (installPoller === null) {
			installPoller = window.setInterval(pollInstallJobs, 2000)
		}
	} catch (err: any) {
		console.error('Error starting install:', err)
		installJobs.value[key] = {
			status: 'failed',
			log: err?.response?.data?.error || '发起安装失败',
			job_id: null,
		}
	}
}

/**
 * 打开日志面板
 */
const openLog = (key: string) => {
	logPanelKey.value = key
	nextTick(() => {
		if (logPanelRef.value) {
			logPanelRef.value.scrollTop = logPanelRef.value.scrollHeight
		}
	})
}

// 组件生命周期
onMounted(async () => {
	await nextTick()
	await fetchNode()
	initCharts()
	await fetchResourceStatus()
	// 加载已存储的运行时列表（不会主动探测节点）
	await fetchCapabilities()
	poller = window.setInterval(fetchResourceStatus, 5000)
	window.addEventListener('resize', onWindowResize)
})

onUnmounted(() => {
	if (poller !== null) clearInterval(poller)
	if (installPoller !== null) clearInterval(installPoller)
	window.removeEventListener('resize', onWindowResize)
	cpuChart?.dispose()
	memChart?.dispose()
})
</script>
