<!--
  - Copyright (c)  2025.3.7
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - ToggleSwitch.vue
  - Last Modified: 2025-03-07 18:37:21  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<label :style="labelStyles">
		<input :style="inputStyles" type="checkbox" @change="toggleChange">
		<div :style="toggleStyles">
			<div :style="circleStyles"></div>
		</div>
	</label>
</template>

<script>
import colors from '@/config/colors';

export default {
	name: 'ToggleSwitch',
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
				position: 'relative',
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
		toggleStyles() {
			return {
				width: '48px',
				height: '28px',
				backgroundColor: this.isChecked ? this.colors.yellowGreen : this.colors.grayLight,
				borderRadius: '14px',
				position: 'relative',
				transition: 'background-color 0.2s',
			};
		},
		circleStyles() {
			return {
				content: '""',
				position: 'absolute',
				top: '2px',
				left: this.isChecked ? '22px' : '2px',
				width: '24px',
				height: '24px',
				backgroundColor: this.colors.white,
				border: `2px solid ${this.colors.grayLight}`,
				borderRadius: '50%',
				transition: 'left 0.2s',
			};
		},
	},
	methods: {
		toggleChange(event) {
			this.isChecked = event.target.checked;
		},
	},
};
</script>