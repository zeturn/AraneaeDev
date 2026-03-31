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
export default {
  data() {
    return {
      message: '正在完成 BasaltPass 登录...',
      error: '',
    };
  },
  mounted() {
    const params = new URLSearchParams(window.location.search);
    const access = params.get('access');
    const refresh = params.get('refresh');
    const next = params.get('next') || '/aprons/workplaces';
    const error = params.get('error');

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

    localStorage.setItem('token', access);
    if (refresh) {
      localStorage.setItem('refresh', refresh);
    }
    this.$router.replace(next);
  },
};
</script>
