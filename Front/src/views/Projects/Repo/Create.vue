<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Create.vue
  - Last Modified: 2025-05-24 14:56:54  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Project>
		<Repo>
			<div class="max-w-lg mx-auto p-4">
				<div
					:class="isDragging ? 'border-indigo-500 bg-indigo-50' : 'border-gray-300 bg-gray-50 hover:bg-gray-100'"
					class="border-2 border-dashed rounded-lg h-64 flex flex-col items-center justify-center cursor-pointer transition
                    bg-gray-50 hover:bg-gray-100"
					@click="triggerFileInput"
					@dragover.prevent="handleDragOver"
					@dragleave.prevent="handleDragLeave"
					@drop.prevent="handleDrop"
				>
					<input
						ref="fileInput"
						accept=".zip"
						class="hidden"
						type="file"
						@change="handleFileChange"
					/>
					<div class="text-gray-400 text-6xl">+</div>
					<p class="mt-2 text-gray-600">拖拽文件到此处或点击上传</p>
				</div>
				<p v-if="selectedFile" class="mt-4 text-gray-700">
					已选择文件：<span class="font-medium">{{ selectedFile.name }}</span>
				</p>
				<button
					class="btn-primary mt-4 w-full"
					@click="uploadFile"
				>
					上传
				</button>
				<p
					v-if="message"
					:class="messageType === 'success' ? 'text-green-500' : 'text-red-500'"
					class="mt-2"
				>
					{{ message }}
				</p>
			</div>
		</Repo>
	</Project>
</template>

<script>
import ApiService from '@/services/ApiService';
import Project from "@/views/Projects/Project.vue";
import Repo from "@/views/Projects/Repo/Repo.vue";
import EventBus from '@/utils/event-bus'

/**
 * 上传脚本文件组件（带成功/错误状态提示）
 * Upload script file component with success/error status indication
 */
export default {
	name: 'RepoUpload',
	components: {Repo, Project},
	data() {
		return {
			selectedFile: null,
			message: '',
			messageType: '',      // 'success' or 'error'
			projectId: this.getProjectIdFromURL(),
			isDragging: false,
		};
	},
	methods: {
		/**
		 * 获取当前项目ID
		 * Get current project ID
		 */
		getProjectIdFromURL() {
			return this.$route.params.id;
		},

		/**
		 * 处理拖拽悬停
		 * Handle drag over event
		 */
		handleDragOver() {
			this.isDragging = true;
		},

		/**
		 * 处理拖拽离开
		 * Handle drag leave event
		 */
		handleDragLeave() {
			this.isDragging = false;
		},

		/**
		 * 处理文件放下
		 * Handle file drop event
		 */
		handleDrop(event) {
			this.isDragging = false;
			const files = event.dataTransfer.files;
			if (files.length) {
				this.selectedFile = files[0];
			}
		},

		/**
		 * 触发文件输入点击
		 * Trigger file input click
		 */
		triggerFileInput() {
			this.$refs.fileInput.click();
		},

		/**
		 * 处理文件选择
		 * Handle file selection
		 */
		handleFileChange(event) {
			this.selectedFile = event.target.files[0];
		},

		/**
		 * 上传文件并提示状态颜色
		 * Upload the selected file and show colored status message
		 */
		async uploadFile() {
			if (!this.selectedFile) {
				this.messageType = 'error';
				this.message = '请选择一个文件！';
				return;
			}
			const formData = new FormData();
			formData.append('project_id', this.projectId);
			formData.append('file', this.selectedFile);

			try {
				const response = await ApiService.uploadCode(formData);
				this.messageType = 'success';
				this.message = response.data.message || '文件上传成功！';
				EventBus.emit('notify', {
					type: 'success',
					title: '创建成功',
					message: '新版本已成功创建'
				});
			} catch (error) {
				this.messageType = 'error';
				this.message = error.response?.data?.error || '文件上传失败';
			}
		},
	},
};
</script>
