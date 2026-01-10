<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-05-19 21:11:54  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<script setup lang="ts">
import {ref, onMounted, computed} from 'vue';
import {useRoute} from 'vue-router';
import ApiService from '@/services/ApiService';
import Schedules from "@/views/Schedules/Schedules.vue";

</script>

<template>
	<Schedules>
		<h1 class="text-2xl font-bold">编辑日程</h1>

		<!-- 这里是编辑日程的内容 -->
		<div class="p-4">
			<!-- 你可以在这里添加表单或其他组件 -->
			<p>编辑日程的内容...</p>
		</div>
	</Schedules>
</template>

<style scoped>

</style>