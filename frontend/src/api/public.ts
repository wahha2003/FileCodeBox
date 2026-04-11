import { request } from '@/utils/request'
import type { ApiResponse } from '@/types/common'

export interface PublicConfig {
  name: string
  description: string
  uploadSize: number
  enableChunk: number
  openUpload: number
  expireStyle: string[]
  initialized?: boolean
}

export const publicApi = {
  // 获取公开配置
  getConfig: () => {
    return request<ApiResponse<PublicConfig>>({
      url: '/config',
      method: 'GET',
    })
  },

  // 检查系统初始化状态
  checkInitialization: () => {
    return request<{
      initialized: boolean
      message: string
    }>({
      url: '/setup/check',
      method: 'GET',
    })
  },
}
