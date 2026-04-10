<template>
  <div class="storage-management">
    <el-card v-loading="loading">
      <template #header>
        <h3>存储管理</h3>
      </template>

      <el-descriptions :column="2" border>
        <el-descriptions-item label="存储类型">
          <el-tag type="info">{{ storageInfo.storageType || 'local' }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="数据路径">
          {{ storageInfo.dataPath || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="总文件数">
          {{ storageInfo.totalFiles }}
        </el-descriptions-item>
        <el-descriptions-item label="总大小">
          {{ formatFileSize(storageInfo.totalSize) }}
        </el-descriptions-item>
        <el-descriptions-item label="可用空间">
          {{ formatFileSize(storageInfo.freeSpace) }}
        </el-descriptions-item>
        <el-descriptions-item label="使用率">
          <el-tag type="success">{{ storageInfo.usagePercent.toFixed(2) }}%</el-tag>
        </el-descriptions-item>
      </el-descriptions>

      <el-divider />

      <div class="storage-tips">
        <el-alert
          title="存储说明"
          type="info"
          :closable="false"
        >
          <p>• 当前页面展示后端实际返回的存储配置与监控状态</p>
          <p>• 数据库文件：{{ storageInfo.dataPath }}/filecodebox.db</p>
          <p>• 上传文件目录：{{ storageInfo.dataPath }}/uploads/</p>
          <p>• 当前存储标识：{{ storageInfo.current || storageInfo.storageType }}</p>
        </el-alert>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { adminApi } from '@/api/admin'

const loading = ref(false)

const storageInfo = reactive({
  current: '',
  storageType: '',
  dataPath: '',
  totalFiles: 0,
  totalSize: 0,
  freeSpace: 0,
  usagePercent: 0
})

const formatFileSize = (bytes: number): string => {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const fetchStorageInfo = async () => {
  loading.value = true
  try {
    const [infoRes, statusRes] = await Promise.all([
      adminApi.getStorageInfo(),
      adminApi.getStorageStatus()
    ])

    if (infoRes.code === 200 && infoRes.data) {
      storageInfo.current = infoRes.data.current || ''
      storageInfo.storageType = infoRes.data.current || ''
      storageInfo.dataPath =
        infoRes.data.storage_config?.storage_path ||
        infoRes.data.storage_details?.[infoRes.data.current]?.storage_path ||
        ''
    }

    if (statusRes.code === 200 && statusRes.data) {
      storageInfo.storageType = statusRes.data.storage_type || storageInfo.storageType
      storageInfo.totalFiles = statusRes.data.file_count || 0
      storageInfo.totalSize = statusRes.data.used_space || 0
      storageInfo.freeSpace = statusRes.data.free_space || 0
      storageInfo.usagePercent = statusRes.data.usage_percent || 0
    }
  } catch (error) {
    console.error('获取存储信息失败:', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchStorageInfo()
})
</script>

<style scoped>
.storage-management {
  padding: 0;
}

.storage-tips {
  margin-top: 20px;
}

.storage-tips p {
  margin: 5px 0;
  line-height: 1.8;
}
</style>
