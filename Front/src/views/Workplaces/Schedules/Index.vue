<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-05-22 20:30:15  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Workplace>
		<Schedules>
			<div v-if="actionError" class="m-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
				{{ actionError }}
			</div>
			<!-- Empty‑state message -->
			<div v-if="schedules.length === 0" class="py-12 text-center text-gray-500">
				没有可用的计划。
			</div>

			<!-- Schedule cards grid -->
			<div v-else class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-3">
				<div
					v-for="schedule in schedules"
					:key="schedule.id"
					class="group rounded-xl bg-[#F9FAFB] p-6 hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
				>
					<RouterLink :to="`/aprons/schedule/${schedule.id}`" class="block">
						<!-- Title -->
						<h2 class="truncate text-xl font-bold text-gray-800 group-hover:text-blue-600">
							{{ schedule.name }}
						</h2>

						<!-- Meta chips -->
						<div class="mt-4 flex flex-wrap gap-2">
							<!-- ID chip -->
							<span class="rounded-lg bg-blue-100 px-3 py-1 text-xs font-mono font-semibold text-blue-600">
              {{ schedule.id }}
            </span>
							<!-- Mode chip -->
							<span
								class="rounded-lg bg-yellow-100 px-3 py-1 text-xs font-mono font-semibold text-yellow-600">
              {{ schedule.mode }}
            </span>
							<!-- Enabled / Disabled chip -->
							<span
								class="rounded-lg px-3 py-1 text-xs font-mono font-semibold"
								:class="schedule.enabled ? 'bg-green-100 text-green-600' : 'bg-red-100 text-red-600'"
							>
              {{ schedule.enabled ? 'Enabled' : 'Disabled' }}
            </span>
						</div>


						<!-- Description -->
						<p class="mt-1 line-clamp-2 text-sm text-gray-600">
							{{ schedule.description }}
						</p>

						<!-- Timestamps -->
						<div class="mt-4 space-y-1 text-xs text-gray-500">
							<div>创建: {{ schedule.created_at }}</div>
							<div>更新: {{ schedule.updated_at }}</div>
						</div>
					</RouterLink>

					<div class="mt-4 flex items-center gap-2">
						<button
							class="rounded-md px-3 py-1.5 text-sm font-medium text-white"
							:class="schedule.enabled ? 'bg-orange-600 hover:bg-orange-700' : 'bg-green-600 hover:bg-green-700'"
							:disabled="isBusy(schedule.id)"
							@click="toggleScheduleEnabled(schedule)"
						>
							{{ isBusy(schedule.id) ? '处理中...' : (schedule.enabled ? '停用' : '启用') }}
						</button>
						<button
							class="rounded-md bg-red-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-red-700"
							:disabled="isBusy(schedule.id)"
							@click="deleteSchedule(schedule)"
						>
							删除
						</button>
					</div>
				</div>
			</div>
		</Schedules>
	</Workplace>
</template>


<script>
import ApiService from "@/services/ApiService.js"; // 引入ApiService
import Schedules from "@/views/Workplaces/Schedules/Schedules.vue";
import Workplace from "@/views/Workplaces/Workplace.vue";

export default {
  components: {Workplace, Schedules},
  data() {
    return {
      schedules: [],  // 存储workplace的日程
		actionError: '',
		busyScheduleId: '',
    };
  },
  methods: {
    getWorkplaceIdFromURL() {
      return this.$route.params.id;
    },
    async fetchWorkplaceSchedule() {
      const workplaceId = this.getWorkplaceIdFromURL(); // 获取workplaceId
      try {
	      const response = await ApiService.getWorkplaceSchedules(workplaceId); // 调用ApiService的方法
        this.schedules = response.data; // 假设返回的数据在response.data中
      } catch (error) {
        console.error("Error fetching workplace schedule:", error);
      }
    },
		isBusy(scheduleId) {
			return this.busyScheduleId === scheduleId;
		},
		async toggleScheduleEnabled(schedule) {
			if (!schedule?.id || this.isBusy(schedule.id)) {
				return;
			}
			this.actionError = '';
			this.busyScheduleId = schedule.id;
			const targetEnabled = !schedule.enabled;
			try {
				const response = targetEnabled
					? await ApiService.enableSchedule(schedule.id)
					: await ApiService.disableSchedule(schedule.id);
				const updated = response?.data;
				schedule.enabled = typeof updated?.enabled === 'boolean' ? updated.enabled : targetEnabled;
				schedule.updated_at = updated?.updated_at || schedule.updated_at;
			} catch (error) {
				this.actionError = error?.response?.data?.message || '更新计划状态失败';
			}
			this.busyScheduleId = '';
		},
		async deleteSchedule(schedule) {
			if (!schedule?.id || this.isBusy(schedule.id)) {
				return;
			}
			if (!window.confirm(`确定删除计划 ${schedule.name || schedule.id} 吗？`)) {
				return;
			}
			this.actionError = '';
			this.busyScheduleId = schedule.id;
			try {
				await ApiService.deleteSchedule(schedule.id);
				this.schedules = this.schedules.filter(item => item.id !== schedule.id);
			} catch (error) {
				this.actionError = error?.response?.data?.message || '删除计划失败';
			}
			this.busyScheduleId = '';
		},
  },
  mounted() {
    this.fetchWorkplaceSchedule(); // 在组件加载时调用fetchWorkplaceSchedule方法
  },
};
</script>
