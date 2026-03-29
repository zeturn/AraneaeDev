/*
 * Copyright (c)  2025.5.24
 * Henry Zhao
 * araneae_front  -  California Beans (HollowData.com)
 * ApiService.js
 * Last Modified: 2025-05-19 22:18:46  -  Davis, CA
 *
 * All rights reserved. Unauthorized copying of this file, via any medium,
 * is strictly prohibited unless prior written permission is obtained.
 */

import axios from 'axios';


const apiFlavor = (import.meta.env.VITE_API_FLAVOR || 'django').toLowerCase();
const isGoApi = apiFlavor === 'go';
const backendBase = import.meta.env.VITE_BACKEND_BASE_URL || (isGoApi ? 'http://localhost:8180' : 'http://localhost:8107');
const apiClient = axios.create({
    baseURL: isGoApi ? `${backendBase}/api/v1` : `${backendBase}/api`,
    withCredentials: true,  // 重要：允许跨域 cookie 传输
    headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
        'X-CSRFToken': localStorage.getItem('csrf_token') || '', // 添加CSRF令牌
    },
});

const setCsrfToken = async () => {
    if (isGoApi) {
        return '';
    }
    try {
        const token = localStorage.getItem('token');
        const response = await axios.get(`${backendBase}/api/csrf-token/`, {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        const csrf_token = response.data.csrfToken;
        localStorage.setItem('csrf_token', csrf_token);
        document.cookie = `csrftoken=${csrf_token}; path=/; SameSite=Strict`;
        console.info('CSRF token set:', csrf_token);
        return csrf_token; // 返回新的 CSRF 令牌
    } catch (error) {
        console.error('Error fetching CSRF token:', error);
        throw error; // 如果获取 CSRF 令牌失败，则抛出错误
    }
};

apiClient.interceptors.request.use(async config => {
    if (!isGoApi) {
        try {
            // 在每个请求之前确保获取到最新的 CSRF 令牌
            const csrfToken = await setCsrfToken();
            config.headers['X-CSRFToken'] = csrfToken; // 设置最新的 CSRF 令牌
        } catch (error) {
            console.error('Error setting CSRF token:', error);
        }
    }

    const token = localStorage.getItem('token');
    if (token) {
        config.headers.Authorization = `Bearer ${token}`;
    }

    return config;
});


apiClient.interceptors.request.use(config => {
    const token = localStorage.getItem('token');
    if (token) {
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});

const parseOrderPayload = order => {
    if (!order) {
        return null;
    }
    if (typeof order === 'string') {
        try {
            return JSON.parse(order);
        } catch (_) {
            return null;
        }
    }
    if (typeof order === 'object') {
        return order;
    }
    return null;
};

const normalizeGoSchedule = (schedule, workplaceId = 'go-workspace') => {
    const parsedOrder = parseOrderPayload(schedule?.order);
    const fallbackOrder = {
        name: schedule?.name || '',
        schedule: [
            {
                task_id: schedule?.task_id || '',
                name: schedule?.name || '',
                project_id: schedule?.project_id || '',
                node: [schedule?.node_queue || 'default'],
                crons: schedule?.cron_expr || '',
            },
        ],
    };

    return {
        ...schedule,
        mode: schedule?.mode || 'recurring',
        workplace: schedule?.workplace || workplaceId,
        updated_at: schedule?.updated_at || schedule?.created_at,
        order: parsedOrder || fallbackOrder,
    };
};

const buildGoSchedulePayload = schedule => {
    const parsedOrder = parseOrderPayload(schedule?.order);
    const firstStep = parsedOrder?.schedule?.[0] || {};

    return {
        name: schedule?.name || firstStep?.name || 'schedule',
        description: schedule?.description || '',
        mode: schedule?.mode || 'recurring',
        enabled: schedule?.enabled !== false,
        task_id: schedule?.task_id || firstStep?.task_id || undefined,
        project_id: schedule?.project_id || firstStep?.project_id || undefined,
        version_id: schedule?.version_id || undefined,
        entry_command: schedule?.entry_command || undefined,
        cron_expr: schedule?.cron_expr || firstStep?.crons || undefined,
        node_queue: schedule?.node_queue || firstStep?.node?.[0] || 'default',
        order: schedule?.order || parsedOrder || undefined,
    };
};

const ApiService = {
    getUsers() {
        return apiClient.get('/users/');
    },
    getUser(userId) {
        return apiClient.get(`/users/${userId}/`);
    },
    getProfile() {
        return apiClient.get('/profile/');
    },
    getProfileAvatar() {
        return apiClient.get('/profile/avatar/');
    },
    updateProfileAvatar(formData) {
        //setCsrfToken()
        return apiClient.put('/profile/avatar/', formData, {
            headers: {
                'Content-Type': 'multipart/form-data',
            },
        });
    },
    // Teams
    getTeam(teamId) {
        return apiClient.get(`/teams/${teamId}/`);
    },
    getMyTeams() {
        return apiClient.get('/teams/my_teams/');
    },
    createTeam(team) {
        return apiClient.post('/teams/', team);
    },
    updateTeam(teamId, team) {
        return apiClient.put(`/teams/${teamId}/`, team);
    },
    deleteTeam(teamId) {
        return apiClient.delete(`/teams/${teamId}/`);
    },
    getTeamMembers(teamId) {
        return apiClient.get(`/teams/${teamId}/members/`);
    },
    addTeamMembers(teamId, userIds) {
        return apiClient.post(
            `/teams/${teamId}/add_members/`,
            {user_ids: userIds}
        );
    },
    // Node
    registerNodes(ip, name) {
        return apiClient.post('/nodes/register/', {ip, name});
    },
    discoverNodes(params = {}) {
        return apiClient.get('/nodes/discover/', {params});
    },
    getNodesList() {
        return apiClient.get('/nodes/');
    },
    getNode(nodeId) {
        return apiClient.get(`/nodes/${nodeId}/`);
    },
    updateNode(nodeId, node) {
        return apiClient.put(`/nodes/${nodeId}/`, node);
    },
    deleteNode(nodeId) {
        return apiClient.delete(`/nodes/${nodeId}`);
    },
    getNodeStatus(nodeId) {
        return apiClient.get(`/nodes/${nodeId}/status/`);
    },
    getNodeCapabilities(nodeId) {
        // GET /api/nodes/{id}/capabilities/ — 读取已存储的运行时能力列表
        return apiClient.get(`/nodes/${nodeId}/capabilities/`);
    },
    refreshNodeCapabilities(nodeId) {
        // POST /api/nodes/{id}/refresh_capabilities/ — 主动拉取执行节点并刷新
        return apiClient.post(`/nodes/${nodeId}/refresh_capabilities/`);
    },
    getNodeInstallers(nodeId) {
        // GET /api/nodes/{id}/installers/ — 获取可安装运行时列表
        return apiClient.get(`/nodes/${nodeId}/installers/`);
    },
    installRuntime(nodeId, key) {
        // POST /api/nodes/{id}/install_runtime/ — 发起安装任务，body: {key}
        return apiClient.post(`/nodes/${nodeId}/install_runtime/`, { key });
    },
    getInstallStatus(nodeId, jobId) {
        // GET /api/nodes/{id}/install_status/{jobId}/ — 轮询安装进度
        return apiClient.get(`/nodes/${nodeId}/install_status/${jobId}/`);
    },
    getWorkplaces() {
        return apiClient.get('/workplaces/');
    },
    addWorkplaceTeams(workplaceId, teamIds) {
        return apiClient.post(
            `/workplaces/${workplaceId}/add_teams/`,
            {team_ids: teamIds}
        );
    },
    addWorkplacePeople(workplaceId, userIds) {
        return apiClient.post(
            `/workplaces/${workplaceId}/add_people/`,
            {user_ids: userIds}
        );
    },
    getWorkplaceProjects(workplaceId) {
        if (isGoApi) {
            return apiClient.get('/projects');
        }
        return apiClient.get(`/workplaces/${workplaceId}/workplaces_projects/`);
    },
    getWorkplaceTaskRecords(workplaceId) {
        if (isGoApi) {
            return apiClient.get('/runs').then(resp => ({
                ...resp,
                data: {
                    records: resp.data.records || [],
                    count: resp.data.count || 0,
                },
            }));
        }
        return apiClient.get(`/workplaces/${workplaceId}/workplace_taskrecords/`);
    },
    getWorkplaceSchedules(workplaceId) {
        if (isGoApi) {
            return apiClient.get('/schedules').then(resp => ({
                ...resp,
                data: Array.isArray(resp.data)
                    ? resp.data.map(item => normalizeGoSchedule(item, workplaceId))
                    : [],
            }));
        }
        return apiClient.get(`/workplaces/${workplaceId}/workplaces_schedules/`);
    },
    getWorkplaceTasks(workplaceId) {
        if (isGoApi) {
            return apiClient.get('/tasks').then(resp => ({
                ...resp,
                data: {
                    tasks: Array.isArray(resp.data) ? resp.data : [],
                },
            }));
        }
        return apiClient.get(`/workplaces/${workplaceId}/workplaces_tasks/`);
    },
    createWorkplace(workplace) {
        return apiClient.post('/workplaces/', workplace);
    },
    getMyWorkplaces() {
        return apiClient.get('/workplaces/my_workplaces/');
    },
    getWorkplace(workplaceId) {
        return apiClient.get(`/workplaces/${workplaceId}/`);
    },
    updateWorkplace(workplaceId, workplace) {
        return apiClient.put(`/workplaces/${workplaceId}/`, workplace);
    },
    deleteWorkplace(workplaceId) {
        return apiClient.delete(`/workplaces/${workplaceId}/`);
    },
    // Project
    getMyProjects() {
        if (isGoApi) {
            return apiClient.get('/projects');
        }
        return apiClient.get(`/projects/my_projects/`);
    },
    getProject(projectId) {
        if (isGoApi) {
            return apiClient.get(`/projects/${projectId}`);
        }
        return apiClient.get(`/projects/${projectId}/`);
    },
    createProject(project) { // 创建项目
        if (isGoApi) {
            return apiClient.post('/projects', {
                name: project?.name || project?.title || 'untitled-project',
            });
        }
        return apiClient.post(`/projects/`, project);
    },
    updateProject(projectId, project) {
        return apiClient.put(`/projects/${projectId}/`, project);
    },
    deleteProject(projectId) {
        return apiClient.delete(`/projects/${projectId}/`);
    },
    getVersionsFromProject(projectId) {
        if (isGoApi) {
            return apiClient.get(`/projects/${projectId}/versions`);
        }
        return apiClient.get(`/projects/${projectId}/versions/`);
    },
    getReposFromProject(projectId) {
        return apiClient.get(`/projects/${projectId}/get_repo/`);
    },
    uploadCode(formData) {
        if (isGoApi) {
            const projectId = formData.get('project_id');
            if (!projectId) {
                throw new Error('project_id is required for Go API upload');
            }
            return apiClient.post(`/projects/${projectId}/upload`, formData, {
                withCredentials: false,
                headers: {
                    'Content-Type': 'multipart/form-data',
                },
            });
        }
        return apiClient.post(`/upload-script/`, formData, {
            withCredentials: true,
            headers: {
                'Content-Type': 'multipart/form-data',
                'X-CSRFToken': localStorage.getItem('csrf_token'), // 添加CSRF令牌
            },
        });
    },
    //  Schedule
    getSchedules() {
        if (isGoApi) {
            return apiClient.get('/schedules').then(resp => ({
                ...resp,
                data: Array.isArray(resp.data)
                    ? resp.data.map(item => normalizeGoSchedule(item))
                    : [],
            }));
        }
        return apiClient.get('/schedules/');
    },
    getSchedule(scheduleId) {
        if (isGoApi) {
            return apiClient.get(`/schedules/${scheduleId}`).then(resp => ({
                ...resp,
                data: normalizeGoSchedule(resp.data),
            }));
        }
        return apiClient.get(`/schedules/${scheduleId}/`);
    },
    createSchedule(schedule) { // 创建日程
        if (isGoApi) {
            return apiClient.post('/schedules', buildGoSchedulePayload(schedule)).then(resp => ({
                ...resp,
                data: normalizeGoSchedule(resp.data, schedule?.workplace || 'go-workspace'),
            }));
        }
        return apiClient.post('/create-task-chain/', schedule);
    },
    updateSchedule(scheduleId, schedule) {
        if (isGoApi) {
            return apiClient.put(`/schedules/${scheduleId}`, buildGoSchedulePayload(schedule)).then(resp => ({
                ...resp,
                data: normalizeGoSchedule(resp.data, schedule?.workplace || 'go-workspace'),
            }));
        }
        return apiClient.put(`/schedules/${scheduleId}/`, schedule);
    },
    deleteSchedule(scheduleId) {
        if (isGoApi) {
            return apiClient.delete(`/schedules/${scheduleId}`);
        }
        return apiClient.delete(`/schedules/${scheduleId}/`);
    },
    enableSchedule(scheduleId) {
        if (isGoApi) {
            return apiClient.post(`/schedules/${scheduleId}/enable`);
        }
        return apiClient.post(`/schedules/${scheduleId}/enable/`);
    },
    disableSchedule(scheduleId) {
        if (isGoApi) {
            return apiClient.post(`/schedules/${scheduleId}/disable`);
        }
        return apiClient.post(`/schedules/${scheduleId}/disable/`);
    },
    createTask(task) {
        if (isGoApi) {
            return apiClient.post('/tasks', {
                name: task.name,
                project_id: task.project_id,
                version_id: task.version_id,
                entry_command: task.entry_command || 'bash run.sh',
                cron_expr: task.cron_expr || '',
                node_queue: task.node_queue || 'default',
            });
        }
        return apiClient.post('/tasks/', task);
    },
    updateTask(taskId, task) {
        return apiClient.put(`/tasks/${taskId}/`, task);
    },
    deleteTask(taskId) {
        return apiClient.delete(`/tasks/${taskId}/`);
    },
    //  Task
    getTasks() {
        if (isGoApi) {
            return apiClient.get('/tasks');
        }
        return apiClient.get('/tasks/');
    },
    //  Account
    login(credentials) {
        if (isGoApi) {
            return apiClient.post('/auth/login', credentials);
        }
        return apiClient.post('/token/', credentials);
    },
    logout() {
        if (isGoApi) {
            localStorage.removeItem('token');
            localStorage.removeItem('refresh_token');
            localStorage.removeItem('csrf_token');
            return Promise.resolve({ data: { ok: true } });
        }
        return apiClient.post('/logout/');
    }
};

export default ApiService;
