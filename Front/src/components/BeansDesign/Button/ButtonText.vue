<!--
  - Copyright (c)  2025.3.7
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - ButtonText.vue
  - Last Modified: 2025-03-07 18:37:21  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<!-- ButtonText.vue -->
<template>
	<button
		:style="computedStyles"
		@blur="blurEffect"
		@focus="focusEffect"
		@mouseout="resetEffect"
		@mouseover="hoverEffect">
		文字按钮
	</button>
</template>

<script>
import colors from '@/config/colors';

export default {
	name: 'ButtonText',
	data() {
		return {
			colors,
		};
	},
	computed: {
		computedStyles() {
			return {
				color: this.getColor(),
				backgroundColor: this.getBackgroundColor(),
				borderRadius: this.getBorderRadius(),
				padding: this.getPadding(),
				fontSize: this.getFontSize(),
				fontWeight: this.getFontWeight(),
				border: this.getBorder(),
				transition: this.getTransition(),
			};
		},
	},
	methods: {
		getColor() {
			return this.colors.yellowGreen;
		},
		getBackgroundColor() {
			return 'transparent';
		},
		getBorderRadius() {
			return '8px';
		},
		getPadding() {
			return '4px 8px';
		},
		getFontSize() {
			return '14px';
		},
		getFontWeight() {
			return '500';
		},
		getBorder() {
			return 'none';
		},
		getTransition() {
			return 'all 0.2s';
		},
		hoverEffect(event) {
			event.target.style.color = this.colors.black;
			event.target.style.backgroundColor = this.colors.grayLight;
		},
		resetEffect(event) {
			event.target.style.color = this.colors.yellowGreen;
			event.target.style.backgroundColor = 'transparent';
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