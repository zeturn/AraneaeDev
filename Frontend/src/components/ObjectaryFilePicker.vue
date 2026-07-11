<!--
  - ObjectaryFilePicker.vue
  - Browses the current user's Objectary file directory (via Araneae backend +
  - BasaltPass cross-app token exchange) and lets the user import a file into a
  - project as a new artifact version.
  -->

<template>
	<div
		v-if="visible"
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
		@click.self="close"
	>
		<div class="bg-white rounded-lg shadow-xl w-full max-w-2xl max-h-[80vh] flex flex-col">
			<div class="flex items-center justify-between px-4 py-3 border-b">
				<h3 class="text-lg font-medium text-gray-800">从 Objectary 导入文件</h3>
				<button
					class="text-gray-400 hover:text-gray-600 text-2xl leading-none"
					@click="close"
				>
					&times;
				</button>
			</div>

			<!-- Breadcrumb -->
			<div class="flex items-center gap-1 px-4 py-2 text-sm text-gray-500 border-b bg-gray-50">
				<button
					class="hover:text-indigo-600"
					@click="goToRoot"
				>
					根目录
				</button>
				<span
					v-for="(crumb, idx) in breadcrumb"
					:key="crumb.id"
				>
					<span class="mx-1">/</span>
					<button
						class="hover:text-indigo-600"
						@click="goToCrumb(idx)"
					>
						{{ crumb.name }}
					</button>
				</span>
				<button
					v-if="breadcrumb.length"
					class="ml-2 text-indigo-600 hover:underline"
					@click="goUp"
				>
					返回上级
				</button>
			</div>

			<!-- File list -->
			<div class="flex-1 overflow-auto px-4 py-2">
				<p
					v-if="loading"
					class="text-gray-400 py-6 text-center"
				>
					加载中…
				</p>
				<p
					v-else-if="error"
					class="text-red-500 py-6 text-center"
				>
					{{ error }}
				</p>
				<p
					v-else-if="!items.length"
					class="text-gray-400 py-6 text-center"
				>
					此目录为空
				</p>
				<ul v-else>
					<li
						v-for="item in items"
						:key="item.nodeId"
						class="flex items-center justify-between px-2 py-2 rounded hover:bg-gray-50 cursor-pointer border-b border-gray-100"
						:class="selectedNode && selectedNode.nodeId === item.nodeId ? 'bg-indigo-50' : ''"
						@click="onRowClick(item)"
					>
						<div class="flex items-center gap-2 min-w-0">
							<span class="text-xl">{{ item.type === 'dir' ? '📁' : '📄' }}</span>
							<span class="truncate">{{ item.name }}</span>
							<span
								v-if="item.type === 'file' && item.size"
								class="text-xs text-gray-400"
							>
								{{ formatSize(item.size) }}
							</span>
						</div>
						<span
							v-if="item.type === 'file'"
							class="text-xs text-indigo-600"
						>
							{{ selectedNode && selectedNode.nodeId === item.nodeId ? '已选择' : '点击选择' }}
						</span>
					</li>
				</ul>
			</div>

			<!-- Footer -->
			<div class="flex items-center justify-between px-4 py-3 border-t">
				<span class="text-sm text-gray-400">
					{{ selectedNode ? '已选择：' + selectedNode.name : '请选择一个文件' }}
				</span>
				<div class="flex gap-2">
					<button
						class="px-3 py-1.5 rounded border text-gray-600 hover:bg-gray-50"
						@click="close"
					>
						取消
					</button>
					<button
						class="px-3 py-1.5 rounded bg-indigo-600 text-white disabled:opacity-50"
						:disabled="!selectedNode || importing"
						@click="doImport"
					>
						{{ importing ? '导入中…' : '导入到项目' }}
					</button>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import ApiService from '@/services/ApiService';

export default {
	name: 'ObjectaryFilePicker',
	props: {
		projectId: {
			type: String,
			required: true,
		},
		visible: {
			type: Boolean,
			default: false,
		},
	},
	emits: ['close', 'imported'],
	data() {
		return {
			items: [],
			breadcrumb: [], // [{ id, name }]
			currentParentId: 'root',
			loading: false,
			error: '',
			selectedNode: null,
			importing: false,
		};
	},
	watch: {
		visible(val) {
			if (val) {
				this.reset();
				this.loadNodes('root');
			}
		},
	},
	methods: {
		reset() {
			this.items = [];
			this.breadcrumb = [];
			this.currentParentId = 'root';
			this.selectedNode = null;
			this.error = '';
		},
		close() {
			this.$emit('close');
		},
		async loadNodes(parentId) {
			this.loading = true;
			this.error = '';
			this.selectedNode = null;
			try {
				const resp = await ApiService.listObjectaryNodes(parentId);
				const data = resp.data || {};
				this.items = Array.isArray(data.items) ? data.items : [];
				this.currentParentId = data.parentId || parentId;
			} catch (e) {
				this.error = e.response?.data?.error || '无法加载 Objectary 文件目录';
			} finally {
				this.loading = false;
			}
		},
		onRowClick(item) {
			if (item.type === 'dir') {
				this.breadcrumb.push({ id: item.nodeId, name: item.name });
				this.loadNodes(item.nodeId);
			} else {
				this.selectedNode = item;
			}
		},
		goToRoot() {
			this.breadcrumb = [];
			this.loadNodes('root');
		},
		goUp() {
			if (!this.breadcrumb.length) {
				return;
			}
			this.breadcrumb.pop();
			const parent = this.breadcrumb.length
				? this.breadcrumb[this.breadcrumb.length - 1].id
				: 'root';
			this.loadNodes(parent);
		},
		goToCrumb(idx) {
			this.breadcrumb = this.breadcrumb.slice(0, idx + 1);
			this.loadNodes(this.breadcrumb[idx].id);
		},
		async doImport() {
			if (!this.selectedNode) {
				return;
			}
			this.importing = true;
			this.error = '';
			try {
				const resp = await ApiService.importFromObjectary({
					project_id: this.projectId,
					node_id: this.selectedNode.nodeId,
					provider: this.selectedNode.provider || '',
				});
				this.$emit('imported', resp.data);
				this.close();
			} catch (e) {
				this.error = e.response?.data?.error || '导入失败';
			} finally {
				this.importing = false;
			}
		},
		formatSize(bytes) {
			if (!bytes) {
				return '';
			}
			const units = ['B', 'KB', 'MB', 'GB'];
			let size = bytes;
			let i = 0;
			while (size >= 1024 && i < units.length - 1) {
				size /= 1024;
				i += 1;
			}
			return `${size.toFixed(1)} ${units[i]}`;
		},
	},
};
</script>
