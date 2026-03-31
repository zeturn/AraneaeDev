<!--
  - Copyright (c)  2025.4.29
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-04-29 00:38:10  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
  <Team>
    <div class="mx-auto max-w-4xl px-4 pb-10">
      <div class="team-panel space-y-6">
        <header class="space-y-2">
          <p class="text-xs uppercase tracking-wider text-slate-500">Team Settings</p>
          <h1 class="text-2xl font-semibold text-slate-900">{{ form.name || '团队设置' }}</h1>
          <p class="text-sm text-slate-500">管理团队名称、描述、加入策略与删除操作。</p>
        </header>

        <div class="grid gap-4 md:grid-cols-2">
          <div class="md:col-span-2">
            <label class="mb-2 block text-sm font-medium text-slate-700">团队名称</label>
            <input v-model="form.name" class="field-input" placeholder="输入团队名称" type="text" />
          </div>
          <div class="md:col-span-2">
            <label class="mb-2 block text-sm font-medium text-slate-700">描述</label>
            <textarea v-model="form.description" class="field-input min-h-[120px] resize-none" placeholder="输入团队描述"></textarea>
          </div>
          <div class="md:col-span-2">
            <CheckboxSquareField v-model="form.join_able">允许成员自由加入</CheckboxSquareField>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <button class="btn-primary" :disabled="loading" @click="saveTeam">{{ loading ? '保存中...' : '保存设置' }}</button>
          <router-link :to="`/aprons/teams/${teamId}`" class="btn-muted">返回团队</router-link>
          <button class="btn-danger" :disabled="loading" @click="deleteTeam">删除团队</button>
          <span class="text-sm text-slate-500">{{ notice }}</span>
        </div>

        <div class="grid gap-3 text-sm text-slate-500 md:grid-cols-2">
          <p>创建时间: {{ formatDate(form.created_at) }}</p>
          <p>更新时间: {{ formatDate(form.updated_at) }}</p>
        </div>
      </div>
    </div>
  </Team>
</template>

<script setup>
import {computed, onMounted, reactive, ref} from 'vue';
import {useRoute, useRouter} from 'vue-router';
import ApiService from '@/services/ApiService.js';
import CheckboxSquareField from '@/components/BeansDesign/Checkbox/CheckboxSquareField.vue';
import Team from '@/views/Teams/Team.vue';

const route = useRoute();
const router = useRouter();
const teamId = computed(() => String(route.params.id || ''));

const loading = ref(false);
const notice = ref('');
const form = reactive({
  id: '',
  name: '',
  description: '',
  join_able: false,
  created_at: '',
  updated_at: '',
});

const formatDate = (value) => {
  if (!value) {
    return '-';
  }
  return new Date(value).toLocaleString();
};

const fetchTeam = async () => {
  loading.value = true;
  notice.value = '';
  try {
    const response = await ApiService.getTeam(teamId.value);
    const data = response?.data || {};
    form.id = data.id || teamId.value;
    form.name = data.name || '';
    form.description = data.description || '';
    form.join_able = !!data.join_able;
    form.created_at = data.created_at || '';
    form.updated_at = data.updated_at || '';
  } catch (error) {
    console.error('fetch team failed:', error);
    notice.value = '加载团队失败';
  } finally {
    loading.value = false;
  }
};

const saveTeam = async () => {
  const name = String(form.name || '').trim();
  if (!name) {
    notice.value = '团队名称不能为空';
    return;
  }
  loading.value = true;
  notice.value = '';
  try {
    await ApiService.updateTeam(teamId.value, {
      name,
      description: form.description,
      join_able: !!form.join_able,
    });
    notice.value = '团队设置已保存';
    form.updated_at = new Date().toISOString();
  } catch (error) {
    console.error('save team failed:', error);
    notice.value = error?.response?.data?.detail || '保存失败';
  } finally {
    loading.value = false;
  }
};

const deleteTeam = async () => {
  if (!window.confirm('确认删除该团队？此操作不可撤销。')) {
    return;
  }
  loading.value = true;
  notice.value = '';
  try {
    await ApiService.deleteTeam(teamId.value);
    await router.push('/aprons/teams');
  } catch (error) {
    console.error('delete team failed:', error);
    notice.value = error?.response?.data?.detail || '删除失败';
  } finally {
    loading.value = false;
  }
};

onMounted(fetchTeam);
</script>