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


const apiClient = axios.create({
    baseURL: 'http://localhost:8000/api',
    withCredentials: true,  // 重要：允许跨域 cookie 传输
    headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
        'X-CSRFToken': localStorage.getItem('csrf_token'), // 添加CSRF令牌
    },
});

const setCsrfToken = async () => {
    try {
        const token = localStorage.getItem('token');
        const response = await axios.get('http://localhost:8000/api/csrf-token/', {
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
    try {
        // 在每个请求之前确保获取到最新的 CSRF 令牌
        const csrfToken = await setCsrfToken();
        config.headers['X-CSRFToken'] = csrfToken; // 设置最新的 CSRF 令牌
    } catch (error) {
        console.error('Error setting CSRF token:', error);
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
    getNodesList() {
        return apiClient.get('/nodes/');
    },
    getNode(nodeId) {
        return apiClient.get(`/nodes/${nodeId}/`);
    },
    updateNode(nodeId, node) {
        return apiClient.put(`/nodes/${nodeId}/`, node);
    },
    orderSourceDistribution(data) {
        return apiClient.post(`/distribute_source/`, data);
    },
    SourceDistributionList(projectId) {
        return apiClient.get(`/distribute_source/project/${projectId}/`);
    },
    deleteNode(nodeId) {
        return apiClient.delete(`/nodes/${nodeId}`);
    },
    getNodeStatus(nodeId) {
        return apiClient.get(`/nodes/${nodeId}/status/`);
    },
    // Workplace
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
        return apiClient.get(`/workplaces/${workplaceId}/workplaces_projects/`);
    },
    getWorkplaceTaskRecords(workplaceId) {
        return apiClient.get(`/workplaces/${workplaceId}/workplace_taskrecords/`);
    },
    getWorkplaceSchedules(workplaceId) {
        return apiClient.get(`/workplaces/${workplaceId}/workplaces_schedules/`);
    },
    getWorkplaceTasks(workplaceId) {
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
        return apiClient.get(`/projects/my_projects/`);
    },
    getProject(projectId) {
        return apiClient.get(`/projects/${projectId}/`);
    },
    createProject(project) { // 创建项目
        return apiClient.post(`/projects/`, project);
    },
    updateProject(projectId, project) {
        return apiClient.put(`/projects/${projectId}/`, project);
    },
    deleteProject(projectId) {
        return apiClient.delete(`/projects/${projectId}/`);
    },
    getVersionsFromProject(projectId) {
        return apiClient.get(`/projects/${projectId}/versions/`);
    },
    getReposFromProject(projectId) {
        return apiClient.get(`/projects/${projectId}/get_repo/`);
    },
    uploadCode(formData) {
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
        return apiClient.get('/schedules/');
    },
    getSchedule(scheduleId) {
        return apiClient.get(`/schedules/${scheduleId}/`);
    },
    createSchedule(schedule) { // 创建日程
        return apiClient.post('/create-task-chain/', schedule);
    },
    updateSchedule(scheduleId, schedule) {
        return apiClient.put(`/schedules/${scheduleId}/`, schedule);
    },
    deleteSchedule(scheduleId) {
        return apiClient.delete(`/schedules/${scheduleId}/`);
    },
    createTask(task) {
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
        return apiClient.get('/tasks/');
    },
    //  Account
    login(credentials) {
        return apiClient.post('/token/', credentials);
    },
    logout() {
        return apiClient.post('/logout/');
    }
};

export default ApiService;
