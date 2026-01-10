<!--
  - Copyright (c)  2025.4.29
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - Index.vue
  - Last Modified: 2025-04-29 00:36:39  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
	<Team>
		<div class="max-w-lg mx-auto mt-10 p-6 bg-white rounded-2xl shadow-lg">
			<h2 class="text-2xl font-bold text-gray-800 mb-6 text-center">团队成员</h2>
			<div v-if="loading" class="text-center text-gray-500">加载中...</div>
			<div v-else>
				<ul class="space-y-4">
					<li
						v-for="(item, index) in members"
						:key="item.user?.id || index"
						class="flex justify-between items-center p-4 bg-gray-50 rounded-lg"
					>
						<div>
							<p class="font-medium text-gray-800">{{ item.user?.username || '未知用户' }}</p>
							<p class="text-gray-500 text-sm">{{ item.user?.email || '' }}</p>
						</div>
						<span class="px-3 py-1 bg-blue-100 text-blue-800 rounded-full text-sm capitalize">{{
								item.role
							}}</span>
					</li>
				</ul>
				<p v-if="error" class="text-red-500 text-sm mt-4">{{ error }}</p>
				<p v-if="!members.length" class="text-gray-500 text-center mt-4">暂无成员</p>
			</div>
		</div>
	</Team>
</template>

<script>
import ApiService from '@/services/ApiService.js'
import Team from '@/views/Teams/Team.vue'

/**
 * === 以下功能：团队成员列表组件 ===
 * 组件用于展示指定团队的成员列表
 *
 * This component displays the list of members for a given team.
 */
export default {
	name: 'TeamMembersList',
	components: {
		Team,
	},
	data() {
		return {
			members: [],
			loading: true,
			error: null,
		}
	},
	created() {
		this.fetchMembers()
	},
	methods: {
		/**
		 * === 以下功能：从 URL 获取团队 ID ===
		 * 返回路由参数中的团队 ID
		 *
		 * Get team ID from route parameters.
		 * @returns {string|number} 团队 ID / Team ID
		 */
		getTeamIdFromURL() {
			return this.$route.params.id
		},
		/**
		 * === 以下功能：获取团队成员 ===
		 * 从 API 获取团队详情并提取成员数组
		 *
		 * Fetch team details and extract member list.
		 * @returns {void}
		 */
		async fetchMembers() {
			const teamId = this.getTeamIdFromURL()
			// === 如果未获取到团队 ID，显示错误 ===
			// If no team ID is found, display error
			if (!teamId) {
				this.error = '无法获取团队 ID'
				this.loading = false
				return
			}
			this.loading = true
			try {
				// === 调用 API 获取团队详情 ===
				// Call API to get team details
				const response = await ApiService.getTeamMembers(teamId)
				// === 提取成员列表，如果不存在则默认空数组 ===
				// Extract member list, default to empty array
				this.members = response.data.members || []
			} catch (err) {
				console.error('Error fetching team members:', err)
				this.error = '获取成员失败'
			} finally {
				this.loading = false
			}
		},
	},
}
</script>

<style scoped>
/* 若已全局引入 Tailwind，此处可留空 */
/* If Tailwind is already globally imported, leave this empty */
</style>
