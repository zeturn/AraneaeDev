<!--
  - Copyright (c)   2024.12  Henry Zhao. All rights reserved.
  - From CA.
  -->

<template>
  <div class="h-screen bg-gradient-to-r from-blue-100 via-purple-100 to-pink-100 flex justify-center items-center">
    <div class="text-center">
      <div v-if="loading">
        <h1 class="text-2xl font-semibold text-gray-700 animate-pulse">Logging out...</h1>
      </div>
      <div v-else>
        <h1 class="text-3xl font-bold text-gray-800 mb-4">You have been logged out.</h1>
        <router-link
            class="text-white bg-blue-500 hover:bg-blue-600 px-4 py-2 rounded shadow-md transition duration-200"
            to="/login">
          Go to Login
        </router-link>
      </div>
    </div>
  </div>
</template>

<script>
import axios from "axios";

export default {
  name: "Logout",
  data() {
    return {
      loading: true,
    };
  },
  methods: {
    async performLogout() {
      try {
        const refreshToken = localStorage.getItem("refresh_token");
        if (refreshToken) {
          await axios.post("http://127.0.0.1:8000/api/logout/", {refresh: refreshToken});
        }
      } catch (error) {
        console.error("Logout failed:", error.response ? error.response.data : error.message);
      } finally {
        // 无论请求成功与否，都清理本地存储
        localStorage.removeItem("token");
        localStorage.removeItem("csrf_token");
        // localStorage.removeItem("refresh_token");
        this.loading = false; // 停止加载
        setTimeout(() => {
          this.$router.push("/login");
        }, 2000); // 2秒后跳转
      }
    },
  },
  created() {
    this.performLogout();
  },
};
</script>

<style scoped>

</style>
