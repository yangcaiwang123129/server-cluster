import request from '@/utils/request'
import type { ApiResponse, PageQuery, PageResult } from '@/types/api'
import type { GmActivity, GmActivityCfg, GmActivityQuery } from '@/types/admin'

/** 运营活动管理：活动实例列表 / 模板配置 / 修改时间 / 提前结束。 */

export function listActivities(query: GmActivityQuery & Partial<PageQuery>) {
  return request<ApiResponse<PageResult<GmActivity>>>({
    url: '/api/v1/gm/activity',
    method: 'get',
    params: query
  })
}

export function listActivityConfigs() {
  return request<ApiResponse<GmActivityCfg[]>>({
    url: '/api/v1/gm/activity/configs',
    method: 'get'
  })
}

export function updateActivityTime(id: number, data: { startTime: number; endTime: number }) {
  return request<ApiResponse<GmActivity>>({
    url: '/api/v1/gm/activity/' + id + '/time',
    method: 'put',
    data
  })
}

export function endActivity(id: number) {
  return request<ApiResponse<GmActivity>>({
    url: '/api/v1/gm/activity/' + id + '/end',
    method: 'post'
  })
}
