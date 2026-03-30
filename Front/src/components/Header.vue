<!--
  - Copyright (c)   2025.2  Henry Zhao. All rights reserved.
  - From CA.
  -->

<template>
  <header class="border-b border-slate-100 bg-white px-4 py-3">
    <div class="flex items-center justify-between gap-4">
      <div class="flex min-w-0 items-center gap-3">
        <button class="rounded-lg px-2 py-1 text-emerald-300 transition hover:bg-slate-100" @click="$emit('toggleSidebar')">
          ☰
        </button>

        <div class="min-w-0">
          <div class="flex items-baseline gap-2">
            <h1 class="text-lg font-semibold text-blue-600">Araneae</h1>
            <p class="text-sm font-semibold text-green-400">demo V0.0.1</p>
          </div>

          <nav aria-label="Breadcrumb" class="mt-1" v-if="breadcrumbItems.length">
            <ol class="flex items-center gap-1 overflow-x-auto whitespace-nowrap text-xs text-slate-500">
              <li v-for="(item, index) in breadcrumbItems" :key="`${item.to}-${index}`" class="flex items-center">
                <router-link
                  v-if="!item.current"
                  :to="item.to"
                  class="rounded-md px-1.5 py-0.5 transition hover:bg-slate-100 hover:text-slate-700"
                >
                  {{ item.label }}
                </router-link>
                <span v-else class="rounded-md bg-slate-100 px-1.5 py-0.5 text-slate-700">{{ item.label }}</span>
                <span v-if="index < breadcrumbItems.length - 1" class="px-1 text-slate-300">/</span>
              </li>
            </ol>
          </nav>
        </div>
      </div>

      <div>
        <AvatarToggle class="h-12 w-12"/>
      </div>
    </div>
  </header>
</template>

<script lang="ts" setup>
import {computed} from 'vue';
import {useRoute} from 'vue-router';
import AvatarToggle from "@/components/AvatarToggle.vue";

const route = useRoute();

const routeLabelMap: Record<string, string> = {
  aprons: '控制台',
  workplaces: '工作区',
  projects: '项目',
  nodes: '节点',
  teams: '团队',
  settings: '设置',
  help: '帮助',
  about: '关于',
  favorites: '收藏',
  tasks: '任务',
  schedule: '计划',
  schedules: '计划',
  repo: '版本仓库',
  versions: '版本',
  distribute: '分发',
  order: '分发任务',
  create: '创建',
  edit: '编辑',
  AnalyticsAndLogging: '分析日志',
  profile: '个人中心',
};

const paramPrefixMap: Record<string, string> = {
  id: 'ID',
  taskId: '任务',
  versionId: '版本',
  projectId: '项目',
  teamId: '团队',
  nodeId: '节点',
};

const formatParamLabel = (key: string, value: string) => {
  const prefix = paramPrefixMap[key] || key;
  const shortValue = value.length > 14 ? `${value.slice(0, 6)}...${value.slice(-4)}` : value;
  return `${prefix} ${shortValue}`;
};

const breadcrumbItems = computed(() => {
  const segments = route.path.split('/').filter(Boolean);
  const paramEntries = Object.entries(route.params).map(([key, value]) => {
    const resolved = Array.isArray(value) ? String(value[0] ?? '') : String(value ?? '');
    return [key, resolved] as const;
  });

  let cursor = '';
  return segments.map((segment, index) => {
    cursor += `/${segment}`;

    const matchedParam = paramEntries.find(([, value]) => value === segment);
    const label = matchedParam
      ? formatParamLabel(matchedParam[0], segment)
      : (routeLabelMap[segment] || decodeURIComponent(segment));

    return {
      label,
      to: cursor,
      current: index === segments.length - 1,
    };
  });
});
</script>

<style scoped>
/* Add any additional styles here */
</style>
