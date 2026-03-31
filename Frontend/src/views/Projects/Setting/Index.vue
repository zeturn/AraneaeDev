<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-05-21 20:42:56  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Project>
		<div class="mx-auto max-w-3xl px-4 pb-10">
			<div class="surface-panel space-y-6">
				<header class="space-y-2">
					<p class="text-xs uppercase tracking-wider text-slate-500">Project Settings</p>
					<h1 class="text-2xl font-semibold text-slate-900">{{ form.name || '项目设置' }}</h1>
					<p class="text-sm text-slate-500">支持项目重命名、修改元信息与删除操作。</p>
				</header>

				<div class="grid gap-4 md:grid-cols-2">
					<div class="md:col-span-2">
						<label class="mb-2 block text-sm font-medium text-slate-700">项目名称</label>
						<input v-model="form.name" type="text" class="field-input" placeholder="输入项目名称" />
					</div>
					<div>
						<label class="mb-2 block text-sm font-medium text-slate-700">语言</label>
						<input v-model="form.language" type="text" class="field-input" placeholder="python / go / node" />
					</div>
					<div>
						<label class="mb-2 block text-sm font-medium text-slate-700">默认命令</label>
						<input v-model="form.command" type="text" class="field-input" placeholder="例如: python app.py" />
					</div>
					<div class="md:col-span-2">
						<label class="mb-2 block text-sm font-medium text-slate-700">描述</label>
						<textarea v-model="form.description" rows="4" class="field-input" placeholder="项目描述"></textarea>
					</div>
				</div>

				<div class="flex flex-wrap items-center gap-3">
					<button class="btn-primary" :disabled="loading" @click="saveProject">
						{{ loading ? '保存中...' : '保存设置' }}
					</button>
					<router-link :to="`/aprons/projects/${projectId}/repo`" class="btn-muted">版本管理</router-link>
					<button class="btn-danger" :disabled="loading" @click="deleteProject">删除项目</button>
					<span class="text-sm text-slate-500">{{ notice }}</span>
				</div>

				<div class="grid gap-3 text-sm text-slate-500 md:grid-cols-2">
					<p>创建时间: {{ formatDate(form.created_at) }}</p>
					<p>更新时间: {{ formatDate(form.updated_at) }}</p>
				</div>
			</div>
		</div>
	</Project>
</template>

<script>
import ApiService from "@/services/ApiService.js"; // 引入ApiService
import Project from "@/views/Projects/Project.vue";

export default {
	components: {Project},
	data() {
		return {
			projectId: this.$route.params.id,
			loading: false,
			notice: '',
			form: {
				id: '',
				name: '',
				description: '',
				language: '',
				command: '',
				created_at: '',
				updated_at: '',
			},
		};
	},
	created() {
		this.fetchProject();
	},
	methods: {
		getProjectIdFromURL() {
			return this.$route.params.id;
		},
		fetchProject() {
			const projectId = this.getProjectIdFromURL();
			ApiService.getProject(projectId)
				.then(response => {
					this.form.id = response?.data?.id || projectId;
					this.form.name = response?.data?.name || '';
					this.form.description = response?.data?.description || '';
					this.form.language = response?.data?.language || '';
					this.form.command = response?.data?.command || '';
					this.form.created_at = response?.data?.created_at || '';
					this.form.updated_at = response?.data?.updated_at || '';
					this.notice = '';
				})
				.catch(error => {
					console.error("Error fetching project data:", error);
					this.notice = '加载项目信息失败';
				});
		},
		saveProject() {
			const name = String(this.form.name || '').trim();
			if (!name) {
				this.notice = '项目名称不能为空';
				return;
			}
			this.loading = true;
			this.notice = '';
			ApiService.updateProject(this.projectId, {
				name,
				description: this.form.description,
				language: this.form.language,
				command: this.form.command,
			})
				.then(() => {
					this.notice = '项目设置已保存';
					this.form.updated_at = new Date().toISOString();
				})
				.catch(error => {
					console.error('save project failed:', error);
					this.notice = error?.response?.data?.detail || '保存失败';
				})
				.finally(() => {
					this.loading = false;
				});
		},
		deleteProject() {
			if (!window.confirm('确认删除该项目？此操作不可撤销。')) {
				return;
			}
			this.loading = true;
			this.notice = '';
			ApiService.deleteProject(this.projectId)
				.then(() => {
					this.$router.push('/aprons/projects');
				})
				.catch(error => {
					console.error('delete project failed:', error);
					this.notice = error?.response?.data?.detail || '删除失败';
				})
				.finally(() => {
					this.loading = false;
				});
		},
		formatDate(dateString) {
			if (!dateString) {
				return '-';
			}
			const options = {year: "numeric", month: "long", day: "numeric", hour: "numeric", minute: "numeric"};
			return new Date(dateString).toLocaleDateString(undefined, options);
		}
	}
};
</script>
