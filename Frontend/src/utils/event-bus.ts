/*
 * Copyright (c)  2025.5.18
 * Henry Zhao
 * araneae_front  -  California Beans (HollowData.com)
 * event-bus.ts
 * Last Modified: 2025-05-18 23:16:58  -  Davis, CA
 *
 * All rights reserved. Unauthorized copying of this file, via any medium,
 * is strictly prohibited unless prior written permission is obtained.
 */

import mitt from 'mitt'

type NotificationType = 'info' | 'success' | 'error' | 'warning' | 'special'

export interface NotifyEvent {
    type: NotificationType
    title: string
    message: string
    duration?: number // 可选
}

type Events = {
    'LayoutTabs:closeTab'?: string
    'LayoutTabs:setTabTitle': string
    'notify': NotifyEvent // 新增
}

const EventBus = mitt<Events>()
export default EventBus
