/*
 * Copyright (c)  2025.5.24
 * Henry Zhao
 * araneae_front  -  California Beans (HollowData.com)
 * index.js
 * Last Modified: 2025-05-19 22:18:12  -  Davis, CA
 *
 * All rights reserved. Unauthorized copying of this file, via any medium,
 * is strictly prohibited unless prior written permission is obtained.
 */

import {createRouter, createWebHistory} from 'vue-router';

const routes = [

    // Auth
    {path: '/', redirect: '/aprons/workplaces'},
    {path: '/login', component: () => import('../views/Auth/Login.vue')},
    {path: '/oauth/callback', component: () => import('../views/Auth/OAuthCallback.vue')},
    {path: '/logout', component: () => import('../views/Auth/Logout.vue')},
    {path: '/register', component: () => import('../views/Auth/Register.vue')},
    {path: '/profile', component: () => import('../views/Profile/Index.vue')},
    {path: '/profile/avatar', component: () => import('../views/Profile/Avatar.vue')},

    // Aprons
    {
        path: '/aprons',
        component: () => import('../views/Aprons/Aprons.vue'),
        meta: {requiresAuth: true}
    },
    // Aprons.Help
    {
        path: '/aprons/help',
        component: () => import('../views/Aprons/ApronsHelp.vue'),
        meta: {requiresAuth: true}
    },
    // Aprons.About
    {
        path: '/aprons/about',
        component: () => import('../views/Aprons/ApronsAbout.vue'),
        meta: {requiresAuth: true}
    },
    // Aprons.Settings
    {
        path: '/aprons/settings',
        component: () => import('../views/Aprons/ApronsSettings.vue'),
        meta: {requiresAuth: true}
    },
    //Aprons.Node
    {
        name: 'apronsNode',
        path: '/aprons/nodes',
        component: () => import('../views/Aprons/ApronsNodes.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: 'apronsNodeCreate',
        path: '/aprons/node/create',
        component: () => import('../views/Aprons/ApronsNodesCreate.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: 'node',
        path: '/aprons/nodes/:id',
        component: () => import('../views/Nodes/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: 'NodeSettings',
        path: '/aprons/nodes/:id/settings',
        component: () => import('../views/Nodes/Settings/Index.vue'),
        meta: {requiresAuth: true}
    },
    // Aprons.Workplaces
    {
        name: 'apronsWorkplaces',
        path: '/aprons/workplaces',
        component: () => import('../views/Aprons/ApronsWorkplaces.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: 'apronsWorkplacesCreate',
        path: '/aprons/workplaces/create',
        component: () => import('../views/Aprons/ApronsWorkplacesCreate.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: 'workplace',
        path: '/aprons/workplaces/:id',
        component: () => import('../views/Workplaces/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: 'workplaceProjects',
        path: '/aprons/workplaces/:id/projects',
        component: () => import('../views/Workplaces/Projects/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: 'workplaceProjectCreate',
        path: '/aprons/workplaces/:id/projects/create',
        component: () => import('../views/Workplaces/Projects/Create.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: 'workplaceTasks',
        path: '/aprons/workplaces/:id/tasks',
        component: () => import('../views/Workplaces/Tasks/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: 'workplaceSchedule',
        path: '/aprons/workplaces/:id/schedules',
        component: () => import('../views/Workplaces/Schedules/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: 'workplaceScheduleCreate',
        path: '/aprons/workplaces/:id/schedules/create',
        component: () => import('../views/Workplaces/Schedules/Create.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: 'workplaceAnalysis',
        path: '/aprons/workplaces/:id/AnalyticsAndLogging',
        component: () => import('../views/Workplaces/Analysis/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        path: '/aprons/workplaces/:id/settings',
        component: () => import('../views/Workplaces/Settings/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        path: '/aprons/workplaces/:id/tasks',
        component: () => import('../views/Workplaces/Tasks/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        path: '/aprons/workplaces/:id/tasks/create',
        component: () => import('../views/Workplaces/Tasks/Create.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: 'workplaceTaskSetting',
        path: '/aprons/workplaces/:id/tasks/:taskId/settings',
        component: () => import('../views/Workplaces/Tasks/Setting.vue'),
        meta: {requiresAuth: true}
    },
    // Aprons.Projects
    {
        path: '/aprons/projects',
        component: () => import('../views/Aprons/ApronsProjects.vue'),
        meta: {requiresAuth: true}
    },
    {
        path: '/aprons/projects/:id',
        name: 'project',
        component: () => import('../views/Projects/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        path: '/aprons/projects/:id/repo',
        component: () => import('../views/Projects/Repo/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        path: '/aprons/projects/:id/repo/create',
        component: () => import('../views/Projects/Repo/Create.vue'),
        meta: {requiresAuth: true}
    },

    {
        path: "/aprons/projects/:id/settings",
        component: () => import('../views/Projects/Setting/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: 'projectVersionSetting',
        path: '/aprons/projects/:id/versions/:versionId/settings',
        component: () => import('../views/Projects/Versions/Setting.vue'),
        meta: {requiresAuth: true}
    },
    {
        path: "/aprons/projects/:id/distribute",
        component: () => import('../views/Projects/Distribute/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        path: '/aprons/projects/:id/distribute/order',
        component: () => import('../views/Projects/Distribute/Order.vue'),
        meta: {requiresAuth: true}
    },
    // Aprons.Teams
    {
        name: "apronsTeams",
        path: '/aprons/teams',
        component: () => import('../views/Aprons/ApronsTeams.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: "apronsTeamsCreate",
        path: '/aprons/teams/create',
        component: () => import('../views/Aprons/ApronsTeamsCreate.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: "team",
        path: '/aprons/teams/:id',
        component: () => import('../views/Teams/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: "apronsTeamSettings",
        path: '/aprons/teams/:id/settings',
        component: () => import('../views/Teams/Setting/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: "apronsFavorites",
        path: '/aprons/favorites',
        component: () => import('../views/Aprons/ApronsFavorites.vue'),
        meta: {requiresAuth: true}
    },
    {
        path: '/projects/index',
        component: () => import('../views/Projects/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: 'schedules',
        path: '/aprons/schedule/:id',
        component: () => import('../views/Schedules/Index.vue'),
        meta: {requiresAuth: true}
    },
    {
        name: 'schedulesEdit',
        path: '/aprons/schedule/:id/edit',
        component: () => import('../views/Schedules/Edit/Index.vue'),
        meta: {requiresAuth: true}
    },
];

const router = createRouter({
    history: createWebHistory(),
    routes,
});

function isAuthenticated() {
    // 假设你存储了一个 token 或其他标识用户已登录的标志
    return !!localStorage.getItem('token');
}

router.beforeEach((to, from, next) => {
    if (to.meta.requiresAuth && !isAuthenticated()) {
        next('/login');
    } else if (to.path === '/login' && isAuthenticated()) {
        next('/');
    } else {
        next();
    }
});

export default router;
