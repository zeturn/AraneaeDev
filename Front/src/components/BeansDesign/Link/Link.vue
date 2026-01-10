<!--
  - Copyright (c)  2025.3.7
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Link.vue
  - Last Modified: 2025-03-07 18:37:21  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<a :href="href" :style="linkStyles" @blur="blurEffect" @focus="focusEffect" @mouseout="normalEffect"
	   @mouseover="hoverEffect">
		<slot>链接文本</slot>
	</a>
</template>

<script>
import colors from '@/config/colors';

export default {
	name: 'LinkComponent',
	props: {
		href: {
			type: String,
			required: false,
			default: '#',
		},
	},
	data() {
		return {
			colors,
			isHovered: false,
			isFocused: false,
		};
	},
	computed: {
		linkStyles() {
			return {
				color: this.getColor(),
				fontWeight: this.getFontWeight(),
				padding: this.getPadding(),
				borderRadius: this.getBorderRadius(),
				fontSize: this.getFontSize(),
				outline: this.getOutline(),
				backgroundColor: this.getBackgroundColor(),
				display: 'inline-block',
				width: 'auto', // Make the width auto to fit the text content
				transition: 'all 0.2s',
			};
		},
	},
	methods: {
		getColor() {
			return this.isHovered ? this.colors.darkGreen : this.colors.yellowGreen;
		},
		getFontWeight() {
			return '500';
		},
		getPadding() {
			return '4px 8px';
		},
		getBorderRadius() {
			return '8px';
		},
		getFontSize() {
			return '14px';
		},
		getOutline() {
			return this.isFocused ? `4px solid ${this.colors.mint}` : 'none';
		},
		getBackgroundColor() {
			return this.isHovered ? this.colors.grayLight : 'transparent';
		},
		hoverEffect() {
			this.isHovered = true;
		},
		normalEffect() {
			this.isHovered = false;
		},
		focusEffect() {
			this.isFocused = true;
		},
		blurEffect() {
			this.isFocused = false;
		},
	},
};
</script>