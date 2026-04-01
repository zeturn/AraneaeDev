<template>
  <div class="h-screen bg-gradient-to-r from-blue-100 via-purple-100 to-pink-100 flex justify-center items-center">
    <div class="bg-white bg-opacity-80 backdrop-blur-lg rounded-xl p-8 shadow-lg w-96">
      <h2 class="text-1xl font-semibold text-gray-700 text-left mb-2 ml-0.5">Araneae</h2>
      <h1 class="text-3xl font-semibold text-gray-700 text-left mb-6 ml-0.5">Sign In</h1>

      <div v-if="loginError" class="text-red-500 bg-red-100 p-3 rounded mb-4 text-center">
        {{ loginError }}
      </div>

      <form @submit.prevent="login">
        <div class="relative">
          <input
            v-model="username"
            :class="['field-input block w-full mb-4 p-3', usernameError ? 'login-input-error' : '']"
            placeholder="Email, phone, or username"
            type="text"
          />
          <span v-if="usernameError" class="absolute right-2 top-3 text-sm text-red-500">
            {{ usernameError }}
          </span>
        </div>

        <div class="relative">
          <input
            v-model="password"
            :class="['field-input block w-full mb-6 p-3', passwordError ? 'login-input-error' : '']"
            placeholder="Password"
            type="password"
          />
          <span v-if="passwordError" class="absolute right-2 top-3 text-sm text-red-500">
            {{ passwordError }}
          </span>
        </div>

        <button
          class="btn-primary w-full"
          type="submit"
        >
          Local Sign In
        </button>
      </form>

      <button
        class="btn-muted mt-3 w-full"
        type="button"
        @click="loginWithBasaltPass"
      >
        Sign In with BasaltPass
      </button>

      <div class="text-center mt-4">
        <a class="text-blue-500 hover:underline" href="/register">No account? Create one.</a>
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
      usernameError: '',
      passwordError: '',
      loginError: '',
    };
  },
  methods: {
    resolveNextRoute() {
      const rawNext = typeof this.$route.query.next === 'string' ? this.$route.query.next : '';
      return rawNext.startsWith('/') ? rawNext : '/aprons/workplaces';
    },
    loginWithBasaltPass() {
      if ((import.meta.env.VITE_API_FLAVOR || 'django').toLowerCase() === 'go') {
        this.loginError = 'Go API 模式暂不支持 BasaltPass 登录，请使用本地账号登录。';
        return;
      }
      const backendBase = import.meta.env.VITE_BACKEND_BASE_URL || 'http://localhost:8107';
      const next = encodeURIComponent(this.resolveNextRoute());
      window.location.href = `${backendBase}/api/auth/basaltpass/login/?next=${next}`;
    },
    validateInputs() {
      this.usernameError = '';
      this.passwordError = '';
      this.loginError = '';

      let isValid = true;
      if (!this.username) {
        this.usernameError = 'Username is required';
        isValid = false;
      }
      if (!this.password) {
        this.passwordError = 'Password is required';
        isValid = false;
      }
      return isValid;
    },
    login() {
      if (!this.validateInputs()) {
        return;
      }

      ApiService.login({ username: this.username, password: this.password })
        .then(response => {
          const token = response.data.access || response.data.token;
          if (!token) {
            this.loginError = '登录响应缺少 token。';
            return;
          }
          localStorage.setItem('token', token);
          if (response.data.refresh) {
            localStorage.setItem('refresh_token', response.data.refresh);
          }
          localStorage.setItem('csrf_token', response.data.csrf || '');
          this.$router.push(this.resolveNextRoute());
        })
        .catch(error => {
          if (error.response && error.response.status === 401) {
            this.loginError = 'Invalid username or password.';
          } else {
            console.error(error);
            this.loginError = 'Login failed. Please try again later.';
          }
        });
    },
  },
};
</script>

<style scoped>
.login-input-error {
  outline: 2px solid #ef4444;
  outline-offset: 1px;
  background-color: #fef2f2 !important;
}
</style>
