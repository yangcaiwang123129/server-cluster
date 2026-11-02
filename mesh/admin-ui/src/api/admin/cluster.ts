import request from '@/utils/request'
import type { ApiResponse, PageResult } from '@/types/api'
import type { ClusterNode, ClusterExecResp } from '@/types/admin'

// ── 集群管理 ────────────────────────────────────────────────────────────

/** 集群节点列表（自动从 etcd 同步）。 */
export function listClusterNodes(params: {
  pageIndex: number
  pageSize: number
  nodeName?: string
  nodeKind?: string
  online?: number
  nodeIdOrder?: string
  nodeNameOrder?: string
  nodeKindOrder?: string
  onlineOrder?: string
}) {
  return request<ApiResponse<PageResult<ClusterNode>>>({
    url: '/api/v1/gm/cluster',
    method: 'get',
    params
  })
}

/** 单个集群节点详情。 */
export function getClusterNode(nodeId: string) {
  return request<ApiResponse<ClusterNode>>({
    url: `/api/v1/gm/cluster/${encodeURIComponent(nodeId)}`,
    method: 'get'
  })
}

/** 更新运维侧 SSH 配置。 */
export function updateClusterNodeSSH(nodeId: string, data: {
  sshHost: string
  sshPort?: number
  sshUser: string
  sshKeyPath?: string
  deployDir: string
  remark?: string
}) {
  return request<ApiResponse<null>>({
    url: `/api/v1/gm/cluster/${encodeURIComponent(nodeId)}`,
    method: 'put',
    data
  })
}

/** 远程运维脚本执行。action: start/stop/restart/deploy。 */
export function execClusterNode(nodeId: string, action: string) {
  return request<ApiResponse<ClusterExecResp>>({
    url: `/api/v1/gm/cluster/${encodeURIComponent(nodeId)}/exec`,
    method: 'post',
    data: { action }
  })
}
