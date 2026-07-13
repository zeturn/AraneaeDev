<template>
  <div class="h-screen bg-gradient-to-r from-blue-100 via-purple-100 to-pink-100 flex justify-center items-center">
    <div class="bg-white bg-opacity-80 backdrop-blur-lg rounded-xl p-8 shadow-lg w-96">
      <h2 class="text-1xl font-semibold text-gray-700 text-left mb-2 ml-0.5">Araneae</h2>
      <h1 class="text-3xl font-semibold text-gray-700 text-left mb-6 ml-0.5">{{ $t('Sign In') }}</h1>

      <div v-if="loginError" class="text-red-500 bg-red-100 p-3 rounded mb-4 text-center">
        {{ loginError }}
      </div>
      <div v-if="loginNotice" class="text-amber-700 bg-amber-100 p-3 rounded mb-4 text-center">
        {{ loginNotice }}
      </div>

      <form @submit.prevent="login">
        <div class="relative">
          <input
            v-model="username"
            :class="['field-input block w-full mb-4 p-3', usernameError ? 'login-input-error' : '']"
            :placeholder="$t('Username')"
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
            :placeholder="$t('Password')"
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
          {{ $t('Local Sign In') }}
        </button>
      </form>

      <button
        class="btn-muted mt-3 w-full gap-2"
        type="button"
        @click="loginWithBasaltPass"
      >
        <img
          alt=""
          aria-hidden="true"
          class="h-5 w-5 shrink-0 object-contain"
          src="@/assets/basaltpass-logo-symbol.svg"
        />
        {{ $t('Sign In with BasaltPass') }}
      </button>

      <div class="text-center mt-4">
        <a class="text-blue-500 hover:underline" href="/register">{{ $t('No account? Create one.') }}</a>
      </div>
    </div>
  </div>
</template>

<script>
import ApiService from '@/services/ApiService';
import { setAccessToken, setCsrfTokenValue, setRefreshToken } from '@/utils/authStorage';
import { resolveBackendBase } from '@/utils/backendBase';

export default {
  data() {
    return {
      username: '',
      password: '',
      usernameError: '',
      passwordError: '',
      loginError: '',
      loginNotice: '',
    };
  },
  mounted() {
    const reason = typeof this.$route.query.reason === 'string' ? this.$route.query.reason : '';
    if (reason === 'session_expired') {
      this.loginNotice = this.$t('Session expired. Please sign in again.');
    }
  },
  methods: {
    resolveNextRoute() {
      const rawNext = typeof this.$route.query.next === 'string' ? this.$route.query.next : '';
      return rawNext.startsWith('/') ? rawNext : '/aprons/workplaces';
    },
    loginWithBasaltPass() {
      this.loginError = '';
      const backendBase = resolveBackendBase();
      const next = encodeURIComponent(this.resolveNextRoute());
      window.location.href = `${backendBase}/api/auth/basaltpass/login/?next=${next}`;
    },
    validateInputs() {
      this.usernameError = '';
      this.passwordError = '';
      this.loginError = '';

      let isValid = true;
      if (!this.username) {
        this.usernameError = this.$t('Username is required');
        isValid = false;
      }
      if (!this.password) {
        this.passwordError = this.$t('Password is required');
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
            this.loginError = this.$t('登录响应缺少 token。');
            return;
          }
          setAccessToken(token);
          if (response.data.refresh) {
            setRefreshToken(response.data.refresh);
          }
          setCsrfTokenValue(response.data.csrf || '');
          this.$router.push(this.resolveNextRoute());
        })
        .catch(error => {
          if (error.response && error.response.status === 401) {
            this.loginError = this.$t('Invalid username or password.');
          } else {
            console.error(error);
            this.loginError = this.$t('Login failed. Please try again later.');
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
