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
								class="w-full px-4 py-2 border rounded-lg shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
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
								class="w-full px-4 py-2 border rounded-lg shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
								placeholder="Enter description"
							/>
						</div>
						<div class="space-y-2">
							<label class="block text-lg font-medium text-gray-800" for="mode">
								<span>Mode</span>
								<span class="ml-1 text-red-500">*</span>
							</label>
							<select
								id="mode"
								v-model="newSchedule.mode"
								class="w-full px-4 py-2 border rounded-lg shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
								required
							>
								<option value="once">Once</option>
								<option value="recurring">Recurring</option>
							</select>
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
										class="h-6 w-6 text-blue-600 border-gray-300 rounded transition duration-150 focus:ring-2 focus:ring-blue-500"
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
											<div class="block text-sm font-medium text-gray-600">
												<label class="block text-sm font-medium text-gray-600">Task
													Status</label>
												<select
													type="checkbox"
													v-model="schedule.task_status"
													class="w-full px-4 py-2 border rounded-lg shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500">
													<option value="new">new</option>
													<option value="exist">exist</option>
												</select>
											</div>
											<div v-if="schedule.task_status == 'new'">
												<div class="flex items-center space-x-4">
													<div class="flex-1">
														<label class="block text-sm font-medium text-gray-600">Task
															Name</label>
														<input
															v-model="schedule.name"
															class="w-full px-4 py-2 border rounded-lg shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
															placeholder="Task Name"
															type="text"
														/>
													</div>
													<div class="w-1/2">
														<label
															class="block text-sm font-medium text-gray-600">Project</label>
														<select
															v-model="schedule.project_id"
															class="w-full px-4 py-2 border rounded-lg shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
														>
															<option v-for="project in projectsList" :key="project.id"
															        :value="project.id">
																{{ project.name }}
															</option>
														</select>
													</div>
												</div>
											</div>
											<div v-else>
												<label class="block text-sm font-medium text-gray-600">Task Name</label>
												<select
													v-model="schedule.task_id"
													class="w-full px-4 py-2 border rounded-lg shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
													@change="updateTaskName(schedule)"
												>
													<option v-for="task in tasksList" :key="task.id" :value="task.id">
														{{ task.name }}
													</option>
												</select>
											</div>

											<!-- trigger config-->
											<div class="flex items-center space-x-4">
												<div class="w-1/2">
													<label
														class="block text-sm font-medium text-gray-600">Trigger</label>
													<select
														v-model="schedule.trigger"
														class="w-full px-4 py-2 border rounded-lg shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
													>
														<option value="crons">Crons</option>
														<option value="previous">Previous Schedule</option>
													</select>
												</div>
												<div v-if="schedule.trigger === 'crons'" class="w-1/2">
													<label class="block text-sm font-medium text-gray-600">Cron
														Expression</label>
													<input
														v-model="schedule.crons"
														class="w-full px-4 py-2 border rounded-lg shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
														placeholder="* * * * * *"
													/>
												</div>
											</div>
											<div v-if="schedule.trigger === 'previous'">
												<label class="block text-sm font-medium text-gray-600">Previous
													Schedule</label>
												<select
													v-model="schedule.previous"
													class="w-full px-4 py-2 border rounded-lg shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
												>
													<option v-for="(s, idx) in schedulesConfig" :key="idx"
													        :value="s.name">
														{{ s.name }}
													</option>
												</select>
											</div>
											<div>
												<label class="block text-sm font-medium text-gray-600">Nodes</label>
												<select
													v-model="schedule.node"
													multiple
													class="w-full px-4 py-2 border rounded-lg shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
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
										class="px-4 py-2 bg-gray-700 text-white rounded-md hover:bg-gray-900 focus:ring-2 focus:ring-gray-500"
										type="button"
										@click="addScheduleConfig"
									>Add New Task
									</button>
								</div>
							</div>
						</div>
						<div class="flex justify-end mt-6">
							<button
								type="submit"
								class="px-6 py-3 bg-gray-800 text-white text-lg rounded-md shadow-lg hover:bg-gray-900 focus:ring-2 focus:ring-gray-500"
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
import Tabs from '@/components/Tabs.vue';
import ApiService from '@/services/ApiService.js';
import Schedules from '@/views/Workplaces/Schedules/Schedules.vue';
import Workplace from '@/views/Workplaces/Workplace.vue';


function updateTaskName(schedule) {
	// 中文：查找选中的任务
	// EN: Find the selected task
	const selected = tasksList.value.find(t => t.id === schedule.task_id);
	if (selected) {
		schedule.name = selected.name;
	} else {
		schedule.name = '';
	}
}

const route = useRoute();
const workplaceId = computed(() => route.params.id || 'default-id');

const tabsList = computed(() => [
	{name: '程序计划', url: `/aprons/workplaces/${workplaceId.value}/schedules`},
	{name: '创建计划', url: `/aprons/workplaces/${workplaceId.value}/schedules/create`}
]);

const schedules = ref([]);
const newSchedule = reactive({name: '', description: '', order: '', mode: 'once', enabled: true});
const newOrder = reactive({name: '', schedules: []});
const schedulesConfig = ref([
	{
		task_status: 'exist',
		task_id: " ",
		name: '',
		project_id: null,
		node: [],
		trigger: 'crons',
		crons: '',
		previous: ''
	}
]);
const nodesList = ref([]);
const projectsList = ref([]);
const tasksList = ref([]);

const generatedOrderJson = computed(() => JSON.stringify(
	{
		name: newSchedule.name,
		schedule: schedulesConfig.value.map(s => ({
			task_status: s.task_status,
			task_id: s.task_id || undefined,
			name: s.name,
			project_id: s.project_id,
			node: s.node,
			trigger: s.trigger,
			crons: s.trigger === 'crons' ? s.crons : undefined,
			previous: s.trigger === 'previous' ? (() => {
				const prev = schedulesConfig.value.find(cfg => cfg.name === s.previous);
				return prev ? prev.name : undefined;
			})() : undefined
		}))
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

const fetchProjectsList = async () => {
	try {
		const res = await ApiService.getWorkplaceProjects(workplaceId.value);
		projectsList.value = res.data;
	} catch (err) {
		console.error('Error fetching projects:', err);
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
		task_status: 'exist',
		task_id: '',
		name: '',
		project_id: null,
		node: [],
		trigger: 'crons',
		crons: '',
		previous: ''
	});
};

const handleCreateSchedule = async () => {
	const orderPayload = {
		name: newSchedule.name, schedule: schedulesConfig.value.map(s => ({
			task_status: s.task_status,
			task_id: s.task_id,
			name: s.task_status === 'exist'
				? tasksList.value.find(task => task.id === s.task_id)?.name
				: s.name,
			project_id: s.project_id ? s.project_id : undefined,
			node: s.node,
			trigger: s.trigger,
			crons: s.trigger === 'crons' ? s.crons : undefined,
			previous: s.trigger === 'previous' ? (() => {
				const prev = schedulesConfig.value.find(cfg => cfg.name === s.previous);
				if (!prev) return undefined;
				return prev.task_status === 'exist'
					? tasksList.value.find(task => task.id === prev.task_id)?.name
					: prev.name;
			})() : undefined
		}))
	};

	const schedulePayload = {
		name: newSchedule.name,
		description: newSchedule.description,
		mode: newSchedule.mode,
		enabled: newSchedule.enabled,
		workplace: workplaceId.value,
		order: JSON.stringify(orderPayload)
	};

	try {
		const res = await ApiService.createSchedule(schedulePayload);
		schedules.value.push(res.data);
		Object.assign(newSchedule, {name: '', description: '', order: '', mode: 'once', enabled: true});
		Object.assign(newOrder, {name: '', schedules: []});
		schedulesConfig.value = [{name: '', project_id: null, node: [], trigger: 'crons', crons: '', previous: ''}];
	} catch (err) {
		console.error('Error creating schedule:', err.response?.data || err);
	}
};

onMounted(() => {
	fetchNodesList();
	fetchProjectsList();
	fetchWorkplaceSchedule();
	fetchWorkplaceTasks();
});
</script>
