<!--
  - Copyright (c)  2025.3.7
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - BadgeOutline.vue
  - Last Modified: 2025-03-07 18:05:43  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
  <span :style="badgeStyles">
    <slot></slot>
  </span>
</template>

<script>
import colors from '@/config/colors';

export default {
	name: 'BadgeOutline',
	data() {
		return {
			colors,
		};
	},
	computed: {
		badgeStyles() {
			return {
				border: this.getBorder(),
				color: this.getColor(),
				fontWeight: this.getFontWeight(),
				padding: this.getPadding(),
				borderRadius: this.getBorderRadius(),
				fontSize: this.getFontSize(),
				display: this.getDisplay(),
				minWidth: this.getMinWidth(),
				textAlign: this.getTextAlign(),
			};
		},
	},
	methods: {
		getBorder() {
			return `2px solid ${this.colors.yellowGreen}`;
		},
		getColor() {
			return this.colors.yellowGreen;
		},
		getFontWeight() {
			return '500';
		},
		getPadding() {
			return '4px 8px';
		},
		getBorderRadius() {
			return '9999px'; // Full rounded
		},
		getFontSize() {
			return '14px';
		},
		getDisplay() {
			return 'inline-block';
		},
		getMinWidth() {
			return '2rem';
		},
		getTextAlign() {
			return 'center';
		},
	},
};
</script>