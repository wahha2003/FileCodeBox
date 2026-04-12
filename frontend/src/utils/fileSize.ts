export const FILE_SIZE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB'] as const

export type FileSizeUnit = (typeof FILE_SIZE_UNITS)[number]

const UNIT_FACTORS: Record<FileSizeUnit, number> = {
  B: 1,
  KB: 1024,
  MB: 1024 ** 2,
  GB: 1024 ** 3,
  TB: 1024 ** 4,
}

export const normalizeFileSizeUnit = (unit?: string): FileSizeUnit => {
  const normalized = (unit || '').trim().toUpperCase()
  return (FILE_SIZE_UNITS.find((item) => item === normalized) || 'MB') as FileSizeUnit
}

export const toBytes = (value: number, unit: FileSizeUnit): number => {
  if (!Number.isFinite(value) || value <= 0) {
    return 0
  }
  return Math.round(value * UNIT_FACTORS[unit])
}

export const fromBytes = (
  bytes: number,
  preferredUnit?: FileSizeUnit,
): { value: number; unit: FileSizeUnit } => {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return {
      value: 0,
      unit: preferredUnit || 'MB',
    }
  }

  const unit =
    preferredUnit ||
    [...FILE_SIZE_UNITS].reverse().find((item) => bytes >= UNIT_FACTORS[item]) ||
    'B'

  return {
    value: Math.round((bytes / UNIT_FACTORS[unit]) * 100) / 100,
    unit,
  }
}

export const formatFileSize = (bytes: number, decimals = 2): string => {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return '0 B'
  }

  const { value, unit } = fromBytes(bytes)
  const digits = unit === 'B' ? 0 : decimals
  return `${value.toFixed(digits).replace(/\.0+$/, '').replace(/(\.\d*[1-9])0+$/, '$1')} ${unit}`
}

export const getFileSizeStep = (unit: FileSizeUnit): number => {
  if (unit === 'B' || unit === 'KB') {
    return 1
  }
  return 0.5
}

export const getFileSizePrecision = (unit: FileSizeUnit): number => {
  if (unit === 'B' || unit === 'KB') {
    return 0
  }
  return 2
}
