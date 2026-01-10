<!--
  - Copyright (c)  2025.3.7
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - InputField.vue
  - Last Modified: 2025-03-07 18:37:21  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<input
		:style="computedStyles"
		type="text"
		@blur="blurEffect"
		@focus="focusEffect">
</template>

<script>
import colors from '@/config/colors';

export default {
	name: 'InputField',
	data() {
		return {
			colors,
		};
	},
	computed: {
		computedStyles() {
			return {
				height: this.getHeight(),
				border: this.getBorder(),
				borderRadius: this.getBorderRadius(),
				padding: this.getPadding(),
				fontSize: this.getFontSize(),
				fontWeight: this.getFontWeight(),
				outline: this.getOutline(),
				transition: this.getTransition(),
			};
		},
	},
	methods: {
		getHeight() {
			return '40px';
		},
		getBorder() {
			return `2px solid ${this.colors.yellowGreen}`;
		},
		getBorderRadius() {
			return '8px';
		},
		getPadding() {
			return '8px 16px';
		},
		getFontSize() {
			return '14px';
		},
		getFontWeight() {
			return '500';
		},
		getOutline() {
			return 'none';
		},
		getTransition() {
			return 'all 0.2s';
		},
		focusEffect(event) {
			event.target.style.outline = `4px solid ${this.colors.mint}`;
		},
		blurEffect(event) {
			event.target.style.outline = 'none';
		},
	},
};
</script>