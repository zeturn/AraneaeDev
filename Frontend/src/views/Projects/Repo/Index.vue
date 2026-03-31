<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-05-22 21:04:50  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Project>
		<Repo>
			<section class="surface-panel repo-panel-flat space-y-4">
				<header v-if="notice" class="flex flex-wrap items-center justify-end gap-3">
					<span class="text-sm text-slate-500">{{ notice }}</span>
				</header>
				<div v-if="repos.length === 0" class="py-6 text-sm text-slate-500">暂无版本，请先上传构建产物。</div>
				<div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2">
					<article v-for="repo in repos" :key="repo.id || repo.version_hash" class="surface-card repo-card-flat space-y-3">
						<div class="space-y-1">
							<h3 class="text-base font-semibold text-slate-900">{{ repo.file_name || repo.version_hash }}</h3>
							<p class="text-xs text-slate-500">Version ID: {{ repo.id || repo.version_hash }}</p>
						</div>
						<div class="flex flex-wrap gap-2">
							<span class="tag-pill">hash: {{ repo.version_hash || '-' }}</span>
							<span class="tag-pill">发布时间: {{ formatDate(repo.release_date) }}</span>
						</div>
						<div class="flex flex-wrap gap-2 pt-1">
							<button class="btn-muted" @click="renameVersion(repo)">重命名</button>
							<router-link class="btn-muted" :to="`/aprons/projects/${projectId}/versions/${repo.id || repo.version_hash}/settings`">设置</router-link>
							<button class="btn-danger" @click="removeVersion(repo)">删除</button>
						</div>
					</article>
				</div>
			</section>
		</Repo>
	</Project>
</template>

<script>
import ApiService from '@/services/ApiService';
import Project from "@/views/Projects/Project.vue";
import Repo from "@/views/Projects/Repo/Repo.vue";

export default {
	components: {Repo, Project},
	data() {
		return {
			projectId: this.$route.params.id,
			repos: [],
			notice: '',
		};
	},
	methods: {
		formatDate(value) {
			if (!value) {
				return '-';
			}
			return new Date(value).toLocaleString();
		},
		async fetchRepos() {
			try {
				const response = await ApiService.getReposFromProject(this.projectId);
				this.repos = response.data.versions;
				this.notice = '';
			} catch (error) {
				console.error("获取仓库列表失败:", error);
				this.notice = '获取版本列表失败';
			}
		},
		async renameVersion(repo) {
			const currentName = repo.file_name || repo.version_hash || '';
			const nextName = window.prompt('输入新的版本名称（file_name）', currentName);
			if (nextName === null) {
				return;
			}
			const name = nextName.trim();
			if (!name) {
				this.notice = '版本名称不能为空';
				return;
			}
			try {
				await ApiService.updateProjectVersion(this.projectId, repo.id || repo.version_hash, {file_name: name});
				this.notice = '版本名称已更新';
				await this.fetchRepos();
			} catch (error) {
				console.error('rename version failed:', error);
				this.notice = error?.response?.data?.detail || '更新版本失败';
			}
		},
		async removeVersion(repo) {
			if (!window.confirm(`确认删除版本 ${repo.file_name || repo.version_hash} ?`)) {
				return;
			}
			try {
				await ApiService.deleteProjectVersion(this.projectId, repo.id || repo.version_hash);
				this.notice = '版本已删除';
				await this.fetchRepos();
			} catch (error) {
				console.error('delete version failed:', error);
				this.notice = error?.response?.data?.detail || '删除版本失败';
			}
		},
	},
	created() {
		this.fetchRepos();
	}
};
</script>

<style scoped>
.repo-panel-flat {
	box-shadow: none;
	margin: 0;
	padding: 0;
}

.repo-card-flat {
	background-color: #f3f4f6;
	box-shadow: none;
}
</style>
