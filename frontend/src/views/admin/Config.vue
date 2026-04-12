<template>
  <div class="system-config">
    <el-card v-loading="loading">
      <template #header>
        <div class="card-header">
          <h3>系统配置</h3>
          <el-button type="primary" @click="saveConfig" :loading="saving">
            保存配置
          </el-button>
        </div>
      </template>

      <el-tabs v-model="activeTab">
        <el-tab-pane label="基础配置" name="basic">
          <el-form :model="configForm.base" label-width="140px" style="max-width: 680px">
            <el-form-item label="站点名称">
              <el-input v-model="configForm.base.name" />
            </el-form-item>

            <el-form-item label="站点描述">
              <el-input v-model="configForm.base.description" type="textarea" :rows="3" />
            </el-form-item>

            <el-form-item label="端口">
              <el-input-number v-model="configForm.base.port" :min="1" :max="65535" />
              <span class="field-hint">修改后需重启服务才会切换监听端口</span>
            </el-form-item>

            <el-form-item label="生产模式">
              <el-switch v-model="configForm.base.production" />
              <span class="field-hint">运行模式变更会在下次重启后完整生效</span>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="上传配置" name="upload">
          <el-form :model="configForm.transfer.upload" label-width="140px" style="max-width: 680px">
            <el-form-item label="开放上传">
              <el-switch v-model="configForm.transfer.upload.openupload" :active-value="1" :inactive-value="0" />
            </el-form-item>

            <el-form-item label="上传大小限制">
              <div class="size-editor">
                <el-input-number
                  v-model="sizeEditors.uploadsize.value"
                  :min="getSizeEditorMin('uploadsize')"
                  :step="getSizeEditorStep('uploadsize')"
                  :precision="getSizeEditorPrecision('uploadsize')"
                  controls-position="right"
                />
                <el-select
                  :model-value="sizeEditors.uploadsize.unit"
                  class="size-unit-select"
                  @change="handleUploadSizeUnitChange"
                >
                  <el-option v-for="unit in sizeUnitOptions" :key="unit" :label="unit" :value="unit" />
                </el-select>
                <span class="size-hint">当前约 {{ getSizePreview('uploadsize') }}</span>
              </div>
            </el-form-item>

            <el-form-item label="需要登录">
              <el-switch v-model="configForm.transfer.upload.requirelogin" :active-value="1" :inactive-value="0" />
            </el-form-item>

            <el-form-item label="启用分片上传">
              <el-switch v-model="configForm.transfer.upload.enablechunk" :active-value="1" :inactive-value="0" />
            </el-form-item>

            <el-form-item label="分享码位数">
              <el-input-number
                v-model="configForm.transfer.upload.sharecodelength"
                :min="1"
                :max="32"
                controls-position="right"
              />
              <span class="field-hint">默认 4 位，位数越短越容易撞码</span>
            </el-form-item>

            <el-form-item label="分享码字符集">
              <el-input
                v-model="configForm.transfer.upload.sharecodecharset"
                placeholder="例如：0123456789 或 AB0123456789"
              />
              <span class="field-hint">仅保留数字和字母；纯数字时前台会显示数字快捷输入</span>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="用户配置" name="user">
          <el-form :model="configForm.user" label-width="140px" style="max-width: 680px">
            <el-form-item label="允许用户注册">
              <el-switch v-model="configForm.user.allowuserregistration" :active-value="1" :inactive-value="0" />
            </el-form-item>

            <el-form-item label="用户上传限制">
              <div class="size-editor">
                <el-input-number
                  v-model="sizeEditors.useruploadsize.value"
                  :min="getSizeEditorMin('useruploadsize')"
                  :step="getSizeEditorStep('useruploadsize')"
                  :precision="getSizeEditorPrecision('useruploadsize')"
                  controls-position="right"
                />
                <el-select
                  :model-value="sizeEditors.useruploadsize.unit"
                  class="size-unit-select"
                  @change="handleUserUploadSizeUnitChange"
                >
                  <el-option v-for="unit in sizeUnitOptions" :key="unit" :label="unit" :value="unit" />
                </el-select>
                <span class="size-hint">当前约 {{ getSizePreview('useruploadsize') }}</span>
              </div>
            </el-form-item>

            <el-form-item label="用户存储配额">
              <div class="size-editor">
                <el-input-number
                  v-model="sizeEditors.userstoragequota.value"
                  :min="getSizeEditorMin('userstoragequota')"
                  :step="getSizeEditorStep('userstoragequota')"
                  :precision="getSizeEditorPrecision('userstoragequota')"
                  controls-position="right"
                />
                <el-select
                  :model-value="sizeEditors.userstoragequota.unit"
                  class="size-unit-select"
                  @change="handleUserStorageQuotaUnitChange"
                >
                  <el-option v-for="unit in sizeUnitOptions" :key="unit" :label="unit" :value="unit" />
                </el-select>
                <span class="size-hint">当前约 {{ getSizePreview('userstoragequota') }}</span>
              </div>
            </el-form-item>

            <el-form-item label="会话过期时间">
              <el-input-number
                v-model="configForm.user.sessionexpiryhours"
                :min="1"
                :max="720"
                controls-position="right"
              />
              <span class="field-hint">小时</span>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="账号配置" name="account">
          <el-form :model="accountForm" label-width="140px" style="max-width: 680px">
            <el-form-item label="管理员账号">
              <el-input v-model="accountForm.username" placeholder="留空则不修改，默认为原来的账号" />
            </el-form-item>
            <el-form-item label="当前密码" required>
              <el-input v-model="accountForm.old_password" type="password" show-password placeholder="请输入当前密码" />
            </el-form-item>
            <el-form-item label="新密码" required>
              <el-input v-model="accountForm.new_password" type="password" show-password placeholder="请输入新密码" />
            </el-form-item>
            <el-form-item label="确认密码" required>
              <el-input v-model="accountForm.confirm_password" type="password" show-password placeholder="请再次输入新密码" />
            </el-form-item>
            <el-form-item>
              <el-button type="danger" @click="saveAccount" :loading="savingAccount">
                修改管理员账号/密码
              </el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { adminApi } from '@/api/admin'
import { useConfigStore } from '@/stores/config'
import {
  FILE_SIZE_UNITS,
  formatFileSize,
  fromBytes,
  getFileSizePrecision,
  getFileSizeStep,
  normalizeFileSizeUnit,
  toBytes,
  type FileSizeUnit,
} from '@/utils/fileSize'

type SizeEditorKey = 'uploadsize' | 'useruploadsize' | 'userstoragequota'

interface SizeEditorState {
  value: number
  unit: FileSizeUnit
}

const loading = ref(false)
const saving = ref(false)
const savingAccount = ref(false)
const activeTab = ref('basic')
const configStore = useConfigStore()
const sizeUnitOptions = FILE_SIZE_UNITS

const configForm = reactive({
  base: {
    name: '',
    description: '',
    port: 12346,
    host: '0.0.0.0',
    production: false,
  },
  transfer: {
    upload: {
      openupload: 1,
      uploadsize: 10485760,
      requirelogin: 1,
      enablechunk: 1,
      chunksize: 2097152,
      sharecodelength: 4,
      sharecodecharset: '0123456789',
    },
  },
  user: {
    allowuserregistration: 0,
    useruploadsize: 52428800,
    userstoragequota: 1073741824,
    sessionexpiryhours: 168,
  },
})

const accountForm = reactive({
  username: '',
  old_password: '',
  new_password: '',
  confirm_password: '',
})

const sizeEditors = reactive<Record<SizeEditorKey, SizeEditorState>>({
  uploadsize: { value: 10, unit: 'MB' },
  useruploadsize: { value: 50, unit: 'MB' },
  userstoragequota: { value: 1, unit: 'GB' },
})

const syncSizeEditorsFromForm = () => {
  const uploadSize = fromBytes(configForm.transfer.upload.uploadsize)
  sizeEditors.uploadsize.value = uploadSize.value
  sizeEditors.uploadsize.unit = uploadSize.unit

  const userUploadSize = fromBytes(configForm.user.useruploadsize)
  sizeEditors.useruploadsize.value = userUploadSize.value
  sizeEditors.useruploadsize.unit = userUploadSize.unit

  const userStorageQuota = fromBytes(configForm.user.userstoragequota)
  sizeEditors.userstoragequota.value = userStorageQuota.value
  sizeEditors.userstoragequota.unit = userStorageQuota.unit
}

const syncFormFromSizeEditors = () => {
  configForm.transfer.upload.uploadsize = toBytes(sizeEditors.uploadsize.value, sizeEditors.uploadsize.unit)
  configForm.user.useruploadsize = toBytes(sizeEditors.useruploadsize.value, sizeEditors.useruploadsize.unit)
  configForm.user.userstoragequota = toBytes(sizeEditors.userstoragequota.value, sizeEditors.userstoragequota.unit)
}

const handleSizeUnitChange = (key: SizeEditorKey, nextUnit: string) => {
  const editor = sizeEditors[key]
  const bytes = toBytes(editor.value, editor.unit)
  const normalizedUnit = normalizeFileSizeUnit(nextUnit)
  const converted = fromBytes(bytes, normalizedUnit)

  editor.unit = normalizedUnit
  editor.value = converted.value || 0
}

const handleUploadSizeUnitChange = (unit: string | number | boolean) => {
  handleSizeUnitChange('uploadsize', String(unit))
}

const handleUserUploadSizeUnitChange = (unit: string | number | boolean) => {
  handleSizeUnitChange('useruploadsize', String(unit))
}

const handleUserStorageQuotaUnitChange = (unit: string | number | boolean) => {
  handleSizeUnitChange('userstoragequota', String(unit))
}

const getSizePreview = (key: SizeEditorKey) => {
  return formatFileSize(toBytes(sizeEditors[key].value, sizeEditors[key].unit))
}

const getSizeEditorStep = (key: SizeEditorKey) => getFileSizeStep(sizeEditors[key].unit)

const getSizeEditorPrecision = (key: SizeEditorKey) => getFileSizePrecision(sizeEditors[key].unit)

const getSizeEditorMin = (key: SizeEditorKey) => {
  const unit = sizeEditors[key].unit
  return unit === 'B' || unit === 'KB' ? 1 : 0.01
}

const fetchConfig = async () => {
  loading.value = true
  try {
    const res = await adminApi.getConfig()
    if (res.code === 200 && res.data) {
      if (res.data.base) {
        Object.assign(configForm.base, res.data.base)
      }
      if (res.data.transfer) {
        Object.assign(configForm.transfer, res.data.transfer)
      }
      if (res.data.user) {
        Object.assign(configForm.user, res.data.user)
      }
      syncSizeEditorsFromForm()
    }
  } catch (error) {
    console.error('获取配置失败:', error)
    ElMessage.error('获取配置失败')
  } finally {
    loading.value = false
  }
}

const saveConfig = async () => {
  syncFormFromSizeEditors()
  saving.value = true

  try {
    const payload = {
      base: { ...configForm.base },
      transfer: {
        upload: { ...configForm.transfer.upload },
      },
      user: { ...configForm.user },
    }

    const res = await adminApi.updateConfig(payload)
    if (res.code === 200) {
      ElMessage.success('配置保存成功')
      await configStore.refreshConfig()
      await fetchConfig()
    } else {
      ElMessage.error(res.message || '保存失败')
    }
  } catch (error) {
    console.error('保存配置失败:', error)
    ElMessage.error('保存配置失败')
  } finally {
    saving.value = false
  }
}

const saveAccount = async () => {
  if (!accountForm.old_password || !accountForm.new_password) {
    ElMessage.warning('原密码和新密码不能为空')
    return
  }
  if (accountForm.new_password !== accountForm.confirm_password) {
    ElMessage.warning('两次输入的新密码不一致')
    return
  }
  
  savingAccount.value = true
  try {
    const res = await adminApi.updateAccount({
      username: accountForm.username,
      old_password: accountForm.old_password,
      new_password: accountForm.new_password,
    })
    
    if (res.code === 200) {
      ElMessage.success('账号密码修改成功，请重新登录')
      // 可选：清空表单或强制重新登录
      accountForm.username = ''
      accountForm.old_password = ''
      accountForm.new_password = ''
      accountForm.confirm_password = ''
    } else {
      ElMessage.error(res.message || '修改失败')
    }
  } catch (error) {
    console.error('修改账号密码失败:', error)
    ElMessage.error('请求失败，请检查网络或后端日志')
  } finally {
    savingAccount.value = false
  }
}

onMounted(() => {
  fetchConfig()
})
</script>

<style scoped>
.system-config {
  padding: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.size-editor {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.size-unit-select {
  width: 100px;
}

.size-hint,
.field-hint {
  color: #909399;
  font-size: 13px;
}
</style>
