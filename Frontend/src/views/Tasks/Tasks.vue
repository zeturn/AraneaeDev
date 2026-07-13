<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Tasks.vue (Global Layout Container)
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<script lang="ts" setup>
import {computed} from 'vue';
import {useRoute} from 'vue-router';

const route = useRoute();
</script>

<template>
	<div>
		<div class="px-6 pt-6">
			<h1 class="text-3xl font-semibold text-gray-500">
				{{ $t('任务') }}
			</h1>
			<p class="text-gray-400 text-sm mt-1">
				{{ $t('任务是一次性的实时自动化操作。如需定时或链式操作，请使用计划。') }}
			</p>
		</div>
		<slot></slot>
	</div>
</template>