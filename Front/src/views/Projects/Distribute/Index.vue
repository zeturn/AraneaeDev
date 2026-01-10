<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-05-22 21:03:37  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Project>
		<Distribute>
			<div class="overflow-x-auto">
				<table class="bg-white rounded-lg overflow-hidden">
					<thead>
					<tr class="bg-gray-100 text-gray-600 uppercase text-sm leading-normal">
						<th class="py-3 px-6 text-left">ID</th>
						<th class="py-3 px-6 text-left">Version Hash</th>
						<th class="py-3 px-6 text-left">Project Name</th>
						<th class="py-3 px-6 text-left">Deployed At</th>
						<th class="py-3 px-6 text-left">Is Active</th>
						<th class="py-3 px-6 text-left">Node</th>
						<th class="py-3 px-6 text-left">Project</th>
						<th class="py-3 px-6 text-left">Version</th>
					</tr>
					</thead>
					<tbody class="text-gray-600 text-sm font-light">
					<tr v-for="item in sourceDistribution" :key="item.id"
					    class="border-b border-gray-200 hover:bg-gray-100">
						<td class="py-3 px-6 text-left whitespace-nowrap">{{ item.id }}</td>
						<td class="py-3 px-6 text-left">{{ item.version_hash }}</td>
						<td class="py-3 px-6 text-left">{{ item.project_name }}</td>
						<td class="py-3 px-6 text-left">{{ new Date(item.deployed_at).toLocaleString() }}</td>
						<td class="py-3 px-6 text-left">{{ item.is_active ? 'Yes' : 'No' }}</td>
						<td class="py-3 px-6 text-left">{{ item.node }}</td>
						<td class="py-3 px-6 text-left">{{ item.project }}</td>
						<td class="py-3 px-6 text-left">{{ item.version }}</td>
					</tr>
					</tbody>
				</table>
			</div>
		</Distribute>
	</Project>
</template>

<script>
import ApiService from "@/services/ApiService.js";
import Project from "@/views/Projects/Project.vue";
import Distribute from "@/views/Projects/Distribute/Distribute.vue";


export default {
	components: {
		Project,
		Distribute,
	},
	data() {
		return {
			project_id: this.$route.params.id,
			sourceDistribution: []
		};
	},
	created() {
		this.fetchSourceDistribution();
	},
	methods: {
		async fetchSourceDistribution() {
			try {
				const response = await ApiService.SourceDistributionList(this.project_id);
				this.sourceDistribution = response.data;
			} catch (error) {
				console.error('Error fetching source distribution:', error);
			}
		}
	}
};
</script>

<style scoped>
/* Add any additional styling here */
</style>