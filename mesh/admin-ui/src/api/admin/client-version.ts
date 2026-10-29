import request from '@/utils/request'
import type { ApiResponse, PageQuery, PageResult, Id } from '@/types/api'
import type { ClientVersion, ClientVersionQuery } from '@/types/admin'

/** 客户端版本管理：安装包清单的增删改查。 */

export function listClientVersion(query: ClientVersionQuery & Partial<PageQuery>) {
  return request<ApiResponse<PageResult<ClientVersion>>>({
    url: '/api/v1/client-version',
    method: 'get',
    params: query
  })
}

export function getClientVersion(versionId: number) {
  return request<ApiResponse<ClientVersion>>({
    url: '/api/v1/client-version/' + versionId,
    method: 'get'
  })
}

export function addClientVersion(data: ClientVersion) {
  return request<ApiResponse<ClientVersion>>({
    url: '/api/v1/client-version',
    method: 'post',
    data
  })
}

export function updateClientVersion(data: ClientVersion & { versionId?: number }) {
  return request<ApiResponse<ClientVersion>>({
    url: '/api/v1/client-version/' + data.versionId,
    method: 'put',
    data
  })
}

export function delClientVersion(ids: Id[]) {
  return request<ApiResponse<null>>({
    url: '/api/v1/client-version',
    method: 'delete',
    data: { ids: ids.map(Number) }
  })
}

export function updateClientVersionStatus(data: { versionId: number; status: string }) {
  return request<ApiResponse<null>>({
    url: '/api/v1/client-version-status',
    method: 'put',
    data
  })
}
