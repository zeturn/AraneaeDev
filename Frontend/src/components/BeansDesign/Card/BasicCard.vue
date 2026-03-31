<!--
  - Copyright (c)  2025.5.3
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - BasicCard.vue
  - Last Modified: 2025-05-03 16:08:20  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<div :style="{ borderColor: borderColor || colors.yellowGreen }" class="card">
		<h2 :style="{ color: titleColor || 'gray-900' }" class="card-title">{{ title }}</h2>
		<p class="card-description">{{ description }}</p>
		<div class="card-content">
			<slot></slot>
		</div>
	</div>
</template>

<script>
import colors from '@/config/colors';

export default {
	name: 'BasicCard',
	props: {
		title: {
			type: String,
			required: true
		},
		description: {
			type: String,
			required: false,
			default: ''
		},
		borderColor: {
			type: String,
			required: false,
			default: ''
		},
		titleColor: {
			type: String,
			required: false,
			default: ''
		}
	},
	data() {
		return {colors};
	}
};
</script>

<style scoped>
.card {
	background-color: white;
	border-radius: 8px;
	padding: 1rem;
	border-width: 2px;
	border-style: solid;
}

.card-title {
	font-size: 1.125rem;
	font-weight: 600;
	color: #1f2937;
}

.card-description {
	font-size: 0.875rem;
	color: #6b7280;
	margin-top: 0.25rem;
}

.card-content {
	margin-top: 1rem;
}
</style>
