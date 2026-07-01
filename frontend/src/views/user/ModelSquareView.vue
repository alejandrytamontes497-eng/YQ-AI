<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-4">
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <div class="rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-800">
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelSquare.stats.models') }}</p>
              <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ uniqueModelCount }}</p>
            </div>
            <div class="rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-800">
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelSquare.stats.platforms') }}</p>
              <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ platformCount }}</p>
            </div>
            <div class="rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-800">
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelSquare.stats.channels') }}</p>
              <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ channelCount }}</p>
            </div>
            <div class="rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-800">
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelSquare.stats.groups') }}</p>
              <p class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ groupCount }}</p>
            </div>
          </div>

          <div class="flex flex-col justify-between gap-3 xl:flex-row xl:items-center">
            <div class="flex flex-1 flex-col gap-3 md:flex-row">
              <div class="relative w-full md:max-w-md">
                <Icon
                  name="search"
                  size="md"
                  class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
                />
                <input
                  v-model="searchQuery"
                  type="text"
                  :placeholder="t('modelSquare.searchPlaceholder')"
                  class="input pl-10"
                />
              </div>
              <div class="w-full md:w-56">
                <Select
                  v-model="selectedPlatform"
                  :options="platformOptions"
                  :placeholder="t('modelSquare.filters.platform')"
                />
              </div>
              <div class="w-full md:w-56">
                <Select
                  v-model="selectedBillingMode"
                  :options="billingModeOptions"
                  :placeholder="t('modelSquare.filters.billingMode')"
                />
              </div>
            </div>

            <button
              @click="loadModels"
              :disabled="loading"
              class="btn btn-secondary"
              :title="t('common.refresh', 'Refresh')"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <div class="table-wrapper">
          <table class="min-w-[1280px] table-fixed border-collapse text-sm">
            <thead>
              <tr class="border-b border-gray-100 bg-gray-50/70 text-xs font-medium uppercase tracking-wide text-gray-500 dark:border-dark-700 dark:bg-dark-800/70 dark:text-gray-400">
                <th class="w-[260px] px-4 py-3 text-left">{{ t('modelSquare.columns.model') }}</th>
                <th class="w-[130px] px-4 py-3 text-left">{{ t('modelSquare.columns.billingMode') }}</th>
                <th class="w-[120px] px-4 py-3 text-right">{{ t('modelSquare.columns.inputPrice') }}</th>
                <th class="w-[120px] px-4 py-3 text-right">{{ t('modelSquare.columns.outputPrice') }}</th>
                <th class="w-[120px] px-4 py-3 text-right">{{ t('modelSquare.columns.cacheReadPrice') }}</th>
                <th class="w-[140px] px-4 py-3 text-right">{{ t('modelSquare.columns.extraPrice') }}</th>
                <th class="w-[210px] px-4 py-3 text-left">{{ t('modelSquare.columns.channel') }}</th>
                <th class="px-4 py-3 text-left">{{ t('modelSquare.columns.groups') }}</th>
              </tr>
            </thead>
            <tbody v-if="loading">
              <tr>
                <td colspan="8" class="py-10 text-center">
                  <Icon name="refresh" size="lg" class="inline-block animate-spin text-gray-400" />
                </td>
              </tr>
            </tbody>
            <tbody v-else-if="filteredRows.length === 0">
              <tr>
                <td colspan="8" class="py-12 text-center">
                  <Icon name="inbox" size="xl" class="mx-auto mb-3 h-12 w-12 text-gray-400" />
                  <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('modelSquare.empty') }}</p>
                </td>
              </tr>
            </tbody>
            <tbody v-else>
              <tr
                v-for="row in filteredRows"
                :key="row.id"
                class="border-b border-gray-100 transition-colors hover:bg-gray-50/60 dark:border-dark-800 dark:hover:bg-dark-800/60"
              >
                <td class="px-4 py-3 align-top">
                  <div class="min-w-0 space-y-2">
                    <div class="flex min-w-0 items-center gap-2">
                      <span class="truncate font-medium text-gray-900 dark:text-white" :title="row.model.name">
                        {{ row.model.name }}
                      </span>
                      <SupportedModelChip
                        :model="row.model"
                        pricing-key-prefix="availableChannels.pricing"
                        :no-pricing-label="t('modelSquare.noPricing')"
                        :show-platform="false"
                        :platform-hint="row.platform"
                      />
                    </div>
                    <span
                      :class="[
                        'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[11px] font-medium uppercase',
                        platformBadgeClass(row.platform),
                      ]"
                    >
                      <PlatformIcon :platform="row.platform as GroupPlatform" size="xs" />
                      {{ row.platform }}
                    </span>
                  </div>
                </td>

                <td class="px-4 py-3 align-top">
                  <span class="inline-flex rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-300">
                    {{ billingModeLabel(row.pricing?.billing_mode) }}
                  </span>
                  <div
                    v-if="row.pricing?.intervals?.length"
                    class="mt-2 text-xs text-gray-500 dark:text-gray-400"
                  >
                    {{ t('modelSquare.intervalCount', { count: row.pricing.intervals.length }) }}
                  </div>
                </td>

                <td class="px-4 py-3 text-right align-top font-mono text-xs">
                  {{ tokenPrice(row.pricing?.input_price) }}
                </td>
                <td class="px-4 py-3 text-right align-top font-mono text-xs">
                  {{ tokenPrice(row.pricing?.output_price) }}
                </td>
                <td class="px-4 py-3 text-right align-top font-mono text-xs">
                  {{ tokenPrice(row.pricing?.cache_read_price) }}
                </td>
                <td class="px-4 py-3 text-right align-top font-mono text-xs">
                  {{ extraPrice(row.pricing) }}
                </td>

                <td class="px-4 py-3 align-top">
                  <div class="min-w-0">
                    <div class="truncate font-medium text-gray-800 dark:text-gray-200" :title="row.channelName">
                      {{ row.channelName }}
                    </div>
                    <div
                      v-if="row.channelDescription"
                      class="mt-1 line-clamp-2 text-xs text-gray-500 dark:text-gray-400"
                      :title="row.channelDescription"
                    >
                      {{ row.channelDescription }}
                    </div>
                  </div>
                </td>

                <td class="px-4 py-3 align-top">
                  <div v-if="row.groups.length > 0" class="flex flex-wrap gap-1.5">
                    <GroupBadge
                      v-for="group in row.groups"
                      :key="`${row.id}-${group.id}`"
                      :name="group.name"
                      :platform="group.platform as GroupPlatform"
                      :subscription-type="(group.subscription_type || 'standard') as SubscriptionType"
                      :rate-multiplier="group.rate_multiplier"
                      :user-rate-multiplier="userGroupRates[group.id] ?? null"
                      always-show-rate
                    />
                  </div>
                  <span v-else class="text-xs text-gray-400">{{ t('modelSquare.noGroups') }}</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import SupportedModelChip from '@/components/channels/SupportedModelChip.vue'
import userChannelsAPI, {
  type UserAvailableChannel,
  type UserAvailableGroup,
  type UserSupportedModel,
  type UserSupportedModelPricing,
} from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_TOKEN,
  type BillingMode,
} from '@/constants/channel'
import type { GroupPlatform, SubscriptionType } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatScaled } from '@/utils/pricing'
import { platformBadgeClass, platformLabel } from '@/utils/platformColors'

interface ModelSquareRow {
  id: string
  model: UserSupportedModel
  platform: string
  pricing: UserSupportedModelPricing | null
  channelName: string
  channelDescription: string
  groups: UserAvailableGroup[]
}

const { t } = useI18n()
const appStore = useAppStore()

const channels = ref<UserAvailableChannel[]>([])
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)
const searchQuery = ref('')
const selectedPlatform = ref('')
const selectedBillingMode = ref('')

const allRows = computed<ModelSquareRow[]>(() => {
  const rows: ModelSquareRow[] = []
  channels.value.forEach((channel, channelIndex) => {
    channel.platforms.forEach((section, sectionIndex) => {
      section.supported_models.forEach((model, modelIndex) => {
        const platform = model.platform || section.platform
        const normalizedModel: UserSupportedModel = {
          ...model,
          platform,
        }
        rows.push({
          id: `${channelIndex}-${sectionIndex}-${modelIndex}-${platform}-${model.name}`,
          model: normalizedModel,
          platform,
          pricing: normalizedModel.pricing,
          channelName: channel.name,
          channelDescription: channel.description || '',
          groups: section.groups,
        })
      })
    })
  })
  return rows.sort((a, b) => {
    const byPlatform = a.platform.localeCompare(b.platform)
    if (byPlatform !== 0) return byPlatform
    const byModel = a.model.name.localeCompare(b.model.name)
    if (byModel !== 0) return byModel
    return a.channelName.localeCompare(b.channelName)
  })
})

const filteredRows = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  return allRows.value.filter((row) => {
    if (selectedPlatform.value && row.platform !== selectedPlatform.value) return false
    if (selectedBillingMode.value && row.pricing?.billing_mode !== selectedBillingMode.value) return false
    if (!q) return true
    return (
      row.model.name.toLowerCase().includes(q) ||
      row.platform.toLowerCase().includes(q) ||
      row.channelName.toLowerCase().includes(q) ||
      row.channelDescription.toLowerCase().includes(q) ||
      row.groups.some((group) => group.name.toLowerCase().includes(q))
    )
  })
})

const uniqueModelCount = computed(() => {
  return new Set(allRows.value.map((row) => `${row.platform}:${row.model.name.toLowerCase()}`)).size
})

const platformCount = computed(() => new Set(allRows.value.map((row) => row.platform)).size)
const channelCount = computed(() => new Set(allRows.value.map((row) => row.channelName)).size)
const groupCount = computed(() => {
  const ids = new Set<number>()
  allRows.value.forEach((row) => row.groups.forEach((group) => ids.add(group.id)))
  return ids.size
})

const platformOptions = computed<SelectOption[]>(() => {
  const platforms = Array.from(new Set(allRows.value.map((row) => row.platform))).sort()
  return [
    { value: '', label: t('modelSquare.filters.allPlatforms') },
    ...platforms.map((platform) => ({ value: platform, label: platformLabel(platform) })),
  ]
})

const billingModeOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('modelSquare.filters.allBillingModes') },
  { value: BILLING_MODE_TOKEN, label: t('availableChannels.pricing.billingModeToken') },
  { value: BILLING_MODE_PER_REQUEST, label: t('availableChannels.pricing.billingModePerRequest') },
  { value: BILLING_MODE_IMAGE, label: t('availableChannels.pricing.billingModeImage') },
])

function billingModeLabel(mode?: BillingMode | string | null): string {
  switch (mode) {
    case BILLING_MODE_TOKEN:
      return t('availableChannels.pricing.billingModeToken')
    case BILLING_MODE_PER_REQUEST:
      return t('availableChannels.pricing.billingModePerRequest')
    case BILLING_MODE_IMAGE:
      return t('availableChannels.pricing.billingModeImage')
    default:
      return t('modelSquare.noPricing')
  }
}

function tokenPrice(value?: number | null): string {
  return formatScaled(value ?? null, 1_000_000)
}

function extraPrice(pricing?: UserSupportedModelPricing | null): string {
  if (!pricing) return '-'
  if (pricing.billing_mode === BILLING_MODE_PER_REQUEST && pricing.per_request_price != null) {
    return formatScaled(pricing.per_request_price, 1)
  }
  if (pricing.billing_mode === BILLING_MODE_IMAGE && pricing.image_output_price != null) {
    return formatScaled(pricing.image_output_price, 1)
  }
  if (pricing.image_output_price != null) {
    return tokenPrice(pricing.image_output_price)
  }
  if (pricing.cache_write_price != null) {
    return tokenPrice(pricing.cache_write_price)
  }
  return '-'
}

async function loadModels() {
  loading.value = true
  try {
    const [list, rates] = await Promise.all([
      userChannelsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((err: unknown) => {
        console.error('Failed to load user group rates:', err)
        return {} as Record<number, number>
      }),
    ])
    channels.value = list
    userGroupRates.value = rates
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(loadModels)
</script>
