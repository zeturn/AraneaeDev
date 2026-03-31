<template>
  <div class="h-screen bg-gradient-to-r from-blue-100 via-purple-100 to-pink-100 flex justify-center items-center">
    <div class="bg-white bg-opacity-80 backdrop-blur-lg rounded-xl p-8 shadow-lg w-96">
      <h2 class="text-1xl font-semibold text-gray-700 text-left mb-2 ml-0.5">Araneae</h2>
      <h1 class="text-3xl font-semibold text-gray-700 text-left mb-6 ml-0.5">注册</h1>

      <h1>
        此项目没有开放注册
      </h1>
      <a class="text-blue-500 hover:underline" href="/login">《=返回登陆</a>

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
    };
  },
  methods: {
    login() {
      const credentials = {
        username: this.username,
        password: this.password,
      };
      ApiService.login(credentials)
          .then(response => {
            localStorage.setItem('token', response.data.access);
            localStorage.setItem('csrf_token', response.data.csrf);
            this.$router.push('/workplaces');
          })
          .catch(error => {
            console.error(error);
          });
    },
  },
};
</script>
