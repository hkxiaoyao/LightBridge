<template>
  <section class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ title }}</h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ subtitle }}</p>
      </div>
      <button type="button" class="btn btn-secondary px-3 py-1.5 text-xs" :disabled="loading" @click="load">
        <Icon name="refresh" size="xs" :class="{ 'animate-spin': loading }" />
        Refresh
      </button>
    </div>

    <div v-if="error" class="mt-3 rounded-md bg-red-50 px-3 py-2 text-xs text-red-700 dark:bg-red-900/20 dark:text-red-200">
      {{ error }}
    </div>

    <div v-if="currentBinding" class="mt-3 rounded-md bg-primary-50 px-3 py-2 text-sm text-primary-800 dark:bg-primary-900/20 dark:text-primary-200">
      <div class="font-medium">{{ profileName(currentBinding.profile_id) }}</div>
      <div class="mt-1 text-xs">
        priority {{ currentBinding.priority }} · fallback {{ currentBinding.fallback_to_direct ? 'enabled' : 'disabled' }}
      </div>
    </div>
    <div v-else class="mt-3 rounded-md bg-gray-50 px-3 py-2 text-sm text-gray-600 dark:bg-dark-900 dark:text-dark-300">
      {{ emptyText }}
    </div>

    <form class="mt-3 grid gap-3 sm:grid-cols-[minmax(0,1fr)_120px_auto] sm:items-end" @submit.prevent="save">
      <div>
        <label class="input-label">Profile</label>
        <select v-model.number="selectedProfileID" class="input">
          <option :value="0">Use inherited policy</option>
          <option v-for="profile in profiles" :key="profile.id" :value="profile.id">
            {{ profile.name }} · {{ profile.strategy }}
          </option>
        </select>
      </div>
      <label class="flex items-center gap-2 pb-2 text-sm text-gray-700 dark:text-dark-200">
        <input v-model="fallbackToDirect" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" />
        Fallback
      </label>
      <button type="submit" class="btn btn-primary" :disabled="saving || loading">
        Save
      </button>
    </form>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import proxyModuleAPI, { type ProxyBindingView, type ProxyProfileView } from '@/api/admin/proxyModule'

const props = defineProps<{
  entityType: 'channel' | 'account'
  entityId?: number | string | null
  title?: string
}>()

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const profiles = ref<ProxyProfileView[]>([])
const bindings = ref<ProxyBindingView[]>([])
const selectedProfileID = ref(0)
const fallbackToDirect = ref(false)

const entityID = computed(() => (props.entityId == null ? '' : String(props.entityId).trim()))
const title = computed(() => props.title || 'Proxy Policy')
const subtitle = computed(() => props.entityType === 'channel'
  ? 'Channel policy is used before provider/global defaults and can be overridden by account policy.'
  : 'Account policy overrides channel, provider, and global proxy policies.'
)
const emptyText = computed(() => props.entityType === 'channel'
  ? 'No channel policy. Requests inherit provider or global policy.'
  : 'No account override. Requests inherit channel, provider, or global policy.'
)
const currentBinding = computed(() => {
  if (!entityID.value) return null
  return bindings.value.find((binding) => binding.entity_type === props.entityType && binding.entity_id === entityID.value) || null
})

watch(currentBinding, (binding) => {
  selectedProfileID.value = binding?.profile_id || 0
  fallbackToDirect.value = binding?.fallback_to_direct || false
}, { immediate: true })

function messageOf(err: unknown) {
  const e = err as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return e.response?.data?.message || e.response?.data?.error || e.message || 'Proxy policy operation failed'
}

async function load() {
  if (!entityID.value) return
  loading.value = true
  error.value = ''
  try {
    const [profileItems, bindingItems] = await Promise.all([
      proxyModuleAPI.listProfiles(),
      proxyModuleAPI.listBindings()
    ])
    profiles.value = profileItems
    bindings.value = bindingItems
  } catch (err) {
    error.value = messageOf(err)
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!entityID.value) return
  saving.value = true
  error.value = ''
  try {
    const existing = currentBinding.value
    if (existing) {
      await proxyModuleAPI.deleteBinding(existing.id)
    }
    if (selectedProfileID.value > 0) {
      await proxyModuleAPI.createBinding({
        entity_type: props.entityType,
        entity_id: entityID.value,
        profile_id: selectedProfileID.value,
        priority: 0,
        fallback_to_direct: fallbackToDirect.value
      })
    }
    await load()
  } catch (err) {
    error.value = messageOf(err)
  } finally {
    saving.value = false
  }
}

function profileName(id: number) {
  return profiles.value.find((profile) => profile.id === id)?.name || `Profile #${id}`
}

onMounted(load)
watch(entityID, load)
</script>
