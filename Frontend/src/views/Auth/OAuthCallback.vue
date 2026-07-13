<template>
  <div class="h-screen flex items-center justify-center bg-gray-50">
    <div class="w-full max-w-md rounded-lg bg-white p-6 shadow">
      <h1 class="text-xl font-semibold text-gray-800">{{ $t('登录处理中') }}</h1>
      <p class="mt-3 text-sm text-gray-600">{{ message }}</p>
      <p v-if="error" class="mt-3 text-sm text-red-600">{{ error }}</p>
    </div>
  </div>
</template>

<script>
import axios from 'axios';
import { setAccessToken, setRefreshToken } from '@/utils/authStorage';
import { resolveBackendBase } from '@/utils/backendBase';

export default {
  data() {
    return {
      message: this.$t('正在完成 BasaltPass 登录...'),
      error: '',
    };
  },
  async mounted() {
    const queryParams = new URLSearchParams(window.location.search);
    const hash = window.location.hash.startsWith('#') ? window.location.hash.slice(1) : '';
    const hashParams = new URLSearchParams(hash);
    const exchangeCode = queryParams.get('code') || hashParams.get('code');
    const refresh = queryParams.get('refresh') || hashParams.get('refresh');
    const next = queryParams.get('next') || hashParams.get('next') || '/aprons/workplaces';
    const safeNext = next.startsWith('/') ? next : '/aprons/workplaces';
    const error = queryParams.get('error') || hashParams.get('error');

    // Scrub sensitive query/hash token fragments from browser history ASAP.
    window.history.replaceState({}, document.title, window.location.pathname);

    if (error) {
      this.error = `登录失败: ${error}`;
      this.message = this.$t('请返回登录页重试。');
      return;
    }

    if (!exchangeCode) {
      this.error = this.$t('登录失败: 缺少 exchange code');
      this.message = this.$t('请返回登录页重试。');
      return;
    }
    const backendBase = resolveBackendBase();

    try {
      const response = await axios.post(`${backendBase}/api/v1/auth/basaltpass/exchange`, {
        code: exchangeCode,
      });
      const access = response?.data?.access || '';
      const exchangeNext = response?.data?.next || safeNext;
      const safeExchangeNext = exchangeNext.startsWith('/') ? exchangeNext : '/aprons/workplaces';
      if (!access) {
        this.error = this.$t('登录失败: access token 交换失败');
        this.message = this.$t('请返回登录页重试。');
        return;
      }

      setAccessToken(access);
      if (refresh) {
        setRefreshToken(refresh);
      }
      this.$router.replace(safeExchangeNext);
    } catch (err) {
      this.error = this.$t('登录失败: exchange 请求失败');
      this.message = this.$t('请返回登录页重试。');
    }
  },
};
</script>
