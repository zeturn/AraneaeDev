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
            :class="['block w-full mb-4 p-3 rounded border', usernameError ? 'border-red-500' : 'border-gray-300']"
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
            :class="['block w-full mb-6 p-3 rounded border', passwordError ? 'border-red-500' : 'border-gray-300']"
            placeholder="Password"
            type="password"
          />
          <span v-if="passwordError" class="absolute right-2 top-3 text-sm text-red-500">
            {{ passwordError }}
          </span>
        </div>

        <button
          class="w-full p-3 rounded bg-blue-600 text-white font-medium hover:bg-blue-700 transition-colors duration-200"
          type="submit"
        >
          Local Sign In
        </button>
      </form>

      <button
        class="mt-3 w-full p-3 rounded border border-gray-300 text-gray-700 font-medium hover:bg-gray-100 transition-colors duration-200"
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
    loginWithBasaltPass() {
      const backendBase = import.meta.env.VITE_BACKEND_BASE_URL || 'http://localhost:8107';
      const next = encodeURIComponent('/aprons/workplaces');
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
          localStorage.setItem('token', response.data.access);
          localStorage.setItem('csrf_token', response.data.csrf || '');
          this.$router.push('/aprons/workplaces');
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
