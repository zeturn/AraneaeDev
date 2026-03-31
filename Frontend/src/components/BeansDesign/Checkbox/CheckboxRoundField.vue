<!--
  - Copyright (c)  2025.3.7
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - CheckboxRoundField.vue
  - Last Modified: 2025-03-07 18:37:21  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<label :style="labelStyles">
		<input :style="inputStyles" type="checkbox" @change="toggleCheck">
		<div :style="checkboxStyles">
			<div :style="innerCircleStyles"></div>
		</div>
		<span :style="labelTextStyles">Checkbox</span>
	</label>
</template>

<script>
import colors from '@/config/colors';

export default {
	name: 'CheckboxRoundField',
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
				borderRadius: '50%',
				display: 'flex',
				alignItems: 'center',
				justifyContent: 'center',
				transition: 'all 0.2s',
				position: 'relative',
			};
		},
		innerCircleStyles() {
			return {
				width: '12px',
				height: '12px',
				backgroundColor: this.colors.white,
				borderRadius: '50%',
				position: 'absolute',
				opacity: this.isChecked ? '1' : '0',
				transition: 'opacity 0.2s',
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
</script>s
