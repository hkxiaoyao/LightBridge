<template>
  <AppLayout>
    <div class="space-y-5">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">LightBridge Proxy</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">Outbound nodes, profiles, bindings, and local runtime state.</p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button class="btn btn-secondary" :disabled="loading" @click="loadAll">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            Refresh
          </button>
          <button class="btn btn-secondary" :disabled="migrating" @click="migrateLegacy">
            <Icon name="upload" size="sm" :class="{ 'animate-pulse': migrating }" />
            Migrate legacy
          </button>
        </div>
      </div>

      <div v-if="error" class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-200">
        {{ error }}
      </div>

      <div class="grid gap-3 md:grid-cols-4">
        <div v-for="item in runtimeCards" :key="item.label" class="rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-dark-400">{{ item.label }}</div>
          <div class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</div>
        </div>
      </div>

      <div v-if="migrationReport" class="rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-800 dark:border-blue-800/50 dark:bg-blue-900/20 dark:text-blue-200">
        Scanned {{ migrationReport.proxies_scanned }} proxies and {{ migrationReport.accounts_scanned }} accounts. Migrated {{ migrationReport.proxies_migrated }} proxies and {{ migrationReport.bindings_migrated }} bindings.
      </div>

      <div class="flex flex-wrap gap-2 border-b border-gray-200 dark:border-dark-700">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          class="px-3 py-2 text-sm font-medium"
          :class="activeTab === tab.key ? 'border-b-2 border-primary-500 text-primary-600 dark:text-primary-300' : 'text-gray-500 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'"
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>

      <section v-if="activeTab === 'nodes'" class="grid gap-4 xl:grid-cols-[360px_minmax(0,1fr)]">
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">Add node</h2>
          <form class="mt-4 space-y-3" @submit.prevent="createNode">
            <input v-model="nodeForm.name" class="input" placeholder="Display name" />
            <input v-model="nodeForm.url" class="input" placeholder="http://user:pass@host:8080" />
            <button class="btn btn-primary w-full" :disabled="submitting || !nodeForm.url.trim()">Add node</button>
          </form>

          <div class="mt-6 border-t border-gray-100 pt-4 dark:border-dark-700">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">Import subscription</h3>
            <form class="mt-3 space-y-3" @submit.prevent="importNodes">
              <select v-model="importForm.format" class="input">
                <option value="clash_yaml">Clash YAML</option>
                <option value="uri">URI</option>
              </select>
              <input v-model="importForm.url" class="input" placeholder="Subscription URL" />
              <textarea v-model="importForm.content" class="input min-h-[120px]" placeholder="Or paste YAML / URI content"></textarea>
              <button class="btn btn-secondary w-full" :disabled="submitting || (!importForm.url.trim() && !importForm.content.trim())">Import</button>
            </form>
          </div>
        </div>

        <div class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-900 dark:text-dark-400">
              <tr>
                <th class="px-4 py-3">Name</th>
                <th class="px-4 py-3">Type</th>
                <th class="px-4 py-3">Source</th>
                <th class="px-4 py-3">Server</th>
                <th class="px-4 py-3">Status</th>
                <th class="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="node in nodes" :key="node.id">
                <td class="px-4 py-3 font-medium text-gray-900 dark:text-white">{{ node.name }}</td>
                <td class="px-4 py-3">{{ node.node_type }}</td>
                <td class="px-4 py-3">{{ node.source_type }}</td>
                <td class="px-4 py-3">{{ node.config.server || '-' }}:{{ node.config.port || '-' }}</td>
                <td class="px-4 py-3">{{ node.status }}</td>
                <td class="px-4 py-3 text-right">
                  <button class="text-sm text-red-600 hover:underline dark:text-red-300" @click="deleteNode(node.id)">Delete</button>
                </td>
              </tr>
              <tr v-if="nodes.length === 0">
                <td colspan="6" class="px-4 py-10 text-center text-gray-500">No nodes</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-else-if="activeTab === 'profiles'" class="grid gap-4 xl:grid-cols-[360px_minmax(0,1fr)]">
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">Create profile</h2>
          <form class="mt-4 space-y-3" @submit.prevent="createProfile">
            <input v-model="profileForm.name" class="input" placeholder="Profile name" />
            <select v-model="profileForm.strategy" class="input">
              <option value="select">select</option>
              <option value="url_test">url_test</option>
              <option value="fallback">fallback</option>
              <option value="load_balance">load_balance</option>
            </select>
            <input v-model="profileForm.test_url" class="input" placeholder="https://www.gstatic.com/generate_204" />
            <input v-model.number="profileForm.interval_seconds" class="input" type="number" min="1" placeholder="Interval seconds" />
            <select v-model="selectedNodeIDs" class="input min-h-[140px]" multiple>
              <option v-for="node in nodes" :key="node.id" :value="node.id">{{ node.name }} / {{ node.node_type }}</option>
            </select>
            <button class="btn btn-primary w-full" :disabled="submitting || !profileForm.name.trim() || selectedNodeIDs.length === 0">Create profile</button>
          </form>
        </div>

        <div class="space-y-3">
          <div v-for="profile in profiles" :key="profile.id" class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h3 class="font-semibold text-gray-900 dark:text-white">{{ profile.name }}</h3>
                <div class="mt-1 text-sm text-gray-500 dark:text-dark-400">#{{ profile.id }} · {{ profile.strategy }} · {{ profile.status }}</div>
              </div>
              <div class="flex flex-wrap gap-2">
                <button class="btn btn-secondary px-3 py-1.5" @click="startProfile(profile.id)">Start</button>
                <button class="btn btn-secondary px-3 py-1.5" @click="stopProfile(profile.id)">Stop</button>
                <button class="btn btn-secondary px-3 py-1.5" @click="restartProfile(profile.id)">Restart</button>
                <button class="btn btn-secondary px-3 py-1.5" @click="testProfile(profile.id)">Test</button>
              </div>
            </div>
            <div v-if="profileResults[profile.id]" class="mt-3 rounded-md bg-gray-50 px-3 py-2 text-sm dark:bg-dark-900">
              {{ profileResults[profile.id] }}
            </div>
          </div>
          <div v-if="profiles.length === 0" class="rounded-lg border border-gray-200 bg-white p-10 text-center text-gray-500 dark:border-dark-700 dark:bg-dark-800">No profiles</div>
        </div>
      </section>

      <section v-else class="grid gap-4 xl:grid-cols-[360px_minmax(0,1fr)]">
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">Create binding</h2>
          <form class="mt-4 space-y-3" @submit.prevent="createBinding">
            <select v-model="bindingForm.entity_type" class="input">
              <option value="global">global</option>
              <option value="provider">provider</option>
              <option value="channel">channel</option>
              <option value="account">account</option>
            </select>
            <input v-model="bindingForm.entity_id" class="input" placeholder="default, provider id, channel id, or account id" />
            <select v-model.number="bindingForm.profile_id" class="input">
              <option :value="0">Select profile</option>
              <option v-for="profile in profiles" :key="profile.id" :value="profile.id">{{ profile.name }}</option>
            </select>
            <input v-model.number="bindingForm.priority" class="input" type="number" placeholder="Priority" />
            <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-dark-200">
              <input v-model="bindingForm.fallback_to_direct" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600" />
              Allow fallback to direct
            </label>
            <button class="btn btn-primary w-full" :disabled="submitting || !bindingForm.entity_id.trim() || bindingForm.profile_id <= 0">Create binding</button>
          </form>
        </div>

        <div class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-dark-900 dark:text-dark-400">
              <tr>
                <th class="px-4 py-3">Entity</th>
                <th class="px-4 py-3">Profile</th>
                <th class="px-4 py-3">Priority</th>
                <th class="px-4 py-3">Fallback</th>
                <th class="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="binding in bindings" :key="binding.id">
                <td class="px-4 py-3">{{ binding.entity_type }}:{{ binding.entity_id }}</td>
                <td class="px-4 py-3">{{ profileName(binding.profile_id) }}</td>
                <td class="px-4 py-3">{{ binding.priority }}</td>
                <td class="px-4 py-3">{{ binding.fallback_to_direct ? 'yes' : 'no' }}</td>
                <td class="px-4 py-3 text-right">
                  <button class="text-sm text-red-600 hover:underline dark:text-red-300" @click="deleteBinding(binding.id)">Delete</button>
                </td>
              </tr>
              <tr v-if="bindings.length === 0">
                <td colspan="5" class="px-4 py-10 text-center text-gray-500">No bindings</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import proxyModuleAPI, {
  type LegacyMigrationReport,
  type ProxyBindingView,
  type ProxyNodeView,
  type ProxyProfileView,
  type ProxyRuntimeStatusView
} from '@/api/admin/proxyModule'

const loading = ref(false)
const submitting = ref(false)
const migrating = ref(false)
const error = ref('')
const activeTab = ref<'nodes' | 'profiles' | 'bindings'>('nodes')
const nodes = ref<ProxyNodeView[]>([])
const profiles = ref<ProxyProfileView[]>([])
const bindings = ref<ProxyBindingView[]>([])
const runtimeStatus = ref<ProxyRuntimeStatusView>({ total: 0, starting: 0, running: 0, failed: 0, stopped: 0 })
const migrationReport = ref<LegacyMigrationReport | null>(null)
const selectedNodeIDs = ref<number[]>([])
const profileResults = reactive<Record<number, string>>({})

const nodeForm = reactive({ name: '', url: '' })
const importForm = reactive({ format: 'clash_yaml', url: '', content: '' })
const profileForm = reactive({ name: '', strategy: 'select', test_url: '', interval_seconds: 300 })
const bindingForm = reactive({ entity_type: 'global', entity_id: 'default', profile_id: 0, priority: 0, fallback_to_direct: false })
const tabs = [
  { key: 'nodes', label: 'Nodes' },
  { key: 'profiles', label: 'Profiles' },
  { key: 'bindings', label: 'Bindings' }
] as const

const runtimeCards = computed(() => [
  { label: 'Running', value: runtimeStatus.value.running },
  { label: 'Starting', value: runtimeStatus.value.starting },
  { label: 'Failed', value: runtimeStatus.value.failed },
  { label: 'Stopped', value: runtimeStatus.value.stopped }
])

function messageOf(err: unknown) {
  const e = err as { response?: { data?: { message?: string; error?: string } }; message?: string }
  return e.response?.data?.message || e.response?.data?.error || e.message || 'Operation failed'
}

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    const [nodeItems, profileItems, bindingItems, status] = await Promise.all([
      proxyModuleAPI.listNodes(),
      proxyModuleAPI.listProfiles(),
      proxyModuleAPI.listBindings(),
      proxyModuleAPI.getRuntimeStatus()
    ])
    nodes.value = nodeItems
    profiles.value = profileItems
    bindings.value = bindingItems
    runtimeStatus.value = status
  } catch (err) {
    error.value = messageOf(err)
  } finally {
    loading.value = false
  }
}

async function run(action: () => Promise<unknown>) {
  submitting.value = true
  error.value = ''
  try {
    await action()
    await loadAll()
  } catch (err) {
    error.value = messageOf(err)
  } finally {
    submitting.value = false
  }
}

function createNode() {
  return run(async () => {
    await proxyModuleAPI.createNode({ name: nodeForm.name, url: nodeForm.url })
    nodeForm.name = ''
    nodeForm.url = ''
  })
}

function importNodes() {
  return run(async () => {
    await proxyModuleAPI.importNodes({
      format: importForm.format,
      url: importForm.url.trim() || undefined,
      content: importForm.content.trim() || undefined
    })
    importForm.url = ''
    importForm.content = ''
  })
}

function deleteNode(id: number) {
  return run(() => proxyModuleAPI.deleteNode(id))
}

function createProfile() {
  return run(async () => {
    await proxyModuleAPI.createProfile({
      name: profileForm.name,
      strategy: profileForm.strategy,
      test_url: profileForm.test_url,
      interval_seconds: profileForm.interval_seconds,
      node_ids: selectedNodeIDs.value
    })
    profileForm.name = ''
    selectedNodeIDs.value = []
  })
}

function startProfile(id: number) {
  return run(async () => {
    const runtime = await proxyModuleAPI.startProfile(id)
    profileResults[id] = `started: ${runtime.proxy_url || runtime.status}`
  })
}

function stopProfile(id: number) {
  return run(async () => {
    const runtime = await proxyModuleAPI.stopProfile(id)
    profileResults[id] = `stopped: ${runtime.status}`
  })
}

function restartProfile(id: number) {
  return run(async () => {
    const runtime = await proxyModuleAPI.restartProfile(id)
    profileResults[id] = `restarted: ${runtime.proxy_url || runtime.status}`
  })
}

function testProfile(id: number) {
  return run(async () => {
    const result = await proxyModuleAPI.testProfile(id)
    profileResults[id] = result.healthy ? `healthy: ${result.version || result.proxy_url || result.status}` : `failed: ${result.error || result.status}`
  })
}

function createBinding() {
  return run(async () => {
    await proxyModuleAPI.createBinding({
      entity_type: bindingForm.entity_type,
      entity_id: bindingForm.entity_id,
      profile_id: bindingForm.profile_id,
      priority: bindingForm.priority,
      fallback_to_direct: bindingForm.fallback_to_direct
    })
  })
}

function deleteBinding(id: number) {
  return run(() => proxyModuleAPI.deleteBinding(id))
}

async function migrateLegacy() {
  migrating.value = true
  error.value = ''
  try {
    migrationReport.value = await proxyModuleAPI.migrateLegacy()
    await loadAll()
  } catch (err) {
    error.value = messageOf(err)
  } finally {
    migrating.value = false
  }
}

function profileName(id: number) {
  return profiles.value.find((profile) => profile.id === id)?.name || `#${id}`
}

onMounted(loadAll)
</script>
