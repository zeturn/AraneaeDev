<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Create.vue
  - Last Modified: 2025-05-22 20:31:08  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Workplace>
		<Schedules>
			<div class="mx-auto w-full max-w-5xl px-4 pb-10">
				<div class="rounded-2xl bg-white py-5">
					<form @submit.prevent="handleCreateSchedule">
						<div class="space-y-2">
							<label class="block text-sm font-normal text-gray-600" for="name">
								Schedule Name
								<span class="ml-1 text-red-500">*</span>
							</label>
							<input
								id="name"
								v-model="newSchedule.name"
								class="field-input"
								placeholder="Enter schedule name"
								required
								type="text"
							/>
						</div>
						<div class="space-y-2">
							<label class="block text-sm font-normal text-gray-600" for="description">Description</label>
							<textarea
								id="description"
								v-model="newSchedule.description"
								class="field-input"
								placeholder="Enter description"
							/>
						</div>

						<div class="space-y-2 py-4">
							<div>
								<label for="enabled" class="mb-2 block text-sm font-normal text-gray-600">
									<span>Enabled</span>
									<span class="ml-1 text-red-500">*</span>
								</label>
								<CheckboxSquareField id="enabled" v-model="newSchedule.enabled">
									Enable this schedule to run automatically.
								</CheckboxSquareField>
							</div>
						</div>


						<div class="space-y-2">
							<div class="rounded-2xl bg-slate-50/70 p-4 sm:p-5 space-y-8">
								<div class="space-y-6">
									<label class="text-sm font-normal text-gray-600">Order Configuration</label>
									<div class="space-y-6">
										<div
											v-for="(schedule, index) in schedulesConfig"
											:key="index"
											class="rounded-xl bg-white p-4 sm:p-5 space-y-5"
										>
											<!-- Order Item -->
											<div>
												<label class="block text-sm font-medium text-gray-600">Task</label>
												<select
													v-model="schedule.task_id"
													class="field-input"
												>
													<option disabled value="">Select an existing task</option>
													<option v-for="task in tasksList" :key="task.id" :value="task.id">
														{{ task.name }}
													</option>
												</select>
											</div>

											<!-- trigger config-->
											<div class="flex flex-col gap-6 md:flex-row md:items-center">
												<div class="w-full md:w-1/2" v-if="index === 0">
													<label
														class="block text-sm font-medium text-gray-600">Trigger</label>
													<select
														v-model="schedule.trigger"
														class="field-input"
													>
														<option value="crons">Crons</option>
														<option value="api">API Trigger</option>
													</select>
												</div>
												<div class="w-full md:w-1/2" v-else>
													<label class="block text-sm font-medium text-gray-600">Trigger</label>
													<div class="w-full rounded-lg bg-slate-100 px-4 py-2 text-gray-700">
														Previous Task Completion
													</div>
												</div>
												<div v-if="index === 0 && schedule.trigger === 'crons'" class="w-full md:w-1/2">
													<label class="block text-sm font-medium text-gray-600">Cron
														Expression</label>
													<input
														v-model="schedule.crons"
														class="field-input"
														placeholder="* * * * * *"
													/>
												</div>
											</div>
											<div v-if="index > 0">
												<label class="block text-sm font-medium text-gray-600">Previous Task</label>
												<div class="w-full rounded-lg bg-slate-100 px-4 py-2 text-gray-700">
													{{ getTaskName(schedulesConfig[index - 1]?.task_id) || 'Select previous task first' }}
												</div>
											</div>
											<div>
												<label class="block text-sm font-medium text-gray-600">Nodes</label>
												<select
													v-model="schedule.node"
													multiple
													class="field-input"
												>
													<option v-for="node in nodesList" :key="node.id" :value="node.celery_queue || String(node.id)">
														{{ node.name }}
													</option>
												</select>
											</div>
											<!-- trigger config end-->
											<!-- Order Item End -->
										</div>
									</div>
									<button
										class="btn-muted"
										type="button"
										@click="addScheduleConfig"
										>Add Task Step
									</button>
								</div>
							</div>
						</div>
						<div class="flex justify-end mt-6">
							<button
								type="submit"
								class="btn-primary px-6 py-3 text-lg"
							>Create Schedule
							</button>
						</div>
					</form>
				</div>
				<div class="mt-6 rounded-2xl bg-slate-50 p-5">
					<h3 class="text-xl font-semibold text-gray-700">Generated JSON</h3>
					<pre class="mt-2 w-full max-w-full overflow-x-auto whitespace-pre-wrap break-words rounded-lg bg-white p-4 text-sm text-gray-800">{{ generatedOrderJson }}</pre>
				</div>
			</div>
		</Schedules>
	</Workplace>
</template>

<script setup>
import {ref, reactive, computed, onMounted} from 'vue';
import {useRoute} from 'vue-router';
import ApiService from '@/services/ApiService.js';
import CheckboxSquareField from '@/components/BeansDesign/Checkbox/CheckboxSquareField.vue';
import Schedules from '@/views/Workplaces/Schedules/Schedules.vue';
import Workplace from '@/views/Workplaces/Workplace.vue';


const route = useRoute();
const workplaceId = computed(() => route.params.id || 'default-id');

const tabsList = computed(() => [
	{name: '程序计划', url: `/aprons/workplaces/${workplaceId.value}/schedules`},
	{name: '创建计划', url: `/aprons/workplaces/${workplaceId.value}/schedules/create`}
]);

const schedules = ref([]);
const newSchedule = reactive({name: '', description: '', order: '', enabled: true});
const schedulesConfig = ref([
	{
		task_id: '',
		node: [],
		trigger: 'crons',
		crons: '',
		previous: ''
	}
]);
const nodesList = ref([]);
const tasksList = ref([]);

const getTaskByID = taskID => tasksList.value.find(task => String(task.id) === String(taskID));
const getTaskName = taskID => getTaskByID(taskID)?.name || (taskID ? `task-${taskID}` : '');

const buildOrderSteps = () => {
	return schedulesConfig.value.map((s, index) => {
		const task = getTaskByID(s.task_id);
		const taskName = task?.name || (s.task_id ? `task-${s.task_id}` : `task-step-${index + 1}`);
		const previousTaskName = index > 0
			? getTaskName(schedulesConfig.value[index - 1]?.task_id)
			: undefined;

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

const generatedOrderJson = computed(() => JSON.stringify(
	{
		name: newSchedule.name,
		schedule: buildOrderSteps(),
	}, null, 2
));

const fetchWorkplaceSchedule = async () => {
	try {
		const res = await ApiService.getWorkplaceSchedules(workplaceId.value);
		schedules.value = res.data;
	} catch (err) {
		console.error('Error fetching workplace schedule:', err);
	}
};

const fetchNodesList = async () => {
	try {
		const res = await ApiService.getNodesList();
		nodesList.value = res.data.results;
	} catch (err) {
		console.error('Error fetching nodes:', err);
	}
};
const fetchWorkplaceTasks = async () => {
	try {
		const res = await ApiService.getWorkplaceTasks(workplaceId.value);
		tasksList.value = res.data.tasks;

		console.log(tasksList.value);
	} catch (err) {
		console.error('Error fetching tasks:', err);
	}
};

const addScheduleConfig = () => {
	schedulesConfig.value.push({
		task_id: '',
		node: [],
		trigger: 'previous',
		crons: '',
		previous: ''
	});
};

const handleCreateSchedule = async () => {
	if (schedulesConfig.value.length === 0) {
		window.alert('Please add at least one task step.');
		return;
	}

	for (let i = 0; i < schedulesConfig.value.length; i += 1) {
		if (!schedulesConfig.value[i].task_id) {
			window.alert(`Please select an existing task for step ${i + 1}.`);
			return;
		}
	}

	const firstStepTrigger = schedulesConfig.value[0].trigger;
	if (firstStepTrigger !== 'crons' && firstStepTrigger !== 'api') {
		window.alert('The first task can only be triggered by cron or API.');
		return;
	}
	if (firstStepTrigger === 'crons' && !String(schedulesConfig.value[0].crons || '').trim()) {
		window.alert('Please provide a cron expression for the first step.');
		return;
	}

	const orderSteps = buildOrderSteps();

	const orderPayload = {
		name: newSchedule.name,
		schedule: orderSteps,
	};

	const schedulePayload = {
		name: newSchedule.name,
		description: newSchedule.description,
		enabled: newSchedule.enabled,
		workplace: workplaceId.value,
		order: JSON.stringify(orderPayload)
	};

	try {
		const res = await ApiService.createSchedule(schedulePayload);
		schedules.value.push(res.data);
		Object.assign(newSchedule, {name: '', description: '', order: '', enabled: true});
		schedulesConfig.value = [{task_id: '', node: [], trigger: 'crons', crons: '', previous: ''}];
	} catch (err) {
		console.error('Error creating schedule:', err.response?.data || err);
	}
};

onMounted(() => {
	fetchNodesList();
	fetchWorkplaceSchedule();
	fetchWorkplaceTasks();
});
</script>
