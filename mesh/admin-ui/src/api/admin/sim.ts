import request from '@/utils/request'
import type { ApiResponse } from '@/types/api'
import type { SimConnectResult, SimLoginPayload, SimLoginResult, SimOptions, SimStepResult } from '@/types/admin'

/**
 * 客户端模拟器接口：admin 服务端代替 U3D 客户端走登录、连网关、首包与进出游戏。
 * 会话保存在 admin 内存中，每一步都要带上登录拿到的 sessionId。
 */

export function getSimOptions() {
  return request<ApiResponse<SimOptions>>({
    url: '/api/v1/gm/client/sim/options',
    method: 'get'
  })
}

export function simLogin(data: SimLoginPayload) {
  return request<ApiResponse<SimLoginResult>>({
    url: '/api/v1/gm/client/sim/login',
    method: 'post',
    data
  })
}

export function simConnect(sessionId: string) {
  return request<ApiResponse<SimConnectResult>>({
    url: '/api/v1/gm/client/sim/connect',
    method: 'post',
    data: { sessionId }
  })
}

export function simStep(sessionId: string, step: 'verify' | 'enter' | 'leave') {
  return request<ApiResponse<SimStepResult>>({
    url: '/api/v1/gm/client/sim/step',
    method: 'post',
    data: { sessionId, step }
  })
}

export function simClose(sessionId: string) {
  return request<ApiResponse<null>>({
    url: '/api/v1/gm/client/sim/close',
    method: 'post',
    data: { sessionId }
  })
}
