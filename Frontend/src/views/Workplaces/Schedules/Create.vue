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
			<div class="container">
				<div class="bg-white rounded-lg m-4">
					<form @submit.prevent="handleCreateSchedule">
						<div class="space-y-2">
							<label class="block text-lg font-medium text-gray-800" for="name">
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
							<label class="block text-lg font-medium text-gray-800" for="description">Description</label>
							<textarea
								id="description"
								v-model="newSchedule.description"
								class="field-input"
								placeholder="Enter description"
							/>
						</div>

						<div class="space-y-2 py-4 bg-white">
							<div>
								<label for="enabled" class="block text-lg font-medium text-gray-800 mb-2">
									<span>Enabled</span>
									<span class="ml-1 text-red-500">*</span>
								</label>
								<div class="flex items-center space-x-3">
									<input
										id="enabled"
										v-model="newSchedule.enabled"
										type="checkbox"
										class="h-6 w-6 accent-teal-600"
										required
									/>
									<p class="text-sm text-gray-500">
										Enable this schedule to run automatically.
									</p>
								</div>
							</div>
						</div>


						<div class="space-y-2">
							<div class="bg-white rounded-lg shadow-md p-6 space-y-6">
								<div class="space-y-4">
									<label class="text-lg font-medium text-gray-800">Order Configuration</label>
									<div class="space-y-4">
										<div
											v-for="(schedule, index) in schedulesConfig"
											:key="index"
											class="bg-gray-50 p-4 rounded-lg shadow-sm space-y-3"
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
											<div class="flex items-center space-x-4">
												<div class="w-1/2" v-if="index === 0">
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
												<div class="w-1/2" v-else>
													<label class="block text-sm font-medium text-gray-600">Trigger</label>
													<div class="w-full px-4 py-2 border rounded-lg shadow-sm bg-gray-100 text-gray-700">
														Previous Task Completion
													</div>
												</div>
												<div v-if="index === 0 && schedule.trigger === 'crons'" class="w-1/2">
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
												<div class="w-full px-4 py-2 border rounded-lg shadow-sm bg-gray-100 text-gray-700">
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
													<option v-for="node in nodesList" :key="node.id" :value="node.id">
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
				<div class="mt-6 bg-white rounded-lg shadow-md p-6">
					<h3 class="text-xl font-semibold text-gray-700">Generated JSON</h3>
					<pre class="mt-2 p-4 bg-gray-50 rounded-lg text-sm text-gray-800">{{ generatedOrderJson }}</pre>
				</div>
			</div>
		</Schedules>
	</Workplace>
</template>

<script setup>
import {ref, reactive, computed, onMounted} from 'vue';
import {useRoute} from 'vue-router';
import ApiService from '@/services/ApiService.js';
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
