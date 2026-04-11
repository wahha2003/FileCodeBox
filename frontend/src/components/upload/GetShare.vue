<template>
  <div class="get-share-container">
    <div class="input-section">
      <div class="input-icon">
        <el-icon size="40" color="#667eea"><Search /></el-icon>
      </div>
      <el-input
        v-if="!numericOnly"
        v-model="shareCode"
        size="large"
        :type="numericOnly ? 'tel' : 'text'"
        :inputmode="numericOnly ? 'numeric' : 'text'"
        :placeholder="inputPlaceholder"
        class="code-input"
        clearable
        :maxlength="maxCodeLength"
        autocomplete="off"
        @input="handleInput"
        @keyup.enter="handleGetShare"
      >
        <template #prefix>
          <el-icon><Key /></el-icon>
        </template>
      </el-input>

      <div v-if="numericOnly" class="number-pad">
        <div class="number-pad-shell">
          <div class="code-progress">
            <div
              v-for="slotIndex in shareCodeLength"
              :key="slotIndex"
              class="code-slot"
              :class="{
                filled: slotIndex <= shareCode.length,
                active: slotIndex === shareCode.length + 1 && shareCode.length < shareCodeLength,
              }"
            >
              {{ shareCode[slotIndex - 1] || '•' }}
            </div>
          </div>
          <p class="number-pad-help">输入满 {{ shareCodeLength }} 位后自动验证，无需再点按钮</p>
          <div class="number-pad-grid">
            <el-button
              v-for="key in keypadKeys"
              :key="key"
              class="number-key"
              :class="{ 'action-key': typeof key !== 'number' }"
              @click="handleKeypadPress(key)"
            >
              {{ keypadLabelMap[key] }}
            </el-button>
          </div>
        </div>
      </div>

      <el-button
        type="primary"
        size="large"
        class="get-btn"
        @click="handleGetShare"
      >
        <template #icon>
          <el-icon><Download /></el-icon>
        </template>
        获取分享
      </el-button>
    </div>

    <div class="tips-section">
      <el-alert
        type="info"
        :closable="false"
      >
        <template #title>
          <div class="tips-content">
            <p><strong>💡 使用提示：</strong></p>
            <p>• 输入分享码可获取他人分享的文件或文本</p>
            <p>• 当前分享码通常为 {{ shareCodeLength }} 位{{ numericOnly ? '数字' : '字符' }}（如：{{ shareCodeExample }})</p>
            <p v-if="numericOnly">• 纯数字分享码已启用数字快捷输入，移动端会优先弹出数字键盘</p>
            <p>• 输入达到设定位数后会自动跳转验证</p>
            <p>• 部分分享可能需要密码访问</p>
          </div>
        </template>
      </el-alert>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Search, Key, Download } from '@element-plus/icons-vue'
import { useConfigStore } from '@/stores/config'

interface Props {
  initialCode?: string
}

const props = defineProps<Props>()
const router = useRouter()
const configStore = useConfigStore()

const shareCode = ref('')
const isNavigating = ref(false)
const maxCodeLength = 32
const defaultCharset = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'
const keypadKeys = [1, 2, 3, 4, 5, 6, 7, 8, 9, 'clear', 0, 'backspace'] as const
const keypadLabelMap: Record<(typeof keypadKeys)[number], string> = {
  0: '0',
  1: '1',
  2: '2',
  3: '3',
  4: '4',
  5: '5',
  6: '6',
  7: '7',
  8: '8',
  9: '9',
  clear: '清空',
  backspace: '删除'
}

const getCaseMode = (charset: string) => {
  const hasUppercase = /[A-Z]/.test(charset)
  const hasLowercase = /[a-z]/.test(charset)

  if (hasUppercase && !hasLowercase) return 'upper'
  if (hasLowercase && !hasUppercase) return 'lower'
  return 'mixed'
}

const normalizeCharset = (charset?: string) => {
  const fallback = defaultCharset
  if (!charset) return fallback

  const seen = new Set<string>()
  const filtered = Array.from(charset).filter((char) => {
    if (!/[0-9A-Za-z]/.test(char) || seen.has(char)) {
      return false
    }
    seen.add(char)
    return true
  })

  return filtered.length > 0 ? filtered.join('') : fallback
}

const shareCodeLength = computed(() => {
  const configured = Number(configStore.config?.shareCodeLength)
  if (!Number.isFinite(configured) || configured <= 0) {
    return 4
  }
  return Math.min(Math.floor(configured), maxCodeLength)
})

const shareCodeCharset = computed(() => normalizeCharset(configStore.config?.shareCodeCharset))
const isConfigReady = computed(() => configStore.loaded && !configStore.loading)
const numericOnly = computed(() => isConfigReady.value && /^[0-9]+$/.test(shareCodeCharset.value))
const inputPlaceholder = computed(() =>
  numericOnly.value ? `请输入数字分享码` : '请输入分享码'
)
const shareCodeExample = computed(() => {
  if (numericOnly.value) {
    const digits = '1234567890'
    return digits.slice(0, shareCodeLength.value)
  }

  const charset = shareCodeCharset.value || defaultCharset
  let example = ''
  while (example.length < shareCodeLength.value) {
    example += charset
  }
  return example.slice(0, shareCodeLength.value)
})
const normalizeInput = (value: string) => {
  let normalized = value.replace(/\s+/g, '')
  const caseMode = getCaseMode(shareCodeCharset.value)

  if (numericOnly.value) {
    normalized = normalized.replace(/\D+/g, '')
  } else {
    if (caseMode === 'upper') {
      normalized = normalized.toUpperCase()
    } else if (caseMode === 'lower') {
      normalized = normalized.toLowerCase()
    }

    normalized = Array.from(normalized)
      .filter((char) => shareCodeCharset.value.includes(char))
      .join('')
  }

  return normalized.slice(0, maxCodeLength)
}

const submitShareCode = async (code: string) => {
  if (isNavigating.value) {
    return
  }

  isNavigating.value = true
  try {
    await router.push(`/share/${code}`)
  } finally {
    isNavigating.value = false
  }
}

// 监听 initialCode 变化
watch(() => props.initialCode, (newCode) => {
  if (newCode) {
    shareCode.value = normalizeInput(newCode)
    handleGetShare()
  }
}, { immediate: true })

onMounted(() => {
  if (!configStore.loaded) {
    configStore.fetchConfig()
  }

  window.addEventListener('keydown', handleWindowKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleWindowKeydown)
})

watch(
  [shareCode, shareCodeLength, isConfigReady],
  ([code, codeLength, ready]) => {
    if (!ready || isNavigating.value) {
      return
    }
    if (codeLength > 0 && code.length === codeLength) {
      submitShareCode(code)
    }
  },
)

const handleInput = (value: string) => {
  shareCode.value = normalizeInput(value)
}

const handleKeypadPress = (key: (typeof keypadKeys)[number]) => {
  if (key === 'clear') {
    shareCode.value = ''
    return
  }

  if (key === 'backspace') {
    shareCode.value = shareCode.value.slice(0, -1)
    return
  }

  shareCode.value = normalizeInput(`${shareCode.value}${key}`)
}

const handleWindowKeydown = (event: KeyboardEvent) => {
  if (!numericOnly.value || isNavigating.value) {
    return
  }

  const activeElement = document.activeElement as HTMLElement | null
  const tagName = activeElement?.tagName?.toLowerCase()
  if (tagName === 'input' || tagName === 'textarea' || activeElement?.isContentEditable) {
    return
  }

  if (/^[0-9]$/.test(event.key)) {
    event.preventDefault()
    handleKeypadPress(Number(event.key) as (typeof keypadKeys)[number])
    return
  }

  if (event.key === 'Backspace') {
    event.preventDefault()
    handleKeypadPress('backspace')
    return
  }

  if (event.key === 'Delete') {
    event.preventDefault()
    handleKeypadPress('clear')
    return
  }

  if (event.key === 'Enter') {
    event.preventDefault()
    handleGetShare()
  }
}

const handleGetShare = () => {
  shareCode.value = normalizeInput(shareCode.value)
  if (!shareCode.value) {
    ElMessage.warning('请输入分享码')
    return
  }
  submitShareCode(shareCode.value)
}
</script>

<style scoped>
.get-share-container {
  padding: 20px 0;
}

.input-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
  margin-bottom: 40px;
}

.input-icon {
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.1);
  }
}

.code-input {
  width: 100%;
  max-width: 500px;
}

.number-pad {
  width: 100%;
  max-width: 500px;
}

.number-pad-shell {
  position: relative;
  overflow: hidden;
  padding: 18px;
  border-radius: 24px;
  background:
    radial-gradient(circle at top left, rgba(102, 126, 234, 0.18), transparent 42%),
    radial-gradient(circle at bottom right, rgba(245, 87, 108, 0.14), transparent 38%),
    linear-gradient(145deg, #f8faff 0%, #eef2ff 55%, #fff7fb 100%);
  border: 1px solid rgba(123, 140, 255, 0.18);
  box-shadow:
    0 18px 40px rgba(102, 126, 234, 0.12),
    inset 0 1px 0 rgba(255, 255, 255, 0.72);
}

.code-progress {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(56px, 1fr));
  gap: 10px;
  margin-bottom: 14px;
}

.code-slot {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 56px;
  border-radius: 18px;
  font-size: 22px;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: rgba(88, 98, 150, 0.45);
  background: rgba(255, 255, 255, 0.68);
  border: 1px solid rgba(255, 255, 255, 0.9);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.8),
    0 10px 24px rgba(102, 126, 234, 0.08);
  transition: all 0.25s ease;
}

.code-slot.filled {
  color: #35406d;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(236, 241, 255, 0.92));
}

.code-slot.active {
  border-color: rgba(102, 126, 234, 0.38);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.9),
    0 0 0 4px rgba(102, 126, 234, 0.1),
    0 12px 28px rgba(102, 126, 234, 0.14);
}

.number-pad-help {
  margin: 0 0 16px;
  text-align: center;
  font-size: 13px;
  font-weight: 500;
  color: #5c6488;
}

.number-pad-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.number-key {
  height: 58px;
  margin: 0;
  border-radius: 18px;
  font-size: 20px;
  font-weight: 700;
  border: 1px solid rgba(255, 255, 255, 0.95);
  color: #2f365f;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(239, 243, 255, 0.96));
  box-shadow:
    0 12px 28px rgba(102, 126, 234, 0.12),
    inset 0 1px 0 rgba(255, 255, 255, 0.88);
  transition: transform 0.18s ease, box-shadow 0.18s ease, background 0.18s ease;
}

.number-key:hover {
  color: #2f365f;
  background: linear-gradient(180deg, #ffffff 0%, #e8efff 100%);
  transform: translateY(-2px);
  box-shadow:
    0 14px 32px rgba(102, 126, 234, 0.18),
    inset 0 1px 0 rgba(255, 255, 255, 0.9);
}

.number-key:active {
  transform: translateY(0);
}

.number-key.action-key {
  font-size: 15px;
  font-weight: 600;
  color: #5a63a8;
  background: linear-gradient(135deg, rgba(247, 248, 255, 0.98), rgba(255, 239, 245, 0.92));
}

.code-input :deep(.el-input__wrapper) {
  padding: 12px 16px;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  transition: all 0.3s;
}

.code-input :deep(.el-input__wrapper:hover) {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
}

.code-input :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 4px 16px rgba(102, 126, 234, 0.2);
}

.get-btn {
  width: 100%;
  max-width: 500px;
  height: 48px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 12px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  transition: all 0.3s;
}

.get-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 8px 20px rgba(102, 126, 234, 0.4);
}

.tips-section {
  padding: 20px;
  background: #f5f7fa;
  border-radius: 12px;
}

.tips-content p {
  margin: 8px 0;
  line-height: 1.6;
  font-size: 14px;
}

.tips-content p:first-child {
  margin-top: 0;
}

.tips-content p:last-child {
  margin-bottom: 0;
}

@media (max-width: 640px) {
  .number-pad-shell {
    padding: 16px;
    border-radius: 20px;
  }

  .code-progress {
    gap: 8px;
  }

  .code-slot {
    min-height: 50px;
    font-size: 20px;
  }

  .number-key {
    height: 52px;
    font-size: 18px;
  }
}
</style>
