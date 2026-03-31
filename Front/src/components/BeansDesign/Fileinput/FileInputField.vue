<!--
  - Copyright (c)  2025.3.7
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - FileInputField.vue
  - Last Modified: 2025-03-07 18:37:21  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<div class="relative beans-file-input">
		<input :accept="acceptTypes"
			   class="beans-file-input__native"
			   multiple
			   type="file"
			   @change="handleFileChange">
		<div class="beans-file-input__button">
			{{ buttonText }}
		</div>
		<div v-if="selectedFiles.length" class="beans-file-input__info">
			<span v-if="selectedFiles.length === 1">已选文件: {{ selectedFiles[0] }}</span>
			<span v-else>已选 {{ selectedFiles.length }} 个文件</span>
		</div>
	</div>
</template>

<script>
export default {
	name: 'FileInputField',
	props: {
		acceptTypes: {
			type: String,
			default: ''
		}
	},
	data() {
		return {
			selectedFiles: []
		};
	},
	computed: {
		buttonText() {
			return this.acceptTypes ? `选择文件 (${this.acceptTypes})` : '选择文件';
		},
	},
	methods: {
		handleFileChange(event) {
			const files = event.target.files;
			this.selectedFiles = files.length > 0 ? Array.from(files).map(file => file.name) : [];
		}
	}
};
</script>

<style scoped>
.beans-file-input__native {
	position: absolute;
	inset: 0;
	width: 100%;
	height: 100%;
	opacity: 0;
	cursor: pointer;
}

.beans-file-input__button {
	border: none;
	border-radius: 12px;
	padding: 0.75rem 1rem;
	font-size: 0.875rem;
	font-weight: 500;
	text-align: center;
	background: #f3f4f6;
	color: #1f2a37;
	transition: background-color 0.2s ease;
}

.beans-file-input:hover .beans-file-input__button,
.beans-file-input:focus-within .beans-file-input__button {
	background: #eef2f7;
}

.beans-file-input:focus-within .beans-file-input__button {
	outline: 2px solid #14b8a6;
	outline-offset: 1px;
	background: #f8fafc;
}

.beans-file-input__info {
	margin-top: 0.5rem;
	font-size: 0.75rem;
	color: #64748b;
}
</style>