<!--
  - Copyright (c)  2025.5.24
  - Henry Zhao
  - araneae_front  -  California Beans (HollowData.com)
  - SmallAvatar.vue
  - Last Modified: 2025-05-19 22:20:07  -  Davis, CA
  -
  - All rights reserved. Unauthorized copying of this file, via any medium,
  - is strictly prohibited unless prior written permission is obtained.
  -->

<template>
  <div class="flex flex-col items-center justify-center bg-transparent p-2">
    <div v-if="avatarUrl" class="">
	    <img :src="avatarUrl" alt="User Avatar" class="w-10 h-8 rounded-full object-cover"/>
    </div>

    <div v-else class="mb-2 text-gray-500">
      <p class="text-sm font-medium">NO AVATAR~</p>
    </div>
  </div>
</template>

<script>
import ApiService from "@/services/ApiService.js"; // 引入ApiService

export default {
	name: "SmallAvatar",
  data() {
    return {
      avatarUrl: null,  // 初始头像URL为空
    };
  },
  created() {
    this.fetchAvatar();
  },
	methods: {
		fetchAvatar() {
			ApiService.getProfileAvatar()
				.then(response => {
					// 确保 Content-Type 是图片
					const contentType = response.headers["content-type"];
					if (contentType && contentType.startsWith("image/")) {
						this.validateImage(response.data.avatar);
					} else {
						console.info("Invalid content type, using default avatar.");
						this.avatarUrl = "/src/assets/default_avatar.jpg"; // 不是图片，使用默认头像
					}
				})
				.catch(error => {
					console.error("Error fetching avatar:", error);
					this.avatarUrl = "/src/assets/default_avatar.jpg"; // 兜底
				});
		},
		validateImage(url) {
			const img = new Image();
			img.src = url;
			img.onload = () => {
				this.avatarUrl = url;
			};
			img.onerror = () => {
				console.warn("Invalid image URL, using default.");
				this.avatarUrl = "/src/assets/default_avatar.jpg";
			};
		}
	},



};
</script>

<style scoped>
/* 保留此处的样式，具体视觉效果由 Tailwind CSS 实现 */
</style>
