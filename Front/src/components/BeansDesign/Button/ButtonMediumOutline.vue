<!--
  - Copyright (c)  2025.3.7
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - ButtonMediumOutline.vue
  - Last Modified: 2025-03-07 18:37:21  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<!-- ButtonMediumOutline.vue -->
<template>
	<button
		:style="computedStyles"
		@blur="blurEffect"
		@focus="focusEffect"
		@mouseout="resetEffect"
		@mouseover="hoverEffect">
		中按钮
	</button>
</template>

<script>
import colors from '@/config/colors';

export default {
	name: 'ButtonMediumOutline',
	data() {
		return {
			colors,
		};
	},
	computed: {
		computedStyles() {
			return {
				width: this.getWidth(),
				height: this.getHeight(),
				border: this.getBorder(),
				color: this.getColor(),
				backgroundColor: this.getBackgroundColor(),
				borderRadius: this.getBorderRadius(),
				padding: this.getPadding(),
				transition: this.getTransition(),
				fontSize: this.getFontSize(),
			};
		},
	},
	methods: {
		getWidth() {
			return '96px';
		},
		getHeight() {
			return '48px';
		},
		getBorder() {
			return `2px solid ${this.colors.yellowGreen}`;
		},
		getColor() {
			return this.colors.yellowGreen;
		},
		getBackgroundColor() {
			return this.colors.white;
		},
		getBorderRadius() {
			return '8px';
		},
		getPadding() {
			return '8px 16px';
		},
		getTransition() {
			return 'all 0.2s';
		},
		getFontSize() {
			return '14px';
		},
		hoverEffect(event) {
			event.target.style.backgroundColor = this.colors.yellowGreen;
			event.target.style.color = this.colors.white;
		},
		resetEffect(event) {
			event.target.style.backgroundColor = this.colors.white;
			event.target.style.color = this.colors.yellowGreen;
		},
		focusEffect(event) {
			event.target.style.outline = `4px solid ${this.colors.grayLight}`;
		},
		blurEffect(event) {
			event.target.style.outline = 'none';
		},
	},
};
</script>