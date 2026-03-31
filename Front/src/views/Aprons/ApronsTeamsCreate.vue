# === 以下功能：美化团队创建页面 (Tailwind CSS 版) ===
<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - ApronsTeamsCreate.vue
  - Last Modified: 2025-05-19 00:49:57  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<!--
  中文: 使用 Tailwind CSS 美化后的团队创建组件
  English: Team creation component beautified with Tailwind CSS.
-->
<template>
	<Aprons>
		<div class="mx-auto max-w-4xl px-4 pb-10">
			<h2 class="text-3xl font-semibold text-gray-500">
				创建团队
			</h2>

			<form
				class="team-panel my-4"
				@submit.prevent="onSubmit"
			>

				<!-- 团队名称 -->
				<div class="mb-5">
					<label
						class="block mb-2 text-gray-700 text-sm font-medium"
						for="name"
					>名称</label>
					<input
						id="name"
						v-model="name"
						class="field-input"
						placeholder="输入团队名称"
						required
						type="text"
					/>
				</div>

				<!-- 描述 -->
				<div class="mb-5">
					<label
						class="block mb-2 text-gray-700 text-sm font-medium"
						for="description"
					>描述</label>
					<textarea
						id="description"
						v-model="description"
						class="field-input h-24 resize-none"
						placeholder="添加团队描述（可选）"
					></textarea>
				</div>

				<!-- 可加入 -->
				<div class="flex items-center mb-6">
					<input
						id="joinAble"
						v-model="joinAble"
						class="h-4 w-4 accent-teal-600"
						type="checkbox"
					/>
					<label
						class="ml-2 text-gray-700 text-sm font-medium"
						for="joinAble"
					>可加入</label>
				</div>

				<button
					:disabled="loading"
					class="btn-primary w-full disabled:opacity-50"
					type="submit"
				>
					<span v-if="loading">提交中...</span>
					<span v-else>创建团队</span>
				</button>

				<p v-if="error" class="mt-2 text-sm text-red-500">{{ error }}</p>
				<p v-if="success" class="mt-2 text-sm text-green-600">创建成功！</p>
			</form>
		</div>
	</Aprons>
</template>


<script setup>
/**
 * 中文: 团队创建组件（Tailwind CSS 版）
 * English: Team creation component (Tailwind CSS version).
 */
import {ref} from 'vue';
import ApiService from '@/services/ApiService.js';
import Aprons from "@/views/Aprons/Aprons.vue";
import EventBus from '@/utils/event-bus'
import router from "@/router/index.js";

const name = ref('');
const description = ref('');
const joinAble = ref(false);
const loading = ref(false);
const error = ref('');
const success = ref(false);

/**
 * 中文: 提交团队创建表单
 * English: Submit the team creation form.
 */
async function onSubmit() {
	// 重置状态 / Reset status
	error.value = '';
	success.value = false;
	loading.value = true;

	try {
		const res = await ApiService.createTeam({
			name: name.value,
			description: description.value,
			join_able: joinAble.value,
		});
		const newId = res.data.id;
		success.value = true;
		// 清空表单 / Clear form
		name.value = '';
		description.value = '';
		joinAble.value = false;

		EventBus.emit('notify', {
			type: 'success',
			title: '创建成功',
			message: '团队已成功创建'
		});

		// 跳转到团队列表 / Redirect to team list
		await router.push({name: 'team', params: {id: newId}});
	} catch (err) {
		// 处理错误并显示 / Handle and display error
		error.value = err.response?.data?.detail || '创建失败，请重试';
	} finally {
		loading.value = false;
	}
}
</script>
