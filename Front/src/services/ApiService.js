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
    withCredentials: !isGoApi,
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

const normalizeGoVersionList = payload => {
    const list = Array.isArray(payload)
        ? payload
        : (Array.isArray(payload?.versions) ? payload.versions : []);

    return list.map(item => ({
        id: item?.id || item?.version_id || '',
        project_id: item?.project_id || '',
        version_hash: item?.version_hash || item?.id || item?.sha256 || '',
        release_date: item?.release_date || item?.created_at || '',
        file_name: item?.file_name || '',
        storage_path: item?.storage_path || '',
        sha256: item?.sha256 || '',
    }));
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

const emptyAvatarResponse = () => ({
    data: {avatar: null},
    headers: {'content-type': 'application/json'},
    status: 204,
    statusText: 'No Content',
    config: {},
});

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
        if (isGoApi) {
            return Promise.resolve(emptyAvatarResponse());
        }
        return apiClient.get('/profile/avatar/');
    },
    updateProfileAvatar(formData) {
        if (isGoApi) {
            return Promise.reject(new Error('Profile avatar update is not supported in Go API mode.'));
        }
        //setCsrfToken()
        return apiClient.put('/profile/avatar/', formData, {
            headers: {
                'Content-Type': 'multipart/form-data',
            },
        });
    },
    // Teams
    getTeam(teamId) {
        if (isGoApi) {
            return apiClient.get(`/teams/${teamId}`);
        }
        return apiClient.get(`/teams/${teamId}/`);
    },
    getMyTeams() {
        if (isGoApi) {
            return apiClient.get('/teams/my_teams').then(resp => ({
                ...resp,
                data: {
                    results: Array.isArray(resp?.data?.results) ? resp.data.results : [],
                    count: resp?.data?.count || 0,
                },
            }));
        }
        return apiClient.get('/teams/my_teams/');
    },
    createTeam(team) {
        if (isGoApi) {
            return apiClient.post('/teams', team);
        }
        return apiClient.post('/teams/', team);
    },
    updateTeam(teamId, team) {
        if (isGoApi) {
            return apiClient.put(`/teams/${teamId}`, team);
        }
        return apiClient.put(`/teams/${teamId}/`, team);
    },
    deleteTeam(teamId) {
        if (isGoApi) {
            return apiClient.delete(`/teams/${teamId}`);
        }
        return apiClient.delete(`/teams/${teamId}/`);
    },
    getTeamMembers(teamId) {
        if (isGoApi) {
            return apiClient.get(`/teams/${teamId}/members`);
        }
        return apiClient.get(`/teams/${teamId}/members/`);
    },
    addTeamMembers(teamId, userIds) {
        if (isGoApi) {
            return apiClient.post(
                `/teams/${teamId}/add_members`,
                {user_ids: userIds}
            );
        }
        return apiClient.post(
            `/teams/${teamId}/add_members/`,
            {user_ids: userIds}
        );
    },
    removeTeamMember(teamId, userId) {
        if (isGoApi) {
            return apiClient.delete(`/teams/${teamId}/members/${userId}`);
        }
        return apiClient.delete(`/teams/${teamId}/members/${userId}/`);
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
                description: project?.description || '',
                language: project?.language || '',
                command: project?.command || '',
            });
        }
        return apiClient.post(`/projects/`, project);
    },
    updateProject(projectId, project) {
        if (isGoApi) {
            return apiClient.put(`/projects/${projectId}`, {
                name: project?.name,
                description: project?.description,
                language: project?.language,
                command: project?.command,
            });
        }
        return apiClient.put(`/projects/${projectId}/`, project);
    },
    deleteProject(projectId) {
        if (isGoApi) {
            return apiClient.delete(`/projects/${projectId}`);
        }
        return apiClient.delete(`/projects/${projectId}/`);
    },
    getVersionsFromProject(projectId) {
        if (isGoApi) {
            return apiClient.get(`/projects/${projectId}/versions`).then(resp => ({
                ...resp,
                data: {
                    versions: normalizeGoVersionList(resp.data),
                },
            }));
        }
        return apiClient.get(`/projects/${projectId}/versions/`);
    },
    getReposFromProject(projectId) {
        if (isGoApi) {
            return apiClient.get(`/projects/${projectId}/versions`).then(resp => ({
                ...resp,
                data: {
                    versions: normalizeGoVersionList(resp.data),
                },
            }));
        }
        return apiClient.get(`/projects/${projectId}/get_repo/`);
    },
    getProjectVersion(projectId, versionId) {
        if (isGoApi) {
            return apiClient.get(`/projects/${projectId}/versions/${versionId}`);
        }
        return apiClient.get(`/versions/${versionId}/`);
    },
    updateProjectVersion(projectId, versionId, payload) {
        if (isGoApi) {
            return apiClient.put(`/projects/${projectId}/versions/${versionId}`, {
                file_name: payload?.file_name,
            });
        }
        return apiClient.put(`/versions/${versionId}/`, payload);
    },
    deleteProjectVersion(projectId, versionId) {
        if (isGoApi) {
            return apiClient.delete(`/projects/${projectId}/versions/${versionId}`);
        }
        return apiClient.delete(`/versions/${versionId}/`);
    },
    orderSourceDistribution(payload) {
        if (!isGoApi) {
            return apiClient.post('/source-distribution/', payload);
        }

        const projectId = payload?.project_id;
        const versionId = payload?.version;
        const targets = Array.isArray(payload?.targets) ? payload.targets : [];

        if (!projectId || !versionId) {
            return Promise.reject(new Error('project_id and version are required'));
        }
        if (targets.length === 0) {
            return Promise.reject(new Error('at least one target node is required'));
        }

        return Promise.resolve().then(async () => {
            const results = [];
            for (let i = 0; i < targets.length; i += 1) {
                const t = targets[i] || {};
                const nodeId = t.node_id || t.id || t;
                let nodeQueue = 'default';

                if (nodeId) {
                    try {
                        const nodeResp = await apiClient.get(`/nodes/${nodeId}/`);
                        nodeQueue = nodeResp?.data?.celery_queue || nodeQueue;
                    } catch (_) {
                        nodeQueue = 'default';
                    }
                }

                const createResp = await apiClient.post('/tasks', {
                    name: `distribute-${String(projectId).slice(0, 8)}-${String(nodeId || i)}`,
                    project_id: projectId,
                    version_id: versionId,
                    entry_command: 'bash run.sh',
                    cron_expr: '',
                    node_queue: nodeQueue,
                });

                const taskId = createResp?.data?.id;
                if (!taskId) {
                    throw new Error('failed to create distribution task');
                }

                const triggerResp = await apiClient.post(`/tasks/${taskId}/trigger`);
                results.push({
                    node_id: nodeId,
                    node_queue: nodeQueue,
                    task_id: taskId,
                    run_id: triggerResp?.data?.id || '',
                });
            }

            return {
                data: {
                    message: `Distribution triggered on ${results.length} node(s)`,
                    results,
                },
            };
        });
    },
    SourceDistributionList(projectId) {
        if (!isGoApi) {
            return apiClient.get(`/source-distribution/?project_id=${projectId}`);
        }

        return Promise.all([apiClient.get('/runs'), apiClient.get('/tasks')]).then(([runsResp, tasksResp]) => {
            const records = Array.isArray(runsResp?.data?.records) ? runsResp.data.records : [];
            const tasks = Array.isArray(tasksResp?.data) ? tasksResp.data : [];
            const taskMap = new Map(tasks.map(t => [t.id, t]));

            const data = records
                .map(r => ({ run: r, task: taskMap.get(r.task_id) }))
                .filter(item => item.task && String(item.task.project_id) === String(projectId))
                .map(item => ({
                    id: item.run.id,
                    version_hash: item.task.version_id,
                    project_name: `project-${String(projectId).slice(0, 8)}`,
                    deployed_at: item.run.finished_at || item.run.started_at || item.run.created_at,
                    is_active: item.run.status === 'queued' || item.run.status === 'running',
                    node: item.task.node_queue || 'default',
                    project: item.task.project_id,
                    version: item.task.version_id,
                }));

            return {
                ...runsResp,
                data,
            };
        });
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
                node_queue: task.node_queue || 'default',
            });
        }
        return apiClient.post('/tasks/', task);
    },
    updateTask(taskId, task) {
        if (isGoApi) {
            return apiClient.put(`/tasks/${taskId}`, task);
        }
        return apiClient.put(`/tasks/${taskId}/`, task);
    },
    deleteTask(taskId) {
        if (isGoApi) {
            return apiClient.delete(`/tasks/${taskId}`);
        }
        return apiClient.delete(`/tasks/${taskId}/`);
    },
    triggerTask(taskId) {
        if (isGoApi) {
            return apiClient.post(`/tasks/${taskId}/trigger`);
        }
        return apiClient.post(`/tasks/${taskId}/trigger/`);
    },
    //  Task
    getTasks() {
        if (isGoApi) {
            return apiClient.get('/tasks');
        }
        return apiClient.get('/tasks/');
    },
    getTask(taskId) {
        if (isGoApi) {
            return apiClient.get(`/tasks/${taskId}`);
        }
        return apiClient.get(`/tasks/${taskId}/`);
    },
    getTaskRuns(taskId) {
        if (isGoApi) {
            return apiClient.get(`/tasks/${taskId}/runs`);
        }
        return apiClient.get(`/tasks/${taskId}/runs/`);
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
