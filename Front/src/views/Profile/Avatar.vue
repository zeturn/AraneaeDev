<template>
  <div class="flex flex-col items-center justify-center min-h-screen bg-gray-100 p-4">
    <h2 class="text-2xl font-semibold text-gray-800 mb-6">Avatar Management</h2>

    <div v-if="avatarUrl" class="mb-4">
      <img :src="avatarUrl" alt="User Avatar" class="w-36 h-36 rounded-full object-cover shadow-lg"/>
    </div>

    <div v-else class="mb-4 text-gray-500">
      <p class="text-lg font-medium">NO AVATAR~</p>
    </div>

    <input class="mb-4 p-2 border rounded-lg cursor-pointer focus:outline-none focus:ring focus:border-blue-300"
           type="file"
           @change="onFileChange"/>

    <button
        class="px-6 py-2 text-white bg-blue-500 rounded-lg hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-400"
        @click="uploadAvatar">
      Upload Avatar
    </button>
  </div>
</template>

<script>
import ApiService from "@/services/ApiService.js"; // 引入ApiService

export default {
  data() {
    return {
      avatar: null,
      avatarUrl: null,
    };
  },
  created() {
    this.fetchAvatar();
  },
  methods: {
    fetchAvatar() {
      ApiService.getProfileAvatar()
          .then(response => {
            this.avatarUrl = response.data.avatar;
          })
          .catch(error => {
            console.error('Error fetching avatar:', error);
          });
    },
    onFileChange(e) {
      const file = e.target.files[0];
      this.avatar = file;
      this.avatarUrl = URL.createObjectURL(file);
    },
    async uploadAvatar() {
      if (!this.avatar) {
        alert('Please select a file first!');
        return;
      }
      const formData = new FormData();
      formData.append('avatar', this.avatar);

      try {
        ApiService.updateProfileAvatar(formData)
            .then(response => {
              alert('Avatar uploaded successfully');
              this.fetchAvatar(); // 重新获取头像以更新显示
            })
            .catch(error => {
              console.error(error);
            });

      } catch (error) {
        console.error('Error uploading avatar:', error);
      }
    },
  },
};
</script>

<style scoped>
/* 此处保留的样式，具体视觉效果由 Tailwind CSS 实现 */
</style>

