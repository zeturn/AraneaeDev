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

<template>
  <Schedules>
    <div class="container">
      <div v-if="loading" class="text-center text-gray-500 text-lg">加载中...</div>
      <div v-else class="bg-white rounded-lg m-4 p-6">
        <form @submit.prevent="handleUpdateSchedule">
          <div class="space-y-2">
            <label class="block text-lg font-medium text-gray-800" for="name">
              Schedule Name
              <span class="ml-1 text-red-500">*</span>
            </label>
            <input
              id="name"
              v-model="form.name"
              class="field-input"
              placeholder="Enter schedule name"
              required
              type="text"
            />
          </div>

          <div class="space-y-2 mt-4">
            <label class="block text-lg font-medium text-gray-800" for="description">Description</label>
            <textarea
              id="description"
              v-model="form.description"
              class="field-input"
              placeholder="Enter description"
            />
          </div>

          <div class="space-y-2 mt-4">
            <label for="enabled" class="block text-lg font-medium text-gray-800 mb-2">
              <span>Enabled</span>
            </label>
            <CheckboxSquareField id="enabled" v-model="form.enabled">
              Enable this schedule to run automatically.
            </CheckboxSquareField>
          </div>

          <div class="space-y-2 mt-6">
            <div class="bg-white rounded-lg shadow-md p-6 space-y-6">
              <div class="space-y-4">
                <label class="text-lg font-medium text-gray-800">Order Configuration</label>
                <div class="space-y-4">
                  <div
                    v-for="(step, index) in schedulesConfig"
                    :key="index"
                    class="bg-gray-50 p-4 rounded-lg shadow-sm space-y-3"
                  >
                    <div class="flex items-center justify-between">
                      <div class="text-sm font-medium text-gray-700">Step {{ index + 1 }}</div>
                      <button
                        v-if="index > 0"
                        type="button"
                        class="btn-danger px-2 py-1 text-sm"
                        @click="removeScheduleConfig(index)"
                      >
                        Remove
                      </button>
                    </div>

                    <div>
                      <label class="block text-sm font-medium text-gray-600">Task</label>
                      <select
                        v-model="step.task_id"
                        class="field-input"
                      >
                        <option disabled value="">Select an existing task</option>
                        <option v-for="task in tasksList" :key="task.id" :value="task.id">
                          {{ task.name }}
                        </option>
                      </select>
                    </div>

                    <div class="flex items-center space-x-4">
                      <div class="w-1/2" v-if="index === 0">
                        <label class="block text-sm font-medium text-gray-600">Trigger</label>
                        <select
                          v-model="step.trigger"
                          class="field-input"
                        >
                          <option value="crons">Crons</option>
                          <option value="api">API Trigger</option>
                        </select>
                      </div>
                      <div class="w-1/2" v-else>
                        <label class="block text-sm font-medium text-gray-600">Trigger</label>
                        <div class="w-full px-4 py-2 border rounded-lg shadow-sm bg-gray-100 text-gray-700">
                          Previous Task Completion
                        </div>
                      </div>
                      <div v-if="index === 0 && step.trigger === 'crons'" class="w-1/2">
                        <label class="block text-sm font-medium text-gray-600">Cron Expression</label>
                        <input
                          v-model="step.crons"
                          class="field-input"
                          placeholder="* * * * * *"
                        />
                      </div>
                    </div>

                    <div v-if="index > 0">
                      <label class="block text-sm font-medium text-gray-600">Previous Task</label>
                      <div class="w-full px-4 py-2 border rounded-lg shadow-sm bg-gray-100 text-gray-700">
                        {{ getTaskName(schedulesConfig[index - 1]?.task_id) || 'Select previous task first' }}
                      </div>
                    </div>

                    <div>
                      <label class="block text-sm font-medium text-gray-600">Nodes</label>
                      <select
                        v-model="step.node"
                        multiple
                        class="field-input"
                      >
                        <option v-for="node in nodesList" :key="node.id" :value="node.celery_queue || String(node.id)">
                          {{ node.name }}
                        </option>
                      </select>
                    </div>
                  </div>
                </div>

                <button
                  class="btn-muted"
                  type="button"
                  @click="addScheduleConfig"
                >
                  Add Task Step
                </button>
              </div>
            </div>
          </div>

          <div class="flex justify-end mt-6">
            <button
              type="submit"
              :disabled="saving"
              class="btn-primary px-6 py-3 text-lg disabled:opacity-50"
            >
              {{ saving ? 'Saving...' : 'Update Schedule' }}
            </button>
          </div>
        </form>

        <p v-if="errorMessage" class="mt-4 text-red-600">{{ errorMessage }}</p>
        <p v-if="successMessage" class="mt-4 text-green-600">{{ successMessage }}</p>
      </div>

      <div v-if="!loading" class="mt-6 bg-white rounded-lg shadow-md p-6">
        <h3 class="text-xl font-semibold text-gray-700">Generated JSON</h3>
        <pre class="mt-2 p-4 bg-gray-50 rounded-lg text-sm text-gray-800">{{ generatedOrderJson }}</pre>
      </div>
    </div>
  </Schedules>
</template>

<script setup>
import {ref, reactive, computed, onMounted} from 'vue';
import {useRoute} from 'vue-router';
import ApiService from '@/services/ApiService';
import CheckboxSquareField from '@/components/BeansDesign/Checkbox/CheckboxSquareField.vue';
import Schedules from '@/views/Schedules/Schedules.vue';

const route = useRoute();
const scheduleId = String(route.params.id || '');

const loading = ref(false);
const saving = ref(false);
const errorMessage = ref('');
const successMessage = ref('');

const nodesList = ref([]);
const tasksList = ref([]);
const form = reactive({
  name: '',
  description: '',
  enabled: true,
  workplace: 'go-workspace',
});

const schedulesConfig = ref([
  {task_id: '', node: [], trigger: 'crons', crons: '', previous: ''},
]);

const parseMaybeJSON = raw => {
  if (!raw) {
    return null;
  }
  if (typeof raw === 'string') {
    try {
      return JSON.parse(raw);
    } catch (_) {
      return null;
    }
  }
  if (typeof raw === 'object') {
    return raw;
  }
  return null;
};

const normalizeNodeList = (node, fallback) => {
  if (Array.isArray(node)) {
    return node.filter(Boolean);
  }
  if (node) {
    return [node];
  }
  if (fallback) {
    return [fallback];
  }
  return [];
};

const getTaskByID = taskID => tasksList.value.find(task => String(task.id) === String(taskID));
const getTaskName = taskID => getTaskByID(taskID)?.name || (taskID ? `task-${taskID}` : '');

const buildOrderSteps = () => {
  return schedulesConfig.value.map((s, index) => {
    const task = getTaskByID(s.task_id);
    const taskName = task?.name || (s.task_id ? `task-${s.task_id}` : `task-step-${index + 1}`);
    const previousTaskName = index > 0 ? getTaskName(schedulesConfig.value[index - 1]?.task_id) : undefined;

    return {
      task_id: s.task_id || undefined,
      name: taskName,
      project_id: task?.project_id || undefined,
      node: Array.isArray(s.node) ? s.node : [],
      trigger: index === 0 ? s.trigger : 'previous',
      crons: index === 0 && s.trigger === 'crons' ? s.crons : undefined,
      previous: index > 0 ? previousTaskName : undefined,
    };
  });
};

const generatedOrderJson = computed(() => JSON.stringify({
  name: form.name,
  schedule: buildOrderSteps(),
}, null, 2));

const addScheduleConfig = () => {
  schedulesConfig.value.push({
    task_id: '',
    node: [],
    trigger: 'previous',
    crons: '',
    previous: '',
  });
};

const removeScheduleConfig = index => {
  if (index <= 0) {
    return;
  }
  schedulesConfig.value.splice(index, 1);
};

const fillFormFromSchedule = schedule => {
  form.name = schedule?.name || '';
  form.description = schedule?.description || '';
  form.enabled = schedule?.enabled !== false;
  form.workplace = schedule?.workplace || 'go-workspace';

  const parsedOrder = parseMaybeJSON(schedule?.order);
  const rawSteps = Array.isArray(parsedOrder?.schedule)
    ? parsedOrder.schedule
    : [{
      task_id: schedule?.task_id || '',
      node: schedule?.node_queue ? [schedule.node_queue] : [],
      crons: schedule?.cron_expr || '',
    }];

  const nextSteps = rawSteps.map((item, index) => {
    const triggerRaw = String(item?.trigger || '').toLowerCase();
    const firstTrigger = triggerRaw === 'crons' || triggerRaw === 'api'
      ? triggerRaw
      : (String(item?.crons || '').trim() ? 'crons' : 'api');

    return {
      task_id: item?.task_id || (index === 0 ? (schedule?.task_id || '') : ''),
      node: normalizeNodeList(item?.node, schedule?.node_queue),
      trigger: index === 0 ? firstTrigger : 'previous',
      crons: index === 0 && firstTrigger === 'crons' ? String(item?.crons || schedule?.cron_expr || '') : '',
      previous: '',
    };
  });

  schedulesConfig.value = nextSteps.length > 0 ? nextSteps : [{task_id: '', node: [], trigger: 'crons', crons: '', previous: ''}];
};

const fetchNodes = async () => {
  const res = await ApiService.getNodesList();
  nodesList.value = Array.isArray(res?.data?.results) ? res.data.results : [];
};

const fetchTasks = async () => {
  const workplace = String(route.params.workplaceId || '');
  if (workplace) {
    const res = await ApiService.getWorkplaceTasks(workplace);
    tasksList.value = Array.isArray(res?.data?.tasks) ? res.data.tasks : [];
    return;
  }

  const res = await ApiService.getTasks();
  tasksList.value = Array.isArray(res?.data) ? res.data : [];
};

const fetchSchedule = async () => {
  const res = await ApiService.getSchedule(scheduleId);
  fillFormFromSchedule(res.data || {});
};

const validateForm = () => {
  if (schedulesConfig.value.length === 0) {
    return 'Please add at least one task step.';
  }

  for (let i = 0; i < schedulesConfig.value.length; i += 1) {
    if (!schedulesConfig.value[i].task_id) {
      return `Please select an existing task for step ${i + 1}.`;
    }
  }

  const firstStepTrigger = schedulesConfig.value[0].trigger;
  if (firstStepTrigger !== 'crons' && firstStepTrigger !== 'api') {
    return 'The first task can only be triggered by cron or API.';
  }
  if (firstStepTrigger === 'crons' && !String(schedulesConfig.value[0].crons || '').trim()) {
    return 'Please provide a cron expression for the first step.';
  }

  return '';
};

const handleUpdateSchedule = async () => {
  errorMessage.value = '';
  successMessage.value = '';

  const validationError = validateForm();
  if (validationError) {
    errorMessage.value = validationError;
    return;
  }

  const orderPayload = {
    name: form.name,
    schedule: buildOrderSteps(),
  };

  saving.value = true;
  try {
    await ApiService.updateSchedule(scheduleId, {
      name: form.name,
      description: form.description,
      enabled: form.enabled,
      workplace: form.workplace,
      order: JSON.stringify(orderPayload),
    });
    successMessage.value = 'Schedule updated successfully.';
  } catch (err) {
    errorMessage.value = err?.response?.data?.message || err?.response?.data?.error || 'Failed to update schedule.';
  } finally {
    saving.value = false;
  }
};

onMounted(async () => {
  loading.value = true;
  errorMessage.value = '';
  try {
    await Promise.all([fetchTasks(), fetchNodes(), fetchSchedule()]);
  } catch (err) {
    errorMessage.value = err?.response?.data?.message || 'Failed to load schedule data.';
  } finally {
    loading.value = false;
  }
});
</script>

<style scoped>
</style>