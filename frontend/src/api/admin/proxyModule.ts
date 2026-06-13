import { apiClient } from '../client'

export interface ProxyNodeView {
  id: number
  name: string
  node_type: string
  source_type: string
  config: Record<string, unknown>
  status: string
}

export interface ProxyProfileView {
  id: number
  name: string
  strategy: string
  test_url: string
  interval_seconds: number
  status: string
  config: Record<string, unknown>
  runtime: Record<string, unknown>
}

export interface ProxyBindingView {
  id: number
  entity_type: string
  entity_id: string
  profile_id: number
  priority: number
  enabled: boolean
  fallback_to_direct: boolean
}

export interface ProxyRuntimeView {
  profile_id: number
  runtime_type: string
  pid?: number
  mixed_port?: number
  controller_port?: number
  config_path?: string
  work_dir?: string
  status: string
  last_error?: string
  proxy_url?: string
}

export interface ProxyRuntimeStatusView {
  total: number
  starting: number
  running: number
  failed: number
  stopped: number
}

export interface ProxyProfileTestView {
  profile_id: number
  healthy: boolean
  status: string
  version?: string
  error?: string
  proxy_url?: string
}

export interface LegacyMigrationReport {
  proxies_scanned: number
  proxies_migrated: number
  accounts_scanned: number
  bindings_migrated: number
  warnings?: string[]
}

export interface CreateProxyNodeInput {
  name: string
  url: string
}

export interface ImportProxyNodesInput {
  format: string
  content?: string
  url?: string
}

export interface CreateProxyProfileInput {
  name: string
  strategy: string
  test_url?: string
  interval_seconds?: number
  node_ids: number[]
  weights?: number[]
}

export interface CreateProxyBindingInput {
  entity_type: string
  entity_id: string
  profile_id: number
  priority?: number
  fallback_to_direct?: boolean
}

export async function listNodes(): Promise<ProxyNodeView[]> {
  const { data } = await apiClient.get<{ nodes: ProxyNodeView[] }>('/admin/proxy/nodes')
  return data.nodes || []
}

export async function createNode(input: CreateProxyNodeInput): Promise<ProxyNodeView> {
  const { data } = await apiClient.post<ProxyNodeView>('/admin/proxy/nodes', input)
  return data
}

export async function importNodes(input: ImportProxyNodesInput): Promise<ProxyNodeView[]> {
  const { data } = await apiClient.post<{ nodes: ProxyNodeView[] }>('/admin/proxy/nodes/import', input)
  return data.nodes || []
}

export async function deleteNode(id: number): Promise<void> {
  await apiClient.delete(`/admin/proxy/nodes/${id}`)
}

export async function listProfiles(): Promise<ProxyProfileView[]> {
  const { data } = await apiClient.get<{ profiles: ProxyProfileView[] }>('/admin/proxy/profiles')
  return data.profiles || []
}

export async function createProfile(input: CreateProxyProfileInput): Promise<ProxyProfileView> {
  const { data } = await apiClient.post<ProxyProfileView>('/admin/proxy/profiles', input)
  return data
}

export async function updateProfile(id: number, input: CreateProxyProfileInput): Promise<ProxyProfileView> {
  const { data } = await apiClient.put<ProxyProfileView>(`/admin/proxy/profiles/${id}`, input)
  return data
}

export async function startProfile(id: number): Promise<ProxyRuntimeView> {
  const { data } = await apiClient.post<ProxyRuntimeView>(`/admin/proxy/profiles/${id}/start`)
  return data
}

export async function stopProfile(id: number): Promise<ProxyRuntimeView> {
  const { data } = await apiClient.post<ProxyRuntimeView>(`/admin/proxy/profiles/${id}/stop`)
  return data
}

export async function restartProfile(id: number): Promise<ProxyRuntimeView> {
  const { data } = await apiClient.post<ProxyRuntimeView>(`/admin/proxy/profiles/${id}/restart`)
  return data
}

export async function testProfile(id: number): Promise<ProxyProfileTestView> {
  const { data } = await apiClient.post<ProxyProfileTestView>(`/admin/proxy/profiles/${id}/test`)
  return data
}

export async function getProfileRuntime(id: number): Promise<ProxyRuntimeView> {
  const { data } = await apiClient.get<ProxyRuntimeView>(`/admin/proxy/profiles/${id}/runtime`)
  return data
}

export async function getRuntimeStatus(): Promise<ProxyRuntimeStatusView> {
  const { data } = await apiClient.get<ProxyRuntimeStatusView>('/admin/proxy/runtime/status')
  return data
}

export async function listBindings(): Promise<ProxyBindingView[]> {
  const { data } = await apiClient.get<{ bindings: ProxyBindingView[] }>('/admin/proxy/bindings')
  return data.bindings || []
}

export async function createBinding(input: CreateProxyBindingInput): Promise<ProxyBindingView> {
  const { data } = await apiClient.post<ProxyBindingView>('/admin/proxy/bindings', input)
  return data
}

export async function deleteBinding(id: number): Promise<void> {
  await apiClient.delete(`/admin/proxy/bindings/${id}`)
}

export async function migrateLegacy(): Promise<LegacyMigrationReport> {
  const { data } = await apiClient.post<LegacyMigrationReport>('/admin/proxy/migrate-legacy')
  return data
}

export default {
  listNodes,
  createNode,
  importNodes,
  deleteNode,
  listProfiles,
  createProfile,
  updateProfile,
  startProfile,
  stopProfile,
  restartProfile,
  testProfile,
  getProfileRuntime,
  getRuntimeStatus,
  listBindings,
  createBinding,
  deleteBinding,
  migrateLegacy
}
