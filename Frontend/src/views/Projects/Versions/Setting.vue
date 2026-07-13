<template>
  <Project>
    <Repo>
      <div class="version-setting-shell">
        <div class="surface-panel version-setting-panel space-y-6">

          <div class="grid gap-4 md:grid-cols-2">
            <div class="md:col-span-2">
              <label class="mb-2 block text-sm font-medium text-slate-700">{{ $t('版本名称 (file_name)') }}</label>
              <input v-model="form.file_name" type="text" class="field-input" :placeholder="$t('输入版本名称')" />
            </div>
            <div>
              <label class="mb-2 block text-sm font-medium text-slate-700">Version ID</label>
              <input :value="form.id" type="text" class="field-input" readonly />
            </div>
            <div>
              <label class="mb-2 block text-sm font-medium text-slate-700">SHA256</label>
              <input :value="form.sha256" type="text" class="field-input" readonly />
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-3">
            <button class="btn-primary" :disabled="loading" @click="saveVersion">
              {{ loading ? '保存中...' : '保存设置' }}
            </button>
            <router-link :to="`/aprons/projects/${projectId}/repo`" class="btn-muted">{{ $t('返回版本列表') }}</router-link>
            <button class="btn-danger" :disabled="loading" @click="deleteVersion">{{ $t('删除版本') }}</button>
            <span class="text-sm text-slate-500">{{ notice }}</span>
          </div>

          <div class="grid gap-3 text-sm text-slate-500 md:grid-cols-2">
            <p>创建时间: {{ formatDate(form.created_at) }}</p>
            <p>存储路径: {{ form.storage_path || '-' }}</p>
          </div>
        </div>
      </div>
    </Repo>
  </Project>
</template>

<script setup>import { useI18n } from '@/i18n';
const { t } = useI18n();

import {onMounted, reactive, ref} from 'vue';
import {useRoute, useRouter} from 'vue-router';
import ApiService from '@/services/ApiService.js';
import Project from '@/views/Projects/Project.vue';
import Repo from '@/views/Projects/Repo/Repo.vue';

const route = useRoute();
const router = useRouter();

const loading = ref(false);
const notice = ref('');
const projectId = String(route.params.id || '');
const versionId = String(route.params.versionId || '');

const form = reactive({
  id: '',
  file_name: '',
  sha256: '',
  storage_path: '',
  created_at: '',
});

const formatDate = (value) => {
  if (!value) {
    return '-';
  }
  return new Date(value).toLocaleString();
};

const fetchVersion = async () => {
  try {
    const response = await ApiService.getProjectVersion(projectId, versionId);
    const data = response?.data || {};
    form.id = data.id || versionId;
    form.file_name = data.file_name || data.version_hash || '';
    form.sha256 = data.sha256 || '';
    form.storage_path = data.storage_path || '';
    form.created_at = data.created_at || data.release_date || '';
    notice.value = '';
  } catch (error) {
    console.error('fetch version failed:', error);
    notice.value = t('加载版本失败');
  }
};

const saveVersion = async () => {
  const fileName = String(form.file_name || '').trim();
  if (!fileName) {
    notice.value = t('版本名称不能为空');
    return;
  }
  loading.value = true;
  notice.value = '';
  try {
    await ApiService.updateProjectVersion(projectId, versionId, {
      file_name: fileName,
    });
    notice.value = t('版本设置已保存');
  } catch (error) {
    console.error('save version failed:', error);
    notice.value = error?.response?.data?.detail || t('保存失败');
  } finally {
    loading.value = false;
  }
};

const deleteVersion = async () => {
  if (!window.confirm(t('确认删除当前版本？此操作不可撤销。'))) {
    return;
  }
  loading.value = true;
  notice.value = '';
  try {
    await ApiService.deleteProjectVersion(projectId, versionId);
    await router.push(`/aprons/projects/${projectId}/repo`);
  } catch (error) {
    console.error('delete version failed:', error);
    notice.value = error?.response?.data?.detail || t('删除失败');
  } finally {
    loading.value = false;
  }
};

onMounted(fetchVersion);
</script>

<style scoped>
.version-setting-shell {
  margin: 0;
  max-width: none;
  padding: 0;
}

.version-setting-panel {
  background-color: #f3f4f6;
  box-shadow: none;
}
</style>