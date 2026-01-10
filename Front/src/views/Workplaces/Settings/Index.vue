<!--
  - Copyright (c)   2024.11  Henry Zhao. All rights reserved.
  - From CA.
  -->

<template>
	<Workplace>
		<div class="max-w-xl mx-auto p-4">
			<h1 class="text-2xl font-bold mb-4">工作站设置: {{ name }}</h1>

			<!-- 修改名称 -->
			<LCard class="p-4 mb-4">
				<h2 class="text-xl font-semibold mb-2">更改工作站名称</h2>
				<!-- 确保 v-model 绑定到 value 事件 -->
				<LInput
					v-model:value="inputValue"
					:activeBorderColor="'#4ADE80'"
					:borderColor="'#D1D5DB'"
					label="请输入新名称"
				/>
				<LButton
					class="mt-3"
					text="保存"
					@click="renameWorkplace"
				/>
			</LCard>

			<!-- 删除工作站 -->
			<LCard class="p-4 mb-4">
				<h2 class="text-xl font-semibold mb-2">删除工作站</h2>
				<LButton
					color="red"
					text="删除"
					type="outline"
					@click="confirmDelete"
				/>
			</LCard>

			<!-- 元信息展示 -->
			<div class="mt-6 text-gray-600">
				<p>创建时间: {{ formatDate(created_at) }}</p>
				<p>最后编辑: {{ formatDate(edited_at) }}</p>
			</div>
		</div>
	</Workplace>
</template>

<script>
import ApiService from "@/services/ApiService.js";
import Workplace from "@/views/Workplaces/Workplace.vue";
import LCard from "@/components/LCard.vue";
import LButton from "@/components/LButton.vue";
import LInput from "@/components/LInput.vue";

export default {
	components: {Workplace, LCard, LButton, LInput},
	data() {
		return {
			id: null,
			name: '',
			inputValue: '',
			created_at: null,
			edited_at: null,
			owners: [],
			editors: []
		};
	},
	created() {
		this.fetchWorkplace();
	},
	methods: {
		getWorkplaceIdFromURL() {
			return this.$route.params.id;
		},
		fetchWorkplace() {
			const workplaceId = this.getWorkplaceIdFromURL();
			ApiService.getWorkplace(workplaceId)
				.then(response => {
					const data = response.data;
					this.id = data.id;
					this.name = data.name;
					this.inputValue = data.name;
					this.created_at = data.created_at;
					this.edited_at = data.updated_at;
					this.owners = data.owners;
					this.editors = data.editors;
				})
				.catch(error => {
					console.error('Error fetching workplace data:', error);
					this.$toast.error('无法加载工作站信息');
				});
		},
		formatDate(dateString) {
			const options = {year: 'numeric', month: 'long', day: 'numeric', hour: 'numeric', minute: 'numeric'};
			return new Date(dateString).toLocaleDateString(undefined, options);
		},
		renameWorkplace() {
			const newName = this.inputValue.trim();
			if (!newName) {
				this.$toast.warning('名称不能为空');
				return;
			}
			// 调试日志：确认 payload
			console.log('Renaming workplace to:', newName);
			console.log('Payload:', {name: newName});
			ApiService.updateWorkplace(this.id, {name: newName})
				.then(() => {
					this.name = newName;
					this.edited_at = new Date().toISOString();
					this.$toast.success('名称已更新');
				})
				.catch(error => {
					console.error('Error updating workplace:', error);
					this.$toast.error('更新失败');
				});
		},
		confirmDelete() {
			if (confirm('确认要删除此工作站吗？此操作不可撤销。')) {
				this.deleteWorkplace();
			}
		},
		deleteWorkplace() {
			ApiService.deleteWorkplace(this.id)
				.then(() => {
					this.$toast.success('工作站已删除');
					this.$router.push({name: 'WorkplacesList'});
				})
				.catch(error => {
					console.error('Error deleting workplace:', error);
					this.$toast.error('删除失败');
				});
		}
	}
};
</script>

<style scoped>
/* 可根据设计系统调整样式 */
</style>

