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
							{{ $t('项目名称') }}
						</label>
						<input
							id="name"
							v-model="projectData.name"
							class="field-input"
							:placeholder="$t('请输入项目名称')"
							required
							type="text"
						/>
					</div>

					<!-- 语言 -->
					<div>
						<label class="block mb-2 text-gray-700 text-sm font-medium" for="language">
							{{ $t('语言') }}
						</label>
						<el-select
							id="language"
							v-model="projectData.language"
							class="w-full"
							:placeholder="$t('请选择语言')"
						>
							<el-option label="Python" value="python" />
							<el-option label="JavaScript" value="js" />
							<el-option label="TypeScript" value="ts" />
							<el-option label="Go" value="go" />
							<el-option label="Java" value="java" />
							<el-option label="Shell" value="shell" />
						</el-select>
					</div>

				<!-- 描述（跨两列） -->
					<div class="col-span-1 md:col-span-2">
						<label class="block mb-2 text-gray-700 text-sm font-medium" for="description">
							{{ $t('描述') }}
						</label>
						<textarea
							id="description"
							v-model="projectData.description"
							class="field-input resize-none"
							:placeholder="$t('请输入项目描述')"
							rows="4"
						></textarea>
					</div>

					<!-- 提交按钮（跨两列） -->
					<div class="col-span-1 md:col-span-2">
						<button
							class="btn-primary w-full"
							type="submit"
						>
							{{ $t('创建项目') }}
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
			if (!this.projectData.language) {
				alert(this.$t("请选择语言。"));
				return;
			}

			// 调用 ApiService 创建项目
			// Call ApiService to create project
			try {
				const response = await ApiService.createProject(this.projectData);
				const newId = response.data.id;
				// 跳转到项目详情页
				// Navigate to project detail page

				EventBus.emit('notify', {
					type: 'success',
					title: this.$t('创建成功'),
					message: this.$t('项目已成功创建')
				});

				this.$router.push({name: 'project', params: {id: newId}});
			} catch (error) {
				console.error(this.$t("创建项目失败："), error);
				alert(this.$t("创建项目失败，请重试。"));
			}
		},
	},
};
</script>

