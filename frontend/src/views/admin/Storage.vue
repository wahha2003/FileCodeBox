<template>
  <div class="storage-management">
    <el-card class="overview-card" v-loading="loading">
      <template #header>
        <div class="card-header">
          <div>
            <h3>存储管理</h3>
            <p>查看当前状态，配置并切换存储后端（本地 / S3 / 七牛云 / 又拍云）。</p>
          </div>
          <el-button @click="fetchStorageInfo">刷新</el-button>
        </div>
      </template>

      <el-descriptions :column="2" border>
        <el-descriptions-item label="当前存储">
          <el-tag type="info">{{ storageLabel(storageInfo.current) }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="存储位置">
          {{ storageInfo.dataPath || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="总文件数">
          {{ storageInfo.totalFiles }}
        </el-descriptions-item>
        <el-descriptions-item label="总大小">
          {{ formatFileSize(storageInfo.totalSize) }}
        </el-descriptions-item>
        <el-descriptions-item label="可用空间">
          {{ storageInfo.freeSpace > 0 ? formatFileSize(storageInfo.freeSpace) : '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="使用率">
          <el-tag :type="storageInfo.usagePercent > 80 ? 'warning' : 'success'">
            {{ storageInfo.usagePercent.toFixed(2) }}%
          </el-tag>
        </el-descriptions-item>
      </el-descriptions>

      <el-divider />

      <el-alert
        title="操作说明"
        type="info"
        :closable="false"
      >
        <p>保存配置会写入后端 `configs/config.yaml`。</p>
        <p>测试连接和切换存储都会先保存当前表单，确保使用的是最新配置。</p>
        <p>切换后新的上传会写入对应的存储后端，已有历史文件不会自动迁移。</p>
      </el-alert>
    </el-card>

    <el-card class="config-card">
      <template #header>
        <div class="card-header">
          <div>
            <h3>存储配置</h3>
            <p>先保存配置，再测试连接或切换存储。</p>
          </div>
          <el-tag type="warning">已选 {{ storageLabel(selectedType) }}</el-tag>
        </div>
      </template>

      <el-form label-width="130px" class="storage-form">
        <el-form-item label="目标存储类型">
          <el-radio-group v-model="selectedType">
            <el-radio-button label="local">本地存储</el-radio-button>
            <el-radio-button label="s3">S3 存储</el-radio-button>
            <el-radio-button label="qiniu">七牛云</el-radio-button>
            <el-radio-button label="upyun">又拍云</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <!-- 本地存储配置 -->
        <template v-if="selectedType === 'local'">
          <el-form-item label="本地存储路径">
            <el-input
              v-model="form.local.storage_path"
              placeholder="./data/uploads"
              clearable
            />
          </el-form-item>
        </template>

        <!-- S3 存储配置 -->
        <template v-if="selectedType === 's3'">
          <el-form-item label="Bucket 名称">
            <el-input v-model="form.s3.bucket_name" placeholder="filecodebox" clearable />
          </el-form-item>
          <el-form-item label="Access Key ID">
            <el-input v-model="form.s3.access_key_id" placeholder="AKIA..." clearable />
          </el-form-item>
          <el-form-item label="Secret Access Key">
            <el-input
              v-model="form.s3.secret_access_key"
              type="password"
              show-password
              placeholder="请输入 S3 Secret"
              clearable
            />
          </el-form-item>
          <el-form-item label="Endpoint URL">
            <el-input
              v-model="form.s3.endpoint_url"
              placeholder="https://s3.amazonaws.com 或兼容 S3 的 Endpoint"
              clearable
            />
          </el-form-item>
          <el-form-item label="Hostname">
            <el-input
              v-model="form.s3.hostname"
              placeholder="当未填写 Endpoint URL 时，可填写主机名"
              clearable
            />
          </el-form-item>
          <el-form-item label="Region">
            <el-input v-model="form.s3.region_name" placeholder="us-east-1" clearable />
          </el-form-item>
          <el-form-item label="HTTP 代理">
            <el-input
              v-model="form.s3.proxy"
              placeholder="http://127.0.0.1:7890"
              clearable
            />
          </el-form-item>
        </template>

        <!-- 七牛云配置 -->
        <template v-if="selectedType === 'qiniu'">
          <el-form-item label="Access Key">
            <el-input v-model="form.qiniu.access_key" placeholder="七牛 Access Key" clearable />
          </el-form-item>
          <el-form-item label="Secret Key">
            <el-input
              v-model="form.qiniu.secret_key"
              type="password"
              show-password
              placeholder="七牛 Secret Key"
              clearable
            />
          </el-form-item>
          <el-form-item label="Bucket 名称">
            <el-input v-model="form.qiniu.bucket" placeholder="my-bucket" clearable />
          </el-form-item>
          <el-form-item label="CDN 域名">
            <el-input v-model="form.qiniu.domain" placeholder="cdn.example.com" clearable />
            <p class="form-tip">绑定到 Bucket 的 CDN 加速域名或源站域名</p>
          </el-form-item>
          <el-form-item label="区域">
            <el-select v-model="form.qiniu.region" placeholder="选择区域" clearable>
              <el-option label="华东-浙江" value="z0" />
              <el-option label="华北-河北" value="z1" />
              <el-option label="华南-广东" value="z2" />
              <el-option label="北美-洛杉矶" value="na0" />
              <el-option label="亚太-新加坡" value="as0" />
              <el-option label="华东-浙江2" value="cn-east-2" />
            </el-select>
          </el-form-item>
          <el-form-item label="使用 HTTPS">
            <el-switch v-model="form.qiniu.use_https" />
          </el-form-item>
          <el-form-item label="私有空间">
            <el-switch v-model="form.qiniu.private" />
            <p class="form-tip">如果是私有空间，生成的下载 URL 会带签名</p>
          </el-form-item>
        </template>

        <!-- 又拍云配置 -->
        <template v-if="selectedType === 'upyun'">
          <el-form-item label="服务名">
            <el-input v-model="form.upyun.bucket" placeholder="又拍云服务名（Bucket）" clearable />
          </el-form-item>
          <el-form-item label="操作员">
            <el-input v-model="form.upyun.operator" placeholder="操作员账号" clearable />
          </el-form-item>
          <el-form-item label="操作密码">
            <el-input
              v-model="form.upyun.password"
              type="password"
              show-password
              placeholder="操作员密码"
              clearable
            />
          </el-form-item>
          <el-form-item label="CDN 域名">
            <el-input v-model="form.upyun.domain" placeholder="cdn.example.com" clearable />
            <p class="form-tip">绑定到服务的 CDN 加速域名</p>
          </el-form-item>
          <el-form-item label="Token 防盗链密钥">
            <el-input
              v-model="form.upyun.secret"
              type="password"
              show-password
              placeholder="可选，用于生成防盗链 URL"
              clearable
            />
          </el-form-item>
        </template>

        <el-form-item>
          <div class="actions">
            <el-button type="primary" :loading="saving" @click="saveConfig()">
              保存配置
            </el-button>
            <el-button :loading="testing" @click="testConnection">
              测试连接
            </el-button>
            <el-button
              type="success"
              :loading="switching"
              @click="switchStorage"
            >
              切换为 {{ storageLabel(selectedType) }}
            </el-button>
          </div>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { adminApi } from '@/api/admin'

// 存储类型标签映射
const storageLabels: Record<string, string> = {
  local: '本地存储',
  s3: 'S3 存储',
  qiniu: '七牛云',
  upyun: '又拍云',
}
const storageLabel = (type: string) => storageLabels[type] || type

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const switching = ref(false)
const selectedType = ref<'local' | 's3' | 'qiniu' | 'upyun'>('local')

const storageInfo = reactive({
  current: '',
  dataPath: '',
  totalFiles: 0,
  totalSize: 0,
  freeSpace: 0,
  usagePercent: 0
})

const form = reactive({
  local: {
    storage_path: './data/uploads'
  },
  s3: {
    access_key_id: '',
    secret_access_key: '',
    bucket_name: '',
    endpoint_url: '',
    region_name: '',
    hostname: '',
    proxy: ''
  },
  qiniu: {
    access_key: '',
    secret_key: '',
    bucket: '',
    domain: '',
    region: '',
    use_https: true,
    private: false,
  },
  upyun: {
    bucket: '',
    operator: '',
    password: '',
    domain: '',
    secret: '',
  },
})

const formatFileSize = (bytes: number): string => {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

const applyStorageInfo = (data: any) => {
  const currentType = data?.current || 'local'
  storageInfo.current = currentType
  storageInfo.dataPath =
    data?.storage_details?.[currentType]?.storage_path ||
    data?.storage_config?.storage_path ||
    ''

  form.local.storage_path = data?.storage_config?.storage_path || './data/uploads'

  // S3 配置回填
  const s3 = data?.storage_config?.s3 || {}
  form.s3.access_key_id = s3.access_key_id || ''
  form.s3.secret_access_key = s3.secret_access_key || ''
  form.s3.bucket_name = s3.bucket_name || ''
  form.s3.endpoint_url = s3.endpoint_url || ''
  form.s3.region_name = s3.region_name || ''
  form.s3.hostname = s3.hostname || ''
  form.s3.proxy = s3.proxy || ''

  // 七牛云配置回填
  const qiniu = data?.storage_config?.qiniu || {}
  form.qiniu.access_key = qiniu.access_key || ''
  form.qiniu.secret_key = qiniu.secret_key || ''
  form.qiniu.bucket = qiniu.bucket || ''
  form.qiniu.domain = qiniu.domain || ''
  form.qiniu.region = qiniu.region || ''
  form.qiniu.use_https = qiniu.use_https !== false
  form.qiniu.private = qiniu.private || false

  // 又拍云配置回填
  const upyun = data?.storage_config?.upyun || {}
  form.upyun.bucket = upyun.bucket || ''
  form.upyun.operator = upyun.operator || ''
  form.upyun.password = upyun.password || ''
  form.upyun.domain = upyun.domain || ''
  form.upyun.secret = upyun.secret || ''
}

const fetchStorageInfo = async (syncSelectedType = false) => {
  loading.value = true
  try {
    const [infoRes, statusRes] = await Promise.all([
      adminApi.getStorageInfo(),
      adminApi.getStorageStatus()
    ])

    if (infoRes.code === 200 && infoRes.data) {
      applyStorageInfo(infoRes.data)
      if (syncSelectedType) {
        const currentType = infoRes.data.current || 'local'
        selectedType.value = (['s3', 'qiniu', 'upyun'].includes(currentType) ? currentType : 'local') as any
      }
    }

    if (statusRes.code === 200 && statusRes.data) {
      storageInfo.totalFiles = statusRes.data.file_count || 0
      storageInfo.totalSize = statusRes.data.used_space || 0
      storageInfo.freeSpace = statusRes.data.free_space || 0
      storageInfo.usagePercent = statusRes.data.usage_percent || 0
      if (!storageInfo.current) {
        storageInfo.current = statusRes.data.storage_type || 'local'
      }
    }
  } catch (error) {
    console.error('获取存储信息失败:', error)
  } finally {
    loading.value = false
  }
}

const buildPayload = () => {
  if (selectedType.value === 'local') {
    const storagePath = form.local.storage_path.trim()
    if (!storagePath) {
      throw new Error('本地存储路径不能为空')
    }
    return { type: 'local', config: { storage_path: storagePath } }
  }

  if (selectedType.value === 's3') {
    if (!form.s3.bucket_name.trim()) throw new Error('Bucket 名称不能为空')
    if (!form.s3.access_key_id.trim()) throw new Error('Access Key ID 不能为空')
    if (!form.s3.secret_access_key.trim()) throw new Error('Secret Access Key 不能为空')
    return {
      type: 's3',
      config: {
        s3: {
          access_key_id: form.s3.access_key_id.trim(),
          secret_access_key: form.s3.secret_access_key.trim(),
          bucket_name: form.s3.bucket_name.trim(),
          endpoint_url: form.s3.endpoint_url.trim(),
          region_name: form.s3.region_name.trim(),
          hostname: form.s3.hostname.trim(),
          proxy: form.s3.proxy.trim(),
        }
      }
    }
  }

  if (selectedType.value === 'qiniu') {
    if (!form.qiniu.access_key.trim()) throw new Error('七牛 Access Key 不能为空')
    if (!form.qiniu.secret_key.trim()) throw new Error('七牛 Secret Key 不能为空')
    if (!form.qiniu.bucket.trim()) throw new Error('七牛 Bucket 不能为空')
    return {
      type: 'qiniu',
      config: {
        qiniu: {
          access_key: form.qiniu.access_key.trim(),
          secret_key: form.qiniu.secret_key.trim(),
          bucket: form.qiniu.bucket.trim(),
          domain: form.qiniu.domain.trim(),
          region: form.qiniu.region,
          use_https: form.qiniu.use_https,
          private: form.qiniu.private,
        }
      }
    }
  }

  if (selectedType.value === 'upyun') {
    if (!form.upyun.bucket.trim()) throw new Error('又拍云服务名不能为空')
    if (!form.upyun.operator.trim()) throw new Error('又拍云操作员不能为空')
    if (!form.upyun.password.trim()) throw new Error('又拍云操作密码不能为空')
    return {
      type: 'upyun',
      config: {
        upyun: {
          bucket: form.upyun.bucket.trim(),
          operator: form.upyun.operator.trim(),
          password: form.upyun.password.trim(),
          domain: form.upyun.domain.trim(),
          secret: form.upyun.secret.trim(),
        }
      }
    }
  }

  throw new Error('未知的存储类型')
}

const persistConfig = async (silent = false) => {
  const payload = buildPayload()
  const response = await adminApi.updateStorageConfig(payload)
  if (response.code !== 200) {
    throw new Error(response.message || '保存配置失败')
  }
  if (!silent) {
    ElMessage.success('存储配置已保存')
  }
  await fetchStorageInfo(false)
}

const saveConfig = async (silent = false) => {
  saving.value = true
  try {
    await persistConfig(silent)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '保存配置失败')
  } finally {
    saving.value = false
  }
}

const testConnection = async () => {
  testing.value = true
  try {
    await persistConfig(true)
    const response = await adminApi.testStorageConnection(selectedType.value)
    if (response.code !== 200) {
      throw new Error(response.message || '连接测试失败')
    }
    ElMessage.success(`${storageLabel(selectedType.value)} 连接测试成功`)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '连接测试失败')
  } finally {
    testing.value = false
  }
}

const switchStorage = async () => {
  switching.value = true
  try {
    await persistConfig(true)
    const response = await adminApi.switchStorage(selectedType.value)
    if (response.code !== 200) {
      throw new Error(response.message || '切换存储失败')
    }
    ElMessage.success(`已切换到 ${storageLabel(selectedType.value)}`)
    await fetchStorageInfo(true)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '切换存储失败')
  } finally {
    switching.value = false
  }
}

onMounted(() => {
  fetchStorageInfo(true)
})
</script>

<style scoped>
.storage-management {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.card-header h3 {
  margin: 0 0 6px;
}

.card-header p {
  margin: 0;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}

.overview-card :deep(.el-alert__content p) {
  margin: 6px 0;
  line-height: 1.7;
}

.storage-form {
  max-width: 860px;
}

.form-tip {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

@media (max-width: 768px) {
  .card-header {
    flex-direction: column;
    align-items: stretch;
  }

  .actions {
    width: 100%;
  }

  .actions :deep(.el-button) {
    flex: 1 1 100%;
  }
}
</style>
