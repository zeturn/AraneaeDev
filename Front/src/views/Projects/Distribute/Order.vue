<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Order.vue
  - Last Modified: 2025-05-22 21:02:45  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Project>
		<Distribute>
			<div class="container">
				<div class="p-6 bg-white rounded-2xl my-4">
				<div class="grid grid-cols-1 gap-6">
					<!-- Project ID（只读） -->
					<div class="mb-2">
						<label class="block mb-2 text-gray-700 text-sm font-medium" for="project_id">Project ID</label>
						<input
							id="project_id"
							v-model="project_id"
							class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
							readonly
							type="text"
						/>
					</div>
					<!-- Version 版本选择 -->
					<div class="mb-2">
						<label class="block mb-2 text-gray-700 text-sm font-medium" for="version">Version</label>
						<select
							id="version"
							v-model="version"
							class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4  focus:border-blue-400 focus:ring-indigo-500"
						>
							<option v-for="ver in versions" :key="ver.version_hash" :value="ver.version_hash">
								{{ ver.version_hash }} - {{ formatDate(ver.release_date) }}
							</option>
						</select>
					</div>
					<!-- Select Nodes 多选节点 -->
					<div class="mb-2">
						<label class="block mb-2 text-gray-700 text-sm font-medium" for="targets">Select Nodes</label>
						<select
							id="targets"
							v-model="selectedTargets"
							class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
							multiple
						>
							<option v-for="node in nodes" :key="node.id" :value="node.id">
								{{ node.name }}
							</option>
						</select>
					</div>
				</div>
				<button
					class="w-full py-3 ring-green-400 text-green-600 rounded-lg hover:bg-green-200 transition-colors font-medium disabled:opacity-50"
					@click="distributeSource"
				>
					Distribute
				</button>
				<p
					v-if="message"
					:class="messageType === 'success' ? 'text-green-600' : 'text-red-600'"
					class="mt-4 text-sm font-medium"
				>
					{{ message }}
				</p>
				</div>
			</div>
		</Distribute>
	</Project>
</template>

<script>
import ApiService from '@/services/ApiService';
import Distribute from "@/views/Projects/Distribute/Distribute.vue";
import Project from "@/views/Projects/Project.vue";

/**
 * 分发源视图组件
 * Distribute source view component
 */
export default {
	name: 'DistributeSourceView',
	components: {Project, Distribute},
	data() {
		return {
			nodes: [],
			selectedTargets: [],
			project_id: this.$route.params.id,
			versions: [],
			version: '',
			message: '',
			messageType: '' // 'success' or 'error'
		};
	},
	mounted() {
		this.fetchNodes();
		this.fetchVersions();
	},
	methods: {
		/**
		 * 获取节点列表
		 * Fetch list of nodes
		 */
		async fetchNodes() {
			try {
				const response = await ApiService.getNodesList();
				if (response.data && Array.isArray(response.data.results)) {
					this.nodes = response.data.results.map(node => ({
						id: node.id,
						name: node.name
					}));
				}
			} catch (error) {
				console.error('Error fetching nodes:', error);
			}
		},

		/**
		 * 获取项目版本
		 * Fetch versions for the given project
		 */
		async fetchVersions() {
			if (!this.project_id) return;
			try {
				const response = await ApiService.getVersionsFromProject(this.project_id);
				if (response.data && Array.isArray(response.data.versions)) {
					this.versions = response.data.versions;
				}
			} catch (error) {
				console.error('Error fetching versions:', error);
			}
		},

		/**
		 * 格式化日期
		 * Format date string to locale date
		 */
		formatDate(dateStr) {
			return new Date(dateStr).toLocaleDateString();
		},

		/**
		 * 分发源代码
		 * Distribute source code to selected nodes
		 */
		async distributeSource() {
			const targets = this.selectedTargets.map(id => ({node_id: id}));
			const payload = {
				project_id: this.project_id,
				version: this.version,
				targets
			};
			try {
				const response = await ApiService.orderSourceDistribution(payload);
				this.messageType = 'success';
				this.message = response.data.message || 'Source distributed successfully!';
			} catch (error) {
				this.messageType = 'error';
				this.message = error.response?.data?.error || 'Error distributing source';
				console.error('Error distributing source:', error);
			}
		}
	}
};
</script>

<style scoped>
/* 统一移除默认样式，所有视觉效果由 Tailwind CSS 管理 */
</style>
