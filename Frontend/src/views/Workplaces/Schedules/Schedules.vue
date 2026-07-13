<!--
  - Copyright (c)   2024.12  Henry Zhao. All rights reserved.
  - From CA.
  -->

<script lang="ts" setup>import { useI18n } from '@/i18n';
const { t } = useI18n();

import {computed} from 'vue';
import {useRoute} from 'vue-router';
import Tabs from '@/components/Tabs.vue';

const tabsList = computed(() => {
  const route = useRoute();
  const id = route.params.id || 'default-id';  // 如果没有ID，使用默认值
  return [
    {name: t('程序计划'), url: `/aprons/workplaces/${id}/schedules`},
    {name: t('创建计划'), url: `/aprons/workplaces/${id}/schedules/create`},
  ];
});
</script>


<template>
	<h1 class="text-gray-500 text-3xl m-4">
		{{ $t('计划') }}
	</h1>
	<p class="text-gray-500 text-sm m-4">
		{{ $t('计划是自动化任务的基础，可以创建定时以及链式计划。') }}
	</p>
	<Tabs :tabs="tabsList" class="m-4"/>
	<slot>
	</slot>
</template>