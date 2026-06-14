<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <!-- 标题 -->
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('admin.privacyFilter.title') }}
        </h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.privacyFilter.description') }}
        </p>
      </div>

      <div v-if="loading" class="card p-8 text-center text-gray-500 dark:text-gray-400">
        {{ t('common.loading') }}
      </div>

      <template v-else>
        <!-- 基础开关 -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.privacyFilter.basic.title') }}
            </h2>
          </div>
          <div class="space-y-5 p-6">
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.privacyFilter.basic.enabled') }}
                </label>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.privacyFilter.basic.enabledHint') }}
                </p>
              </div>
              <Toggle v-model="form.enabled" />
            </div>
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.privacyFilter.basic.filterRequest') }}
                </label>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.privacyFilter.basic.filterRequestHint') }}
                </p>
              </div>
              <Toggle v-model="form.filter_request" />
            </div>
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.privacyFilter.basic.filterResponse') }}
                </label>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.privacyFilter.basic.filterResponseHint') }}
                </p>
              </div>
              <Toggle v-model="form.filter_response" />
            </div>
          </div>
        </div>

        <!-- 内置规则 -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.privacyFilter.builtin.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.privacyFilter.builtin.description') }}
            </p>
          </div>
          <div class="grid grid-cols-1 gap-4 p-6 sm:grid-cols-2">
            <label
              v-for="id in builtinIds"
              :key="id"
              class="flex items-center gap-3 rounded-lg border border-gray-200 px-4 py-3 dark:border-dark-700"
            >
              <input
                type="checkbox"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="form.builtin_rules[id] !== false"
                @change="toggleBuiltin(id, ($event.target as HTMLInputElement).checked)"
              />
              <span class="text-sm text-gray-700 dark:text-gray-300">{{ builtinLabel(id) }}</span>
            </label>
          </div>
        </div>

        <!-- 自定义规则 -->
        <div class="card">
          <div class="flex items-center justify-between border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('admin.privacyFilter.custom.title') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.privacyFilter.custom.description') }}
              </p>
            </div>
            <button class="btn-secondary" type="button" @click="addCustomRule">
              {{ t('admin.privacyFilter.custom.add') }}
            </button>
          </div>
          <div class="space-y-3 p-6">
            <p v-if="form.custom_rules.length === 0" class="text-sm text-gray-400">
              {{ t('admin.privacyFilter.custom.empty') }}
            </p>
            <div
              v-for="(rule, index) in form.custom_rules"
              :key="index"
              class="grid grid-cols-1 items-center gap-3 rounded-lg border border-gray-200 p-4 dark:border-dark-700 md:grid-cols-12"
            >
              <input
                v-model="rule.name"
                class="input md:col-span-3"
                :placeholder="t('admin.privacyFilter.custom.namePlaceholder')"
              />
              <input
                v-model="rule.pattern"
                class="input font-mono md:col-span-4"
                :placeholder="t('admin.privacyFilter.custom.patternPlaceholder')"
              />
              <input
                v-model="rule.replacement"
                class="input md:col-span-3"
                :placeholder="t('admin.privacyFilter.custom.replacementPlaceholder')"
              />
              <div class="flex items-center justify-end gap-3 md:col-span-2">
                <Toggle v-model="rule.enabled" />
                <button
                  class="text-sm text-red-600 hover:underline dark:text-red-400"
                  type="button"
                  @click="removeCustomRule(index)"
                >
                  {{ t('common.delete') }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- 作用域 -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.privacyFilter.scope.title') }}
            </h2>
          </div>
          <div class="space-y-5 p-6">
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.privacyFilter.scope.allGroups') }}
                </label>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.privacyFilter.scope.allGroupsHint') }}
                </p>
              </div>
              <Toggle v-model="form.all_groups" />
            </div>
            <div v-if="!form.all_groups">
              <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.privacyFilter.scope.groups') }}
              </label>
              <div class="grid grid-cols-1 gap-2 sm:grid-cols-2 md:grid-cols-3">
                <label
                  v-for="group in groups"
                  :key="group.id"
                  class="flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 dark:border-dark-700"
                >
                  <input
                    type="checkbox"
                    class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                    :checked="form.group_ids.includes(group.id)"
                    @change="toggleGroup(group.id, ($event.target as HTMLInputElement).checked)"
                  />
                  <span class="truncate text-sm text-gray-700 dark:text-gray-300">{{ group.name }}</span>
                </label>
              </div>
            </div>
            <div>
              <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.privacyFilter.scope.modelFilter') }}
              </label>
              <Select v-model="form.model_filter.type" :options="modelFilterOptions" class="max-w-xs" />
              <textarea
                v-if="form.model_filter.type !== 'all'"
                v-model="modelsText"
                class="input mt-3 h-24 w-full font-mono"
                :placeholder="t('admin.privacyFilter.scope.modelsPlaceholder')"
              />
            </div>
          </div>
        </div>

        <!-- 保存 -->
        <div class="flex items-center justify-end gap-3">
          <span v-if="statusMessage" :class="statusError ? 'text-red-600' : 'text-green-600'" class="text-sm">
            {{ statusMessage }}
          </span>
          <button class="btn-primary" type="button" :disabled="saving" @click="save">
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Toggle from '@/components/common/Toggle.vue'
import Select from '@/components/common/Select.vue'
import { adminAPI } from '@/api/admin'
import type {
  PrivacyFilterConfig,
  PrivacyFilterModelFilterType,
  PrivacyFilterRule,
} from '@/api/admin/privacyFilter'
import type { AdminGroup, SelectOption } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()

const loading = ref(true)
const saving = ref(false)
const statusMessage = ref('')
const statusError = ref(false)
const groups = ref<AdminGroup[]>([])
const builtinIds = ref<string[]>([])

const form = reactive<{
  enabled: boolean
  filter_request: boolean
  filter_response: boolean
  builtin_rules: Record<string, boolean>
  custom_rules: PrivacyFilterRule[]
  all_groups: boolean
  group_ids: number[]
  model_filter: { type: PrivacyFilterModelFilterType; models: string[] }
}>({
  enabled: false,
  filter_request: true,
  filter_response: true,
  builtin_rules: {},
  custom_rules: [],
  all_groups: true,
  group_ids: [],
  model_filter: { type: 'all', models: [] },
})

const modelsText = computed({
  get: () => form.model_filter.models.join('\n'),
  set: (val: string) => {
    form.model_filter.models = val
      .split(/[\n,]+/)
      .map((m) => m.trim())
      .filter((m) => m !== '')
  },
})

const modelFilterOptions = computed<SelectOption[]>(() => [
  { value: 'all', label: t('admin.privacyFilter.scope.modelFilterAll') },
  { value: 'include', label: t('admin.privacyFilter.scope.modelFilterInclude') },
  { value: 'exclude', label: t('admin.privacyFilter.scope.modelFilterExclude') },
])

function builtinLabel(id: string): string {
  const key = `admin.privacyFilter.builtins.${id}`
  const label = t(key)
  return label === key ? id : label
}

function toggleBuiltin(id: string, checked: boolean) {
  form.builtin_rules[id] = checked
}

function toggleGroup(id: number, checked: boolean) {
  if (checked) {
    if (!form.group_ids.includes(id)) form.group_ids.push(id)
  } else {
    form.group_ids = form.group_ids.filter((g) => g !== id)
  }
}

function addCustomRule() {
  form.custom_rules.push({ name: '', pattern: '', replacement: '[REDACTED]', enabled: true })
}

function removeCustomRule(index: number) {
  form.custom_rules.splice(index, 1)
}

function applyConfig(cfg: PrivacyFilterConfig) {
  form.enabled = cfg.enabled
  form.filter_request = cfg.filter_request
  form.filter_response = cfg.filter_response
  form.builtin_rules = { ...cfg.builtin_rules }
  form.custom_rules = (cfg.custom_rules || []).map((r) => ({ ...r }))
  form.all_groups = cfg.all_groups
  form.group_ids = [...(cfg.group_ids || [])]
  form.model_filter = {
    type: cfg.model_filter?.type || 'all',
    models: [...(cfg.model_filter?.models || [])],
  }
  builtinIds.value = cfg.builtin_rule_ids || Object.keys(cfg.builtin_rules || {})
}

async function load() {
  loading.value = true
  try {
    const [cfg, groupList] = await Promise.all([
      adminAPI.privacyFilter.getConfig(),
      adminAPI.groups.getAll(),
    ])
    applyConfig(cfg)
    groups.value = groupList
  } catch (e) {
    statusError.value = true
    statusMessage.value = extractApiErrorMessage(e)
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  statusMessage.value = ''
  try {
    const cfg = await adminAPI.privacyFilter.updateConfig({
      enabled: form.enabled,
      filter_request: form.filter_request,
      filter_response: form.filter_response,
      builtin_rules: { ...form.builtin_rules },
      custom_rules: form.custom_rules.map((r) => ({ ...r })),
      all_groups: form.all_groups,
      group_ids: [...form.group_ids],
      model_filter: { type: form.model_filter.type, models: [...form.model_filter.models] },
    })
    applyConfig(cfg)
    statusError.value = false
    statusMessage.value = t('common.saved')
  } catch (e) {
    statusError.value = true
    statusMessage.value = extractApiErrorMessage(e)
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
