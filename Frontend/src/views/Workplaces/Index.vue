<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-05-22 21:07:18  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Workplace>
		<div class="max-w-3xl mx-auto p-6 bg-white rounded-xl transition-all">
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
					<p>Created: {{ formatDate(createdAt) }}</p>
					<p>Updated: {{ formatDate(updatedAt) }}</p>
				</div>
			</div>

			<!-- Main Info Section -->
			<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
				<!-- Description -->
				<div>
					<h2 class="text-lg font-semibold text-gray-700 mb-2">Description</h2>
					<p v-if="description" class="text-gray-600">{{ description }}</p>
					<p v-else class="text-gray-400 italic">No description provided.</p>
				</div>

				<!-- Status -->
				<div>
					<h2 class="text-lg font-semibold text-gray-700 mb-2">Status</h2>
					<span class="tag-pill">
						{{ status }}
					</span>
				</div>
			</div>

			<!-- Teams Section -->
			<div class="mt-6">
				<h2 class="text-lg font-semibold text-gray-700 mb-2">Teams</h2>
				<ul class="list-disc list-inside text-gray-600">
					<li v-for="teamId in teams" :key="teamId">
						Team ID: {{ teamId }}
					</li>
					<li v-if="teams.length === 0" class="text-gray-400 italic">No teams assigned.</li>
				</ul>
			</div>
		</div>
	</Workplace>
</template>

<script>
import ApiService from "@/services/ApiService.js";
import Workplace from "@/views/Workplaces/Workplace.vue";

export default {
	name: 'WorkplaceDetail',
	components: {Workplace},
	data() {
		return {
			id: null,
			name: '',
			description: '',
			status: '',
			createdAt: null,
			updatedAt: null,
			teams: []
		};
	},
	created() {
		this.fetchWorkplace();
	},
	methods: {
		getWorkplaceIdFromRoute() {
			return this.$route.params.id;
		},
		async fetchWorkplace() {
			try {
				const res = await ApiService.getWorkplace(this.getWorkplaceIdFromRoute());
				// If using Axios, unwrap data
				const data = res.data || res;
				// Destructure fields
				const {id, name, description, status, created_at, updated_at, teams} = data;
				this.id = id;
				this.name = name;
				this.description = description;
				this.status = status;
				this.createdAt = created_at;
				this.updatedAt = updated_at;
				this.teams = Array.isArray(teams) ? teams : [];
			} catch (error) {
				console.error('Error fetching workplace data:', error);
			}
		},
		formatDate(dateString) {
			if (!dateString) return '—';
			const options = {year: 'numeric', month: 'long', day: 'numeric', hour: 'numeric', minute: 'numeric'};
			return new Date(dateString).toLocaleDateString(undefined, options);
		}
	}
};
</script>
