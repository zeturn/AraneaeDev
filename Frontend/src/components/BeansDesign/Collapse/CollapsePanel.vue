<!--
  - Copyright (c)  2025.3.7
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - CollapsePanel.vue
  - Last Modified: 2025-03-07 18:37:21  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<div :style="panelStyles">
		<div :style="[headerStyles, isOpen && headerOpenStyles]" @click="toggle">
			{{ title }}
		</div>
		<div v-show="isOpen" :style="contentStyles">
			<slot></slot>
		</div>
	</div>
</template>

<script>
import colors from '@/config/colors';

export default {
	name: 'CollapsePanel',
	props: {
		title: {
			type: String,
			required: true,
		},
	},
	data() {
		return {
			colors,
			isOpen: false,
		};
	},
	computed: {
		panelStyles() {
			return {
				width: '100%',
			};
		},
		headerStyles() {
			return {
				cursor: 'pointer',
				backgroundColor: this.colors.yellowGreen,
				color: this.colors.white,
				fontWeight: '500',
				padding: '8px 16px',
				borderRadius: this.isOpen ? '8px 8px 0 0' : '8px',
				transition: 'border-radius 0.2s',
			};
		},
		headerOpenStyles() {
			return {
				borderBottomLeftRadius: '0',
				borderBottomRightRadius: '0',
			};
		},
		contentStyles() {
			return {
				border: `2px solid ${this.colors.yellowGreen}`,
				padding: '8px 16px',
				borderRadius: '0 0 8px 8px',
				overflow: 'hidden',
				transition: 'all 0.3s',
			};
		},
	},
	methods: {
		toggle() {
			this.isOpen = !this.isOpen;
		},
	},
};
</script>