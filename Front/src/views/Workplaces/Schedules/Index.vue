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
			<!-- Empty‑state message -->
			<div v-if="schedules.length === 0" class="py-12 text-center text-gray-500">
				没有可用的计划。
			</div>

			<!-- Schedule cards grid -->
			<div v-else class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-3">
				<RouterLink
					v-for="schedule in schedules"
					:key="schedule.id"
					:to="`/aprons/schedule/${schedule.id}`"
					class="group rounded-xl bg-[#F9FAFB] p-6 hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
				>
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
  },
  mounted() {
    this.fetchWorkplaceSchedule(); // 在组件加载时调用fetchWorkplaceSchedule方法
  },
};
</script>
