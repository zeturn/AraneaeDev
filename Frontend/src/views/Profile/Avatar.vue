<template>
  <Aprons>
    <div class="mx-auto flex max-w-3xl flex-col items-center rounded-2xl bg-[#F8FAFC] p-6">
      <h2 class="mb-6 text-2xl font-semibold text-gray-800">头像管理</h2>

      <div v-if="avatarUrl" class="mb-4">
        <img :src="avatarUrl" alt="User Avatar" class="h-36 w-36 rounded-full object-cover shadow-lg"/>
      </div>

      <div v-else class="mb-4 text-gray-500">
        <p class="text-lg font-medium">暂无头像</p>
      </div>

      <input class="mb-4 w-full max-w-md cursor-pointer"
             type="file"
             @change="onFileChange"/>

      <div class="flex w-full max-w-md gap-3">
        <button
            class="btn-primary flex-1 px-6 py-2"
            @click="uploadAvatar">
          上传头像
        </button>
        <button
            class="btn-muted px-4 py-2"
            @click="$router.push('/aprons/profile')">
          返回资料
        </button>
      </div>
    </div>
  </Aprons>
</template>

<script>
import ApiService from "@/services/ApiService.js"; // 引入ApiService
import Aprons from "@/views/Aprons/Aprons.vue";

export default {
  components: {
    Aprons,
  },
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

