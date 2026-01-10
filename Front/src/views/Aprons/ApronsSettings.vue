<!--
  - Copyright (c)  2025.5.3
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - ApronsSettings.vue
  - Last Modified: 2025-05-03 14:41:45  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->


<script lang="ts" setup>
import {ref, onMounted} from 'vue';
import ApiService from '@/services/ApiService.js';
import Aprons from "@/views/Aprons/Aprons.vue";
</script>

<template>
	<Aprons>
		<h1 class="text-gray-500 text-3xl m-4">
			设置
		</h1>
		<p class="text-gray-500 text-sm m-4">
			Aprons 是一个开源的自动化工具，旨在帮助用户更高效地管理和执行任务。
		</p>

	</Aprons>
</template>


<style scoped>

</style>