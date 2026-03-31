<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - ApronsAbout.vue
  - Last Modified: 2025-05-22 22:24:32  -  Davis, CA
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
			关于
		</h1>
		<p class="text-gray-500 text-sm m-4">
			Aprons 是一个开源的自动化工具，HD网络数据套件之一，旨在帮助用户更高效地获取数据以及实现数据源封装。
		</p>

		<h2 class="text-gray-500 text-2xl m-4">
			许可证
		</h2>
		<p class="text-green-400  m-4">
			Demo V0.0.1
		</p>

		<p class="text-gray-500 text-sm m-4">
			Demo 版本是一个演示版本，旨在展示 Aprons 的基本功能和交互设计。
			不包含所有功能。
		</p>

	</Aprons>
</template>


<style scoped>

</style>