<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Create.vue
  - Last Modified: 2025-05-21 20:45:43  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->
# === 以下功能：项目创建表单组件 ===
<template>
	<Workplace>
		<Project>
			<div class="container max-w-4xl mx-auto p-6 bg-white rounded-2xl my-8 overflow-x-hidden overflow-y-auto">
				<form class="grid grid-cols-1 md:grid-cols-2 gap-6" @submit.prevent="submitProject">
					<!-- 项目名称 -->
					<div>
						<label class="block mb-2 text-gray-700 text-sm font-medium" for="name">
							项目名称
						</label>
						<input
							id="name"
							v-model="projectData.name"
							class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
							placeholder="请输入项目名称"
							required
							type="text"
						/>
					</div>

					<!-- 语言 -->
					<div>
						<label class="block mb-2 text-gray-700 text-sm font-medium" for="language">
							语言
						</label>
						<select
							id="language"
							v-model="projectData.language"
							class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
							required
						>
							<option disabled value="">请选择语言</option>
							<option value="python">Python</option>
							<option value="js">JavaScript</option>
							<option value="ts">TypeScript</option>
						</select>
					</div>

					<!-- 命令 -->
					<div>
						<label class="block mb-2 text-gray-700 text-sm font-medium" for="command">
							命令
						</label>
						<input
							id="command"
							v-model="projectData.command"
							class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400"
							placeholder="请输入项目命令"
							required
							type="text"
						/>
					</div>

					<!-- 描述（跨两列） -->
					<div class="col-span-1 md:col-span-2">
						<label class="block mb-2 text-gray-700 text-sm font-medium" for="description">
							描述
						</label>
						<textarea
							id="description"
							v-model="projectData.description"
							class="w-full p-3 bg-gray-100 rounded-lg focus:ring-4 focus:ring-blue-400 focus:border-blue-400 resize-none"
							placeholder="请输入项目描述"
							rows="4"
						></textarea>
					</div>

					<!-- 提交按钮（跨两列） -->
					<div class="col-span-1 md:col-span-2">
						<button
							class="w-full py-3 bg-gray-800 text-white rounded-lg hover:bg-gray-900 transition-colors font-medium"
							type="submit"
						>
							创建项目
						</button>
					</div>
				</form>
			</div>
		</Project>
	</Workplace>
</template>


<script>
import ApiService from "@/services/ApiService.js";
import Workplace from "@/views/Workplaces/Workplace.vue";
import Project from "@/views/Workplaces/Projects/Project.vue";
import EventBus from '@/utils/event-bus'

/**
 * 项目创建表单组件
 * Project creation form component
 */
export default {
	name: "ProjectCreationForm",
	components: {Workplace, Project},
	data() {
		return {
			projectData: {
				name: "",
				description: "",
				language: "",
				command: "",
				workplace: null,
			},
		};
	},
	created() {
		this.projectData.workplace = this.$route.params.id;
	},
	methods: {
		/**
		 * 提交项目数据并跳转到项目详情页
		 * Submit project data and navigate to project detail page
		 */
		async submitProject() {
			// 调用 ApiService 创建项目
			// Call ApiService to create project
			try {
				const response = await ApiService.createProject(this.projectData);
				const newId = response.data.id;
				// 跳转到项目详情页
				// Navigate to project detail page

				EventBus.emit('notify', {
					type: 'success',
					title: '创建成功',
					message: '项目已成功创建'
				});

				this.$router.push({name: 'project', params: {id: newId}});
			} catch (error) {
				console.error("创建项目失败：", error);
				alert("创建项目失败，请重试。");
			}
		},
	},
};
</script>

