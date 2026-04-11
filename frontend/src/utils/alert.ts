import { ElMessage } from 'element-plus'

export type AlertType = 'success' | 'error' | 'warning' | 'info'

declare global {
  interface Window {
    showAlert?: (message: string, type?: AlertType, duration?: number) => void
  }
}

export const showAlert = (
  message: string,
  type: AlertType = 'info',
  duration = 3000,
) => {
  if (typeof window !== 'undefined' && typeof window.showAlert === 'function') {
    window.showAlert(message, type, duration)
    return
  }

  ElMessage({
    message,
    type,
    duration,
    grouping: true,
  })
}
