<template>
  <Aprons>
    <div class="mx-auto max-w-5xl space-y-6 p-1 md:p-2">
      <header class="rounded-2xl bg-[#F8FAFC] p-6">
        <h1 class="text-2xl font-semibold text-slate-900">{{ $t('个人信息') }}</h1>
        <p class="mt-2 text-sm text-slate-500">{{ $t('查看账号详情、组织归属和认证状态。') }}</p>
      </header>

      <p
        v-if="errorMessage"
        class="rounded-xl bg-amber-50 px-4 py-3 text-sm text-amber-700"
      >
        {{ errorMessage }}
      </p>

      <section class="grid gap-6 lg:grid-cols-3">
        <article class="rounded-2xl bg-[#F8FAFC] p-6">
          <div class="relative mx-auto mb-4 w-fit cursor-pointer group" @click="goToProfileAvatar">
            <img
              v-if="avatarUrl"
              :src="avatarUrl"
              alt="User Avatar"
              class="h-32 w-32 rounded-full object-cover shadow-lg transition duration-300 ease-in-out group-hover:opacity-50"
            />
            <div
              v-else
              class="flex h-32 w-32 items-center justify-center rounded-full bg-slate-200 text-3xl font-bold text-slate-600 shadow-lg transition duration-300 ease-in-out group-hover:opacity-50"
            >
              {{ avatarInitial }}
            </div>
            <div
              class="absolute inset-0 flex items-center justify-center rounded-full text-sm font-semibold text-white opacity-0 transition duration-300 ease-in-out group-hover:bg-slate-900/50 group-hover:opacity-100"
            >
              {{ $t('修改头像') }}
            </div>
          </div>

          <p class="text-center text-lg font-semibold text-slate-900">{{ displayName }}</p>
          <p class="mt-1 text-center text-sm text-slate-500">角色: {{ display(profile.role) }}</p>

          <button class="btn-primary mt-4 w-full py-2" @click="goToProfileAvatar">{{ $t('管理头像') }}</button>
        </article>

        <article class="rounded-2xl bg-[#F8FAFC] p-6 lg:col-span-2">
          <h2 class="text-lg font-semibold text-slate-900">{{ $t('账号详情') }}</h2>
          <dl class="mt-4 grid gap-4 sm:grid-cols-2">
            <div class="rounded-xl bg-slate-100 p-3">
              <dt class="text-xs font-medium uppercase tracking-wide text-slate-500">{{ $t('用户 ID') }}</dt>
              <dd class="mt-1 break-all text-sm font-semibold text-slate-900">
                {{ display(profile.id || tokenMeta.uid || tokenMeta.subject) }}
              </dd>
            </div>
            <div class="rounded-xl bg-slate-100 p-3">
              <dt class="text-xs font-medium uppercase tracking-wide text-slate-500">{{ $t('姓名') }}</dt>
              <dd class="mt-1 text-sm font-semibold text-slate-900">{{ display(profile.name) }}</dd>
            </div>
            <div class="rounded-xl bg-slate-100 p-3">
              <dt class="text-xs font-medium uppercase tracking-wide text-slate-500">{{ $t('用户名') }}</dt>
              <dd class="mt-1 text-sm font-semibold text-slate-900">{{ display(profile.username) }}</dd>
            </div>
            <div class="rounded-xl bg-slate-100 p-3">
              <dt class="text-xs font-medium uppercase tracking-wide text-slate-500">{{ $t('角色') }}</dt>
              <dd class="mt-1 text-sm font-semibold text-slate-900">{{ display(profile.role) }}</dd>
            </div>
            <div class="rounded-xl bg-slate-100 p-3">
              <dt class="text-xs font-medium uppercase tracking-wide text-slate-500">{{ $t('邮箱') }}</dt>
              <dd class="mt-1 break-all text-sm font-semibold text-slate-900">{{ display(profile.email) }}</dd>
            </div>
            <div class="rounded-xl bg-slate-100 p-3 sm:col-span-2">
              <dt class="text-xs font-medium uppercase tracking-wide text-slate-500">{{ $t('账号创建时间') }}</dt>
              <dd class="mt-1 text-sm font-semibold text-slate-900">{{ formatDateTime(profile.created_at) }}</dd>
            </div>
          </dl>
        </article>
      </section>

      <section class="grid gap-6 md:grid-cols-2">
        <article class="rounded-2xl bg-[#F8FAFC] p-6">
          <h2 class="text-lg font-semibold text-slate-900">{{ $t('组织归属') }}</h2>
          <div class="mt-4 grid grid-cols-2 gap-3">
            <div class="rounded-xl bg-slate-100 p-3">
              <p class="text-xs uppercase tracking-wide text-slate-500">{{ $t('我的团队') }}</p>
              <p class="mt-1 text-xl font-semibold text-slate-900">{{ teamCount }}</p>
            </div>
            <div class="rounded-xl bg-slate-100 p-3">
              <p class="text-xs uppercase tracking-wide text-slate-500">{{ $t('我的工作区') }}</p>
              <p class="mt-1 text-xl font-semibold text-slate-900">{{ workplaceCount }}</p>
            </div>
          </div>

          <div class="mt-4 space-y-3">
            <div>
              <p class="text-xs font-medium uppercase tracking-wide text-slate-500">{{ $t('团队预览') }}</p>
              <ul v-if="teamPreview.length" class="mt-2 space-y-1 text-sm text-slate-700">
                <li v-for="team in teamPreview" :key="`team-${team.id || team.name}`">
                  {{ team.name || `团队 ${team.id}` }}
                </li>
              </ul>
              <p v-else class="mt-2 text-sm text-slate-500">{{ $t('暂无团队信息') }}</p>
            </div>

            <div>
              <p class="text-xs font-medium uppercase tracking-wide text-slate-500">{{ $t('工作区预览') }}</p>
              <ul v-if="workplacePreview.length" class="mt-2 space-y-1 text-sm text-slate-700">
                <li v-for="workplace in workplacePreview" :key="`workplace-${workplace.id || workplace.name}`">
                  {{ workplace.name || `工作区 ${workplace.id}` }}
                </li>
              </ul>
              <p v-else class="mt-2 text-sm text-slate-500">{{ $t('暂无工作区信息') }}</p>
            </div>
          </div>
        </article>

        <article class="rounded-2xl bg-[#F8FAFC] p-6">
          <h2 class="text-lg font-semibold text-slate-900">{{ $t('认证信息') }}</h2>
          <dl class="mt-4 space-y-3 text-sm">
            <div>
              <dt class="text-xs font-medium uppercase tracking-wide text-slate-500">Subject</dt>
              <dd class="mt-1 break-all font-semibold text-slate-900">{{ display(tokenMeta.subject) }}</dd>
            </div>
            <div>
              <dt class="text-xs font-medium uppercase tracking-wide text-slate-500">UID</dt>
              <dd class="mt-1 break-all font-semibold text-slate-900">{{ display(tokenMeta.uid) }}</dd>
            </div>
            <div>
              <dt class="text-xs font-medium uppercase tracking-wide text-slate-500">Scope</dt>
              <dd class="mt-1 break-all font-semibold text-slate-900">{{ display(tokenMeta.scope) }}</dd>
            </div>
            <div>
              <dt class="text-xs font-medium uppercase tracking-wide text-slate-500">{{ $t('签发时间') }}</dt>
              <dd class="mt-1 font-semibold text-slate-900">{{ formatUnixTimestamp(tokenMeta.issuedAt) }}</dd>
            </div>
            <div>
              <dt class="text-xs font-medium uppercase tracking-wide text-slate-500">{{ $t('过期时间') }}</dt>
              <dd class="mt-1 font-semibold text-slate-900">{{ formatUnixTimestamp(tokenMeta.expiresAt) }}</dd>
            </div>
          </dl>
        </article>
      </section>

      <p v-if="isLoading" class="text-center text-sm text-slate-500">{{ $t('正在加载用户资料...') }}</p>
    </div>
  </Aprons>
</template>

<script>
import ApiService from "@/services/ApiService.js";
import { getAccessToken } from "@/utils/authStorage";
import Aprons from "@/views/Aprons/Aprons.vue";

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

const pickFirstValue = (...values) => {
  for (const value of values) {
    if (value === null || value === undefined) {
      continue;
    }
    if (typeof value === 'string') {
      const trimmed = value.trim();
      if (trimmed) {
        return trimmed;
      }
      continue;
    }
    return value;
  }
  return '';
};

const normalizeProfile = responseData => {
  if (!responseData) {
    return null;
  }
  if (Array.isArray(responseData?.results) && responseData.results.length > 0) {
    const first = responseData.results[0];
    return first?.user ? { ...first, ...first.user } : first;
  }
  if (responseData?.user) {
    return { ...responseData, ...responseData.user };
  }
  return responseData;
};

const toListPayload = payload => {
  if (Array.isArray(payload)) {
    return payload;
  }
  if (Array.isArray(payload?.results)) {
    return payload.results;
  }
  if (Array.isArray(payload?.records)) {
    return payload.records;
  }
  return [];
};

export default {
  components: {
    Aprons,
  },
  data() {
    return {
      isLoading: false,
      errorMessage: '',
      avatarUrl: null,
      profile: {
        id: '',
        name: '',
        username: '',
        email: '',
        role: '',
        created_at: '',
      },
      tokenMeta: {
        subject: '',
        uid: '',
        scope: '',
        issuedAt: null,
        expiresAt: null,
      },
      teamCount: 0,
      workplaceCount: 0,
      teamPreview: [],
      workplacePreview: [],
    };
  },
  computed: {
    displayName() {
      return this.display(this.profile.name || this.profile.username || this.tokenMeta.uid || this.tokenMeta.subject, 'Unknown User');
    },
    avatarInitial() {
      const value = this.displayName;
      return value ? String(value).charAt(0).toUpperCase() : 'U';
    },
  },
  created() {
    this.hydrateFromToken();
    this.fetchProfile();
    this.fetchMembershipSummary();
  },
  methods: {
    hydrateFromToken() {
      const token = getAccessToken();
      const payload = decodeJwtPayload(token);
      if (!payload) {
        return;
      }

      const scopeValue = Array.isArray(payload.scope)
        ? payload.scope.join(' ')
        : pickFirstValue(payload.scope, Array.isArray(payload.scp) ? payload.scp.join(' ') : '');

      this.tokenMeta.subject = pickFirstValue(payload.sub);
      this.tokenMeta.uid = pickFirstValue(payload.uid, payload.user_id, payload.sub);
      this.tokenMeta.scope = pickFirstValue(scopeValue);
      this.tokenMeta.issuedAt = Number.isFinite(Number(payload.iat)) ? Number(payload.iat) : null;
      this.tokenMeta.expiresAt = Number.isFinite(Number(payload.exp)) ? Number(payload.exp) : null;

      this.profile.id = pickFirstValue(this.profile.id, this.tokenMeta.uid);
      this.profile.name = pickFirstValue(
        this.profile.name,
        payload.name,
        payload.preferred_username,
        payload.nickname
      );
      this.profile.username = pickFirstValue(
        this.profile.username,
        payload.username,
        payload.preferred_username,
        this.tokenMeta.uid
      );
      this.profile.email = pickFirstValue(this.profile.email, payload.email);
      this.profile.role = pickFirstValue(
        this.profile.role,
        payload.role,
        Array.isArray(payload.roles) ? payload.roles[0] : ''
      );
    },
    async fetchProfile() {
      this.isLoading = true;
      this.errorMessage = '';
      try {
        const response = await ApiService.getProfile();
        const normalized = normalizeProfile(response?.data);
        if (!normalized) {
          this.errorMessage = this.$t('未获取到完整资料，已展示当前可用信息。');
          return;
        }

        this.profile.id = pickFirstValue(normalized.id, normalized.user_id, this.profile.id, this.tokenMeta.uid);
        this.profile.name = pickFirstValue(
          normalized.name,
          normalized.nickname,
          normalized.preferred_username,
          this.profile.name
        );
        this.profile.username = pickFirstValue(normalized.username, this.profile.username);
        this.profile.email = pickFirstValue(normalized.email, this.profile.email);
        this.profile.role = pickFirstValue(normalized.role, normalized.user_role, this.profile.role);
        this.profile.created_at = pickFirstValue(normalized.created_at, normalized.date_joined, this.profile.created_at);
        this.avatarUrl = pickFirstValue(normalized.avatar, normalized.avatar_url, this.avatarUrl) || null;
      } catch (error) {
        console.error('Error fetching profile:', error);
        this.errorMessage = this.$t('资料接口不可用，已回退到本地认证信息。');
      } finally {
        this.isLoading = false;
      }
    },
    async fetchMembershipSummary() {
      const [teamResult, workplaceResult] = await Promise.allSettled([
        ApiService.getMyTeams(),
        ApiService.getMyWorkplaces(),
      ]);

      if (teamResult.status === 'fulfilled') {
        const data = teamResult.value?.data || {};
        const teams = toListPayload(data);
        const parsedCount = Number(data?.count);
        this.teamCount = Number.isFinite(parsedCount) ? parsedCount : teams.length;
        this.teamPreview = teams.slice(0, 3);
      }

      if (workplaceResult.status === 'fulfilled') {
        const data = workplaceResult.value?.data || {};
        const workplaces = toListPayload(data);
        const parsedCount = Number(data?.count);
        this.workplaceCount = Number.isFinite(parsedCount) ? parsedCount : workplaces.length;
        this.workplacePreview = workplaces.slice(0, 3);
      }
    },
    formatDateTime(value) {
      if (!value) {
        return '-';
      }
      const date = new Date(value);
      if (Number.isNaN(date.getTime())) {
        return '-';
      }
      return date.toLocaleString();
    },
    formatUnixTimestamp(value) {
      if (!Number.isFinite(Number(value))) {
        return '-';
      }
      return this.formatDateTime(Number(value) * 1000);
    },
    display(value, fallback = '-') {
      if (value === null || value === undefined) {
        return fallback;
      }
      if (typeof value === 'string') {
        const trimmed = value.trim();
        return trimmed || fallback;
      }
      return String(value);
    },
    goToProfileAvatar() {
      this.$router.push('/aprons/profile/avatar');
    },
  },
};
</script>
