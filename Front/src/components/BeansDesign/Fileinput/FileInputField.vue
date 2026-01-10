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
	<div class="relative">
		<input :accept="acceptTypes"
			   :style="inputStyles"
			   multiple
			   type="file"
			   @change="handleFileChange">
		<div :style="buttonStyles">
			{{ buttonText }}
		</div>
		<div v-if="selectedFiles.length" :style="infoStyles">
			<span v-if="selectedFiles.length === 1">已选文件: {{ selectedFiles[0] }}</span>
			<span v-else>已选 {{ selectedFiles.length }} 个文件</span>
		</div>
	</div>
</template>

<script>
import colors from '@/config/colors';

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
			colors,
			selectedFiles: []
		};
	},
	computed: {
		buttonText() {
			return this.acceptTypes ? `选择文件 (${this.acceptTypes})` : '选择文件';
		},
		inputStyles() {
			return {
				position: 'absolute',
				inset: '0',
				width: '100%',
				height: '100%',
				opacity: '0',
				cursor: 'pointer',
				border: `2px solid ${this.colors.yellowGreen}`,
				borderRadius: '8px',
			};
		},
		buttonStyles() {
			return {
				border: `2px solid ${this.colors.yellowGreen}`,
				borderRadius: '8px',
				padding: '8px 16px',
				fontSize: '14px',
				fontWeight: '500',
				textAlign: 'center',
				backgroundColor: this.colors.white,
				color: this.colors.yellowGreen,
				cursor: 'pointer',
				transition: 'all 0.2s',
			};
		},
		infoStyles() {
			return {
				marginTop: '8px',
				fontSize: '12px',
				color: this.colors.gray,
			};
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