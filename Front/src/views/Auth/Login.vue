<!--
  - Copyright (c)   2024.12  Henry Zhao. All rights reserved.
  - From CA.
  -->

<template>
  <div class="h-screen bg-gradient-to-r from-blue-100 via-purple-100 to-pink-100 flex justify-center items-center">
    <div class="bg-white bg-opacity-80 backdrop-blur-lg rounded-xl p-8 shadow-lg w-96">
      <h2 class="text-1xl font-semibold text-gray-700 text-left mb-2 ml-0.5">Araneae</h2>
      <h1 class="text-3xl font-semibold text-gray-700 text-left mb-6 ml-0.5">登录</h1>

      <!-- 全局错误提示 -->
      <div v-if="loginError" class="text-red-500 bg-red-100 p-3 rounded mb-4 text-center">
        {{ loginError }}
      </div>

      <form @submit.prevent="login">
        <!-- 用户名输入框 -->
        <div class="relative">
          <input
              v-model="username"
              :class="['block w-full mb-4 p-3 rounded border', usernameError ? 'border-red-500' : 'border-gray-300']"
              placeholder="电子邮件、电话 或 用户名"
              type="text"
          />
          <!-- 用户名错误提示 -->
          <span v-if="usernameError" class="absolute right-2 top-3 text-sm text-red-500">
            {{ usernameError }}
          </span>
        </div>

        <!-- 密码输入框 -->
        <div class="relative">
          <input
              v-model="password"
              :class="['block w-full mb-6 p-3 rounded border', passwordError ? 'border-red-500' : 'border-gray-300']"
              placeholder="密码"
              type="password"
          />
          <!-- 密码错误提示 -->
          <span v-if="passwordError" class="absolute right-2 top-3 text-sm text-red-500">
            {{ passwordError }}
          </span>
        </div>

        <button
            class="w-full p-3 rounded bg-blue-600 text-white font-medium hover:bg-blue-700 transition-colors duration-200"
            type="submit"
        >
          下一步
        </button>
      </form>
      <div class="text-center mt-4">
        <a class="text-blue-500 hover:underline" href="/register">没有帐户？立即创建一个！</a>
      </div>
    </div>
  </div>
</template>


<script>
import ApiService from '@/services/ApiService';

export default {
  data() {
    return {
      username: '',
      password: '',
      usernameError: '', // 用户名错误信息
      passwordError: '', // 密码错误信息
      loginError: '',    // 登录失败全局错误
    };
  },
  methods: {
    validateInputs() {
      // 重置错误信息
      this.usernameError = '';
      this.passwordError = '';
      this.loginError = '';

      let isValid = true;

      // 验证用户名
      if (!this.username) {
        this.usernameError = '用户名不能为空';
        isValid = false;
      }

      // 验证密码
      if (!this.password) {
        this.passwordError = '密码不能为空';
        isValid = false;
      }

      return isValid;
    },
    login() {
      if (!this.validateInputs()) {
        return; // 如果验证失败，不提交表单
      }

      const credentials = {
        username: this.username,
        password: this.password,
      };

      ApiService.login(credentials)
          .then(response => {
            localStorage.setItem('token', response.data.access);
            localStorage.setItem('csrf_token', response.data.csrf);
            this.$router.push('/aprons/workplaces');
          })
          .catch(error => {
            if (error.response && error.response.status === 401) {
              // 如果是 401 Unauthorized，显示全局错误提示
              this.loginError = '用户名或密码错误，请重试。';
            } else {
              // 处理其他错误
              console.error(error);
              this.loginError = '登录失败，请稍后重试。';
            }
          });
    },
  },
};


</script>
