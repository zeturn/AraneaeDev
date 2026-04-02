<!--
  - Copyright (c)   2025.2  Henry Zhao. All rights reserved.
  - From CA.
  -->

<template>
  <div ref="menuRef" class="relative">
    <button
      type="button"
      class="flex items-center gap-2 rounded-full border border-slate-200 bg-white px-1.5 py-1 transition hover:bg-slate-50"
      aria-haspopup="menu"
      :aria-expanded="menuOpen"
      @click="toggleMenu"
    >
      <SmallAvatar/>
      <span class="hidden max-w-[120px] truncate text-xs font-medium text-slate-600 sm:block">
        {{ userName }}
      </span>
    </button>

    <div
      v-if="menuOpen"
      class="absolute right-0 z-50 mt-2 w-64 rounded-xl border border-slate-200 bg-white p-3"
      role="menu"
    >
      <div class="border-b border-slate-100 pb-3">
        <p class="text-sm font-semibold text-slate-900">{{ userName }}</p>
        <p class="mt-1 truncate text-xs text-slate-500">{{ userEmail || 'No email information' }}</p>
      </div>
      <div class="mt-2 space-y-1">
        <RouterLink
          to="/profile"
          class="block rounded-md px-3 py-2 text-sm text-slate-700 transition hover:bg-slate-100"
          role="menuitem"
          @click="closeMenu"
        >
          个人信息
        </RouterLink>
        <RouterLink
          to="/logout"
          class="block rounded-md px-3 py-2 text-sm text-red-600 transition hover:bg-red-50"
          role="menuitem"
          @click="closeMenu"
        >
          退出登录
        </RouterLink>
      </div>
    </div>
  </div>
</template>

<script setup>
import {onBeforeUnmount, onMounted, ref} from 'vue';
import ApiService from '@/services/ApiService';
import SmallAvatar from '@/components/SmallAvatar.vue';
import { getAccessToken } from '@/utils/authStorage';

const menuOpen = ref(false);
const menuRef = ref(null);
const userName = ref('Current user');
const userEmail = ref('');

const decodeJwtPayload = token => {
  if (!token || token.split('.').length < 2) {
    return null;
  }
  try {
    const base64 = token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64.padEnd(base64.length + ((4 - base64.length % 4) % 4), '=');
    return JSON.parse(atob(padded));
  } catch (_) {
    return null;
  }
};

const fillUserFromToken = () => {
  const token = getAccessToken();
  const payload = decodeJwtPayload(token);
  if (!payload) {
    return;
  }
  userName.value = payload.username || payload.name || payload.sub || userName.value;
  userEmail.value = payload.email || userEmail.value;
};

const normalizeProfile = responseData => {
  if (!responseData) {
    return null;
  }
  if (Array.isArray(responseData?.results) && responseData.results.length > 0) {
    const first = responseData.results[0];
    return first?.user ? {...first, ...first.user} : first;
  }
  if (responseData?.user) {
    return {...responseData, ...responseData.user};
  }
  return responseData;
};

const fetchProfile = async () => {
  fillUserFromToken();
  try {
    const response = await ApiService.getProfile();
    const profile = normalizeProfile(response?.data);
    if (!profile) {
      return;
    }
    userName.value = profile.username || profile.name || userName.value;
    userEmail.value = profile.email || userEmail.value;
  } catch (_) {
    // Keep token-derived fallback when profile API is unavailable.
  }
};

const closeMenu = () => {
  menuOpen.value = false;
};

const toggleMenu = () => {
  menuOpen.value = !menuOpen.value;
};

const onDocumentClick = event => {
  if (!menuOpen.value || !menuRef.value) {
    return;
  }
  if (!menuRef.value.contains(event.target)) {
    closeMenu();
  }
};

const onDocumentKeydown = event => {
  if (event.key === 'Escape') {
    closeMenu();
  }
};

onMounted(() => {
  fetchProfile();
  document.addEventListener('click', onDocumentClick);
  document.addEventListener('keydown', onDocumentKeydown);
});

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocumentClick);
  document.removeEventListener('keydown', onDocumentKeydown);
});
</script>
