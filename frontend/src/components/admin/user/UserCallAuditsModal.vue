<template>
  <BaseDialog :show="show" title="调用审计" width="wide" @close="handleClose">
    <div v-if="user" class="space-y-4">
      <div class="flex items-center justify-between rounded-lg bg-gray-50 p-4 dark:bg-dark-700">
        <div>
          <p class="font-medium text-gray-900 dark:text-white">{{ user.email }}</p>
          <p class="text-sm text-gray-500 dark:text-dark-400">最近 10 条调用记录</p>
        </div>
        <button class="btn-secondary px-3 py-2 text-sm" :disabled="loading" @click="load">
          刷新
        </button>
      </div>

      <div v-if="loading" class="flex justify-center py-8">
        <svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
      </div>

      <div v-else-if="audits.length === 0" class="py-8 text-center text-sm text-gray-500">
        暂无调用审计记录
      </div>

      <div v-else class="max-h-[520px] overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
        <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-600">
          <thead class="bg-gray-50 text-left text-xs font-medium uppercase text-gray-500 dark:bg-dark-700 dark:text-dark-300">
            <tr>
              <th class="px-3 py-2">时间</th>
              <th class="px-3 py-2">状态</th>
              <th class="px-3 py-2">模型</th>
              <th class="px-3 py-2">路径</th>
              <th class="px-3 py-2">账号/分组</th>
              <th class="px-3 py-2">入参</th>
              <th class="px-3 py-2">出参</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-800">
            <tr v-for="item in audits" :key="item.id">
              <td class="whitespace-nowrap px-3 py-2 text-gray-500">{{ formatDateTime(item.created_at) }}</td>
              <td class="px-3 py-2">
                <span :class="statusClass(item.status_code)" class="inline-flex rounded px-2 py-0.5 text-xs font-medium">
                  {{ item.status_code || '-' }}
                </span>
              </td>
              <td class="max-w-[150px] truncate px-3 py-2 font-medium text-gray-900 dark:text-white" :title="item.model">
                {{ item.model || '-' }}
              </td>
              <td class="max-w-[180px] truncate px-3 py-2 text-gray-600 dark:text-dark-300" :title="`${item.method} ${item.path}`">
                {{ item.method }} {{ item.path }}
              </td>
              <td class="whitespace-nowrap px-3 py-2 text-gray-500">
                A:{{ item.account_id || '-' }} / G:{{ item.group_id || '-' }}
              </td>
              <td class="max-w-[210px] truncate px-3 py-2 font-mono text-xs text-gray-600 dark:text-dark-300" :title="item.request_excerpt">
                {{ item.request_excerpt || '-' }}
              </td>
              <td class="max-w-[210px] truncate px-3 py-2 font-mono text-xs text-gray-600 dark:text-dark-300" :title="item.response_excerpt">
                {{ item.response_excerpt || '-' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { adminAPI } from '@/api/admin'
import type { AdminUser } from '@/types'
import type { CallAuditLog } from '@/api/admin/users'
import { formatDateTime } from '@/utils/format'
import BaseDialog from '@/components/common/BaseDialog.vue'

const props = defineProps<{ show: boolean; user: AdminUser | null }>()
const emit = defineEmits(['close'])

const loading = ref(false)
const audits = ref<CallAuditLog[]>([])

watch(() => props.show, (visible) => {
  if (visible && props.user) {
    load()
  } else {
    audits.value = []
  }
})

const load = async () => {
  if (!props.user) return
  loading.value = true
  try {
    audits.value = await adminAPI.users.getUserCallAudits(props.user.id)
  } finally {
    loading.value = false
  }
}

const handleClose = () => {
  emit('close')
}

const statusClass = (status: number) => {
  if (status >= 200 && status < 300) return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
  if (status >= 400 && status < 500) return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300'
  if (status >= 500) return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-dark-300'
}
</script>
