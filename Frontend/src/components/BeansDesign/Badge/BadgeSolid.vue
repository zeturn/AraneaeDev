<!--
  - Copyright (c)  2025.3.7
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - BadgeSolid.vue
  - Last Modified: 2025-03-07 18:05:48  -  Davis, CA
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
	name: 'BadgeSolid',
	data() {
		return {
			colors,
		};
	},
	computed: {
		badgeStyles() {
			return {
				backgroundColor: this.getBackgroundColor(),
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
		getBackgroundColor() {
			return this.colors.yellowGreen;
		},
		getColor() {
			return this.colors.white;
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