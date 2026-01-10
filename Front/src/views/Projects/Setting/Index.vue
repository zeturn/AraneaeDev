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
		<div>
			<a>
				项目设置
			</a>
			<button class="delete-button" @click="deleteProject">
				删除项目
			</button>
		</div>
	</Project>
</template>

<script>
import {ref, onMounted} from "vue";
import ApiService from "@/services/ApiService.js"; // 引入ApiService
import Project from "@/views/Projects/Project.vue";

export default {
	components: {Project},
	data() {
		return {
			id: this.$route.params.id,
			name: null,
			description: null,
			language: null,
			command: null,
			mode: null,
			created_at: null,
			edited_at: null,
			owners: [],
			editors: []
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
					this.id = response.data.id;
					this.name = response.data.name;
					this.description = response.data.description;
					this.language = response.data.language;
					this.command = response.data.command;
					this.mode = response.data.mode;
					this.created_at = response.data.created_at;
					this.edited_at = response.data.updated_at;
				})
				.catch(error => {
					console.error("Error fetching project data:", error);
				});
		},
		deleteProject() {
			if (confirm("确定要删除该项目吗？此操作不可撤销！")) {
				ApiService.deleteProject(this.id)
					.then(() => {
						alert("项目已成功删除！");
						this.$router.push("/aprons/projects"); // 删除后跳转到项目列表
					})
					.catch(error => {
						console.error("删除项目时出错:", error);
						alert("删除项目失败，请重试！");
					});
			}
		},
		formatDate(dateString) {
			const options = {year: "numeric", month: "long", day: "numeric", hour: "numeric", minute: "numeric"};
			return new Date(dateString).toLocaleDateString(undefined, options);
		}
	}
};
</script>

<style scoped>
.delete-button {
	margin-top: 10px;
	padding: 8px 12px;
	background-color: red;
	color: white;
	border: none;
	border-radius: 4px;
	cursor: pointer;
}

.delete-button:hover {
	background-color: darkred;
}
</style>
