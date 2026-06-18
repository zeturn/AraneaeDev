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

const loading = ref(false);
const version = ref('loading...');
const shortCommit = ref('');
const buildTime = ref('');
const loadError = ref('');

const getErrorMessage = (err: unknown): string => {
	if (err && typeof err === 'object') {
		const maybeAxiosError = err as { response?: { data?: { message?: unknown } }; message?: unknown };
		const responseMessage = maybeAxiosError.response?.data?.message;
		if (typeof responseMessage === 'string') {
			return responseMessage;
		}
		if (typeof maybeAxiosError.message === 'string') {
			return maybeAxiosError.message;
		}
	}
	return '无法获取版本信息';
};

const loadVersionInfo = async () => {
	loading.value = true;
	loadError.value = '';
	try {
		const response = await ApiService.getSystemInfo();
		const payload = response?.data || {};
		version.value = payload?.version || 'unknown';
		shortCommit.value = payload?.git?.short_commit || '';
		buildTime.value = payload?.build_time || '';
	} catch (err: unknown) {
		version.value = 'unknown';
		loadError.value = getErrorMessage(err);
	} finally {
		loading.value = false;
	}
};

onMounted(() => {
	loadVersionInfo();
});
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
			版本信息
		</h2>
		<p class="text-green-400  m-4">
			{{ version }}
		</p>
		<p class="text-gray-500 text-sm m-4" v-if="shortCommit">
			Git Commit: {{ shortCommit }}
		</p>
		<p class="text-gray-500 text-sm m-4" v-if="buildTime">
			Build Time: {{ buildTime }}
		</p>
		<p class="text-gray-400 text-sm m-4" v-if="loading">
			版本信息加载中...
		</p>
		<p class="text-red-500 text-sm m-4" v-if="loadError">
			{{ loadError }}
		</p>

		<p class="text-gray-500 text-sm m-4">
			版本号来自当前服务端构建，自动跟随 git revision。
		</p>

	</Aprons>
</template>


<style scoped>

</style>