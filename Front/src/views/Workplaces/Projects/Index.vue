<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-05-19 20:32:35  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<script>
import {ref, onMounted} from 'vue';
import {useRoute} from 'vue-router'; // 导入 useRoute
import Workplace from "@/views/Workplaces/Workplace.vue";
import Project from "@/views/Workplaces/Projects/Project.vue";
import ApiService from '@/services/ApiService.js';

export default {
	components: {Project, Workplace},
	setup() {
		const projects = ref([]);
		const route = useRoute(); // 使用 useRoute 提取路由信息

// 提取 workplaceId
		const workplaceId = route.params.workplaceId || route.path.split('/')[3];

		const fetchWorkplacesProject = async () => {
			try {
				console.info('Fetching workplaces projects...');
				const response = await ApiService.getWorkplaceProjects(workplaceId);
				projects.value = response.data;
				console.log('Projects fetched:', response.data);
			} catch (error) {
				console.error('[HDT]Error fetching workplaces projects:', error);
			}
		};
		onMounted(() => {
			fetchWorkplacesProject();
		});
		return {projects};
	},
};
</script>

<template>
	<Workplace>
		<Project>
			<!-- Empty‑state message -->
			<div v-if="projects.length === 0" class="py-12 text-center text-gray-500">
				没有可用的项目。
			</div>

			<!-- Project cards grid -->
			<div
				v-else
				class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
			>
				<RouterLink
					v-for="project in projects"
					:key="project.id"
					:to="`/aprons/projects/${project.id}`"
					class="group rounded-xl bg-[#F9FAFB] p-6 transition-all duration-200 hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
				>
					<!-- Title -->
					<h2
						class="truncate text-xl font-bold text-gray-800 group-hover:text-blue-600"
					>
						{{ project.name }}
					</h2>

					<!-- Description -->
					<p class="mt-1 line-clamp-2 text-sm text-gray-600">
						{{ project.description }}
					</p>

					<!-- Meta chips -->
					<div class="mt-4 flex flex-wrap gap-2">
			            <span
				            class="rounded-lg bg-blue-100 px-3 py-1 text-xs font-mono font-semibold text-blue-600"
			            >
			              {{ project.id }}
			            </span>
					</div>
				</RouterLink>
			</div>
		</Project>
	</Workplace>
</template>
