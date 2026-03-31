<template>
  <div class="flex flex-col items-center justify-center min-h-screen bg-gray-100 p-4">
    <h2 class="text-2xl font-semibold text-gray-800 mb-6">
      {{ name }}
    </h2>
    <div
        v-if="avatarUrl"
        class="relative mb-4 cursor-pointer group"
        @click="goToProfileAvatar"
    >
      <img
          :src="avatarUrl"
          alt="User Avatar"
          class="w-36 h-36 rounded-full object-cover shadow-lg transition duration-300 ease-in-out group-hover:opacity-50"
      />
      <div
          class="absolute inset-0 flex items-center justify-center text-white font-bold text-lg opacity-0 transition duration-300 ease-in-out group-hover:opacity-100"
      >
        修改头像
      </div>
    </div>
  </div>
</template>
<script>
import ApiService from "@/services/ApiService.js"; // 引入ApiService

export default {
  data() {
    return {
      name: null,
      avatarUrl: null,
    };
  },
  created() {
    this.fetchProfile();
  },
  methods: {
    fetchProfile() {
      ApiService.getProfile()
          .then(response => {
            this.name = response.data.results[0].user.username;
            this.avatarUrl = response.data.results[0].avatar;
          })
          .catch(error => {
            console.error('Error fetching avatar:', error);
          });
    },
    goToProfileAvatar() {
      // 跳转到 /profile/avatar 页面
      this.$router.push('/profile/avatar');
    },
  },
};
</script>
