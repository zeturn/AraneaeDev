<!--
  - Copyright (c)  2025.3.7
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - CheckboxSquareField.vue
  - Last Modified: 2025-03-07 18:37:21  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<label :style="labelStyles">
		<input :style="inputStyles" type="checkbox" @change="toggleCheck">
		<div :style="checkboxStyles">
			<svg :style="svgStyles" class="size-6" fill="currentColor" viewBox="0 0 24 24"
				 xmlns="http://www.w3.org/2000/svg">
				<path clip-rule="evenodd"
					  d="M19.916 4.626a.75.75 0 0 1 .208 1.04l-9 13.5a.75.75 0 0 1-1.154.114l-6-6a.75.75 0 0 1 1.06-1.06l5.353 5.353 8.493-12.74a.75.75 0 0 1 1.04-.207Z"
					  fill-rule="evenodd"/>
			</svg>
		</div>
		<span :style="labelTextStyles">Checkbox</span>
	</label>
</template>

<script>
import colors from '@/config/colors';

export default {
	name: 'CheckboxSquareField',
	data() {
		return {
			colors,
			isChecked: false,
		};
	},
	computed: {
		labelStyles() {
			return {
				display: 'inline-flex',
				alignItems: 'center',
				cursor: 'pointer',
			};
		},
		inputStyles() {
			return {
				position: 'absolute',
				opacity: '0',
				width: '0',
				height: '0',
			};
		},
		checkboxStyles() {
			return {
				width: '24px',
				height: '24px',
				border: `2px solid ${this.colors.yellowGreen}`,
				backgroundColor: this.isChecked ? this.colors.yellowGreen : 'transparent',
				borderRadius: '4px',
				display: 'flex',
				alignItems: 'center',
				justifyContent: 'center',
				transition: 'all 0.2s',
			};
		},
		svgStyles() {
			return {
				width: '16px',
				height: '16px',
				color: this.colors.white,
				display: this.isChecked ? 'block' : 'none',
				transform: this.isChecked ? 'scale(1)' : 'scale(0)',
				transition: 'transform 0.2s',
			};
		},
		labelTextStyles() {
			return {
				marginLeft: '8px',
				fontSize: '14px',
				fontWeight: '500',
			};
		},
	},
	methods: {
		toggleCheck(event) {
			this.isChecked = event.target.checked;
		},
	},
};
</script>