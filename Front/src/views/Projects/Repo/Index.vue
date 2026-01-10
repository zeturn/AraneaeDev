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
			<div class="container mx-auto">
				<ul class="border border-gray-300 rounded-md divide-y divide-gray-300">
					<li v-for="repo in repos" :key="repo.version_hash" class="p-3">
						<span class="font-medium">版本号:</span> {{ repo.version_hash }}
						<span class="ml-4 text-gray-600">发布日期:</span> {{ repo.release_date }}
					</li>
				</ul>
			</div>
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
			repos: []
		};
	},
	methods: {
		async fetchRepos() {
			try {
				const response = await ApiService.getReposFromProject(this.projectId);
				this.repos = response.data.versions;
			} catch (error) {
				console.error("获取仓库列表失败:", error);
			}
		}
	},
	created() {
		this.fetchRepos();
	}
};
</script>
