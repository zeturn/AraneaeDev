<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-05-22 21:12:44  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Project>
		<div class="max-w-3xl mx-auto p-6 bg-white rounded-xl transition-all">
			<div v-if="loading" class="text-gray-500">Loading projects...</div>
			<div v-else-if="error" class="text-red-500">{{ error }}</div>

			<div v-else>
				<!-- Header Section -->
				<div class="flex justify-between items-center border-b pb-4 mb-6">
					<div>
						<h1 class="text-3xl font-bold text-gray-900">{{ name }}</h1>
						<p class="text-sm text-gray-500 mt-1">
		            <span class="tag-pill">
			              ID: {{ id }}
			            </span>
						</p>
					</div>
					<div class="text-right text-sm text-gray-500">
						<p>Created: {{ formatDate(created_at) }}</p>
						<p>Updated: {{ formatDate(updated_at) }}</p>
					</div>
				</div>
				<div class="grid grid-cols-1 md:grid-cols-2 gap-6">

					<!-- Language -->
					<div class="border border-gray-200 rounded-xl p-4 bg-gray-50">
						<p class="font-semibold text-gray-700">Language:</p>
						<span class="tag-pill">
			              {{ language }}
			            </span>
					</div>

			</div>
			</div>
		</div>
	</Project>
</template>

<script>
import ApiService from "@/services/ApiService.js";
import Project from "@/views/Projects/Project.vue";

export default {
  components: {Project},
  data() {
    return {
      id: null,
      name: null,
      description: null,
      language: null,
      created_at: null,
      updated_at: null,
      loading: true,
      error: null
    };
  },
  created() {
    this.fetchProject();
  },
  methods: {
    getProjectIdFromURL() {
      return this.$route.params.id;
    },
    fetchProject() {
      const projectId = this.getProjectIdFromURL();
      ApiService.getProject(projectId)
          .then(response => {
            this.id = response.data.id;
            this.name = response.data.name;
            this.description = response.data.description;
            this.language = response.data.language;
            this.created_at = response.data.created_at;
            this.updated_at = response.data.updated_at;
            this.loading = false;
          })
          .catch(error => {
            console.error('Error fetching project data:', error);
            this.error = 'Failed to load project data.';
            this.loading = false;
          });
    },
	  formatDate(dateString) {
		  if (!dateString) return '—';
		  const options = {year: 'numeric', month: 'long', day: 'numeric', hour: 'numeric', minute: 'numeric'};
		  return new Date(dateString).toLocaleDateString(undefined, options);
	  }
  }
};
</script>

<style>
</style>
