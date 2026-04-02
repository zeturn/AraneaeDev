<template>
  <div class="h-screen flex items-center justify-center bg-gray-50">
    <div class="w-full max-w-md rounded-lg bg-white p-6 shadow">
      <h1 class="text-xl font-semibold text-gray-800">登录处理中</h1>
      <p class="mt-3 text-sm text-gray-600">{{ message }}</p>
      <p v-if="error" class="mt-3 text-sm text-red-600">{{ error }}</p>
    </div>
  </div>
</template>

<script>
import { setAccessToken, setRefreshToken } from '@/utils/authStorage';

export default {
  data() {
    return {
      message: '正在完成 BasaltPass 登录...',
      error: '',
    };
  },
  mounted() {
    const queryParams = new URLSearchParams(window.location.search);
    const hash = window.location.hash.startsWith('#') ? window.location.hash.slice(1) : '';
    const hashParams = new URLSearchParams(hash);
    const access = queryParams.get('access') || hashParams.get('access');
    const refresh = queryParams.get('refresh') || hashParams.get('refresh');
    const next = queryParams.get('next') || hashParams.get('next') || '/aprons/workplaces';
    const safeNext = next.startsWith('/') ? next : '/aprons/workplaces';
    const error = queryParams.get('error') || hashParams.get('error');

    // Scrub sensitive query/hash token fragments from browser history ASAP.
    window.history.replaceState({}, document.title, window.location.pathname);

    if (error) {
      this.error = `登录失败: ${error}`;
      this.message = '请返回登录页重试。';
      return;
    }

    if (!access) {
      this.error = '登录失败: 缺少 access token';
      this.message = '请返回登录页重试。';
      return;
    }

    setAccessToken(access);
    if (refresh) {
      setRefreshToken(refresh);
    }
    this.$router.replace(safeNext);
  },
};
</script>
