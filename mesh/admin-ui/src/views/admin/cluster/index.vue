<template>
  <PageContainer>
    <ProTable
      :table="table"
      row-key="nodeId"
      :default-sort="{ prop: 'nodeKind', order: 'ascending' }"
    >
      <template #search>
        <el-form-item :label="$t('admin.cluster.nodeName')">
          <el-input
            v-model="table.query.nodeName"
            :placeholder="$t('admin.cluster.nodeNamePlaceholder')"
            clearable
            @keyup.enter="table.search"
          />
        </el-form-item>
        <el-form-item :label="$t('admin.cluster.nodeKind')">
          <el-select
            v-model="table.query.nodeKind"
            :placeholder="$t('common.all')"
            clearable
            style="width: 140px"
          >
            <el-option
              v-for="kind in kinds"
              :key="kind.value"
              :label="kind.label"
              :value="kind.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('admin.cluster.online')">
          <el-select
            v-model="onlineFilter"
            :placeholder="$t('common.all')"
            clearable
            style="width: 140px"
          >
            <el-option :label="$t('admin.cluster.onlineYes')" :value="1" />
            <el-option :label="$t('admin.cluster.onlineNo')" :value="0" />
          </el-select>
        </el-form-item>
      </template>

      <template #toolbar>
        <el-tooltip :content="$t('common.refresh')" placement="top">
          <el-button
            circle
            :loading="table.loading"
            @click="refresh"
          >
            <el-icon><Refresh /></el-icon>
          </el-button>
        </el-tooltip>
        <span class="sync-hint">
          {{ $t('admin.cluster.syncHint', { n: table.rows.length }) }}
        </span>
      </template>

      <template #empty>
        <el-empty :description="$t('admin.cluster.emptyTip')" :image-size="120">
          <el-button type="primary" @click="refresh">{{ $t('admin.cluster.refreshSync') }}</el-button>
        </el-empty>
      </template>

      <el-table-column :label="$t('admin.cluster.nodeId')" prop="nodeId" min-width="200">
        <template #default="{ row }">
          <div class="cell-node-id">
            <el-tag effect="light" type="info" class="node-id-tag">
              {{ row.nodeId }}
            </el-tag>
            <el-tooltip :content="$t('admin.cluster.copyNodeId')" placement="top">
              <el-button link class="copy-btn" @click.stop="copyText(row.nodeId!)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </template>
      </el-table-column>

      <el-table-column :label="$t('admin.cluster.alias')" prop="alias" width="130" show-overflow-tooltip>
        <template #default="{ row }">
          <span :class="{ muted: !row.alias }">{{ row.alias || '-' }}</span>
        </template>
      </el-table-column>

      <el-table-column :label="$t('admin.cluster.nodeKind')" prop="nodeKind" width="100" align="center">
        <template #default="{ row }">
          <el-tag :type="kindType(row.nodeKind)" effect="plain" round>
            {{ kindLabel(row.nodeKind) }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column :label="$t('admin.cluster.online')" width="90" align="center">
        <template #default="{ row }">
          <div :class="['online-dot', row.online ? 'online-dot--up' : 'online-dot--down']">
            <span class="dot" />
            <span>{{ row.online ? $t('admin.cluster.onlineYes') : $t('admin.cluster.onlineNo') }}</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column :label="$t('admin.cluster.services')" min-width="180">
        <template #default="{ row }">
          <span v-if="!row.services?.length" class="muted">-</span>
          <div v-else class="services">
            <el-tag
              v-for="s in row.services"
              :key="s"
              size="small"
              effect="light"
              class="service-tag"
            >{{ svcLabel(s) }}</el-tag>
          </div>
        </template>
      </el-table-column>

      <el-table-column :label="$t('admin.cluster.endpoint')" prop="endpoint" min-width="160" show-overflow-tooltip />

      <el-table-column :label="$t('admin.cluster.remark')" prop="remark" min-width="160" show-overflow-tooltip>
        <template #default="{ row }">
          <span :class="{ muted: !row.remark }">{{ row.remark || '-' }}</span>
        </template>
      </el-table-column>

      <el-table-column :label="$t('common.operation')" width="230" fixed="right">
        <template #default="{ row }">
          <el-button
            v-permisaction="['game:cluster:list']"
            link
            type="warning"
            :disabled="!canExec(row)"
            :loading="execLoading[row.nodeId!]"
            @click="execAction(row, 'deploy')"
          >{{ $t('admin.cluster.actionDeploy') }}</el-button>
          <el-button
            v-permisaction="['game:cluster:list']"
            link
            type="success"
            :disabled="!canExec(row)"
            :loading="execLoading[row.nodeId!]"
            @click="execAction(row, 'start')"
          >{{ $t('admin.cluster.actionStart') }}</el-button>
          <el-button
            v-permisaction="['game:cluster:list']"
            link
            type="danger"
            :disabled="!canExec(row)"
            :loading="execLoading[row.nodeId!]"
            @click="execAction(row, 'stop')"
          >{{ $t('admin.cluster.actionStop') }}</el-button>
        </template>
      </el-table-column>
    </ProTable>

    <!-- 远程执行回显弹窗 -->
    <el-dialog
      v-model="execDialog.visible"
      :title="$t('admin.cluster.execTitle', { action: actionLabel(execDialog.action) })"
      width="760px"
      destroy-on-close
    >
      <div class="exec-meta">
        <span class="exec-meta__node">
          <el-tag type="info" size="small">{{ execDialog.nodeId }}</el-tag>
        </span>
        <span
          :class="['exec-meta__exit', exitClass(execDialog.exitCode)]"
        >
          {{ $t('admin.cluster.exitCode') }} = {{ execDialog.exitCode }}
        </span>
      </div>
      <div class="exec-panes">
        <div class="exec-pane">
          <div class="exec-pane__title">{{ $t('admin.cluster.stdout') }}</div>
          <pre class="exec-pane__code">{{ execDialog.stdout || $t('admin.cluster.execEmpty') }}</pre>
        </div>
        <div class="exec-pane">
          <div class="exec-pane__title exec-pane__title--err">{{ $t('admin.cluster.stderr') }}</div>
          <pre class="exec-pane__code exec-pane__code--err">{{ execDialog.stderr || $t('admin.cluster.execEmpty') }}</pre>
        </div>
      </div>
      <template #footer>
        <el-button @click="copyExec">{{ $t('admin.cluster.copyExec') }}</el-button>
        <el-button @click="execDialog.visible = false">{{ $t('common.dialogCancel') }}</el-button>
      </template>
    </el-dialog>
  </PageContainer>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { CopyDocument, Refresh } from '@element-plus/icons-vue'

import PageContainer from '@/components/PageContainer/index.vue'
import ProTable from '@/components/ProTable/index.vue'
import { useTable } from '@/composables'

import { listClusterNodes, execClusterNode } from '@/api/admin/cluster'
import type { ClusterNode, ClusterExecResp } from '@/types/admin'

defineOptions({ name: 'ClusterManage' })

const { t } = useI18n()

// ── 列表 ────────────────────────────────────────────────

const kinds = [
  { value: 'gate', label: t('admin.cluster.kindGate') },
  { value: 'node', label: t('admin.cluster.kindNode') },
  { value: 'mesh', label: t('admin.cluster.kindMesh') },
  { value: 'master', label: t('admin.cluster.kindMaster') }
]

// nodeKind 枚举 → 中文标签
const kindLabel = (kind?: string): string => {
  switch (kind) {
    case 'gate': return t('admin.cluster.kindGate')
    case 'node': return t('admin.cluster.kindNode')
    case 'mesh': return t('admin.cluster.kindMesh')
    case 'master': return t('admin.cluster.kindMaster')
    default: return kind || '-'
  }
}

// pb 内部服务全名（如 "internal.GameAdminService"）→ 中文显示
const svcLabel = (svc?: string): string => {
  if (!svc) return '-'
  // 取最后一段，如 "internal.GameAdminService" → "GameAdminService"
  const short = svc.includes('.') ? svc.split('.').pop()! : svc
  const key = `admin.cluster.svc_${short}`
  return t(key) === key ? short : t(key)
}

const table = useTable<ClusterNode, {
  nodeName?: string
  nodeKind?: string
  online?: number
  nodeIdOrder?: string
  nodeNameOrder?: string
  nodeKindOrder?: string
  onlineOrder?: string
}>({
  api: listClusterNodes,
  idKey: 'nodeId',
  defaultSort: { prop: 'nodeKind', order: 'ascending' },
  defaultQuery: () => ({ nodeName: undefined, nodeKind: undefined, online: undefined })
})

// ProTable 的 search slot 直接改 query 对象即可，等 ProTable 触发 search 时用最新值
const onlineFilter = ref<number | undefined>(undefined)
watch(onlineFilter, (v) => { table.query.online = v; table.search() })

const refresh = () => table.search()

const kindType = (kind?: string): 'primary' | 'success' | 'warning' | 'info' | '' => {
  switch (kind) {
    case 'gate': return 'primary'
    case 'node': return 'success'
    case 'mesh': return 'warning'
    default: return 'info'
  }
}

const canExec = (row: ClusterNode) =>
  !!row.endpoint && row.alias !== 'admin'

// ── 远程执行 ────────────────────────────────────────────

const execLoading = reactive<Record<string, boolean>>({})

const execDialog = reactive({
  visible: false,
  nodeId: '',
  action: '',
  exitCode: -1,
  stdout: '',
  stderr: ''
})

const actionLabel = (a: string) => {
  switch (a) {
    case 'start': return t('admin.cluster.actionStart')
    case 'stop': return t('admin.cluster.actionStop')
    case 'restart': return t('admin.cluster.actionRestart')
    case 'deploy': return t('admin.cluster.actionDeploy')
    default: return a
  }
}

const execAction = async(row: ClusterNode, action: string) => {
  if (!canExec(row)) {
    ElMessage.warning(t('admin.cluster.sshConfigMissing'))
    return
  }
  try {
    await ElMessageBox.confirm(
      t('admin.cluster.execConfirm', { node: row.nodeId, action: actionLabel(action) }),
      t('common.notice'),
      { type: 'warning', confirmButtonText: t('admin.cluster.execConfirmOk') }
    )
  } catch {
    return
  }
  execLoading[row.nodeId!] = true
  execDialog.visible = false
  const res = await execClusterNode(row.nodeId!, action).catch(() => null)
  execLoading[row.nodeId!] = false
  if (res?.data) {
    fillExecDialog(res.data)
  } else {
    fillExecDialog({
      nodeId: row.nodeId,
      action,
      exitCode: -1,
      stdout: '',
      stderr: t('admin.cluster.execNetworkError')
    })
  }
  execDialog.visible = true
  refresh()
}

const fillExecDialog = (resp: ClusterExecResp) => {
  execDialog.nodeId = resp.nodeId ?? ''
  execDialog.action = resp.action ?? ''
  execDialog.exitCode = resp.exitCode ?? -1
  execDialog.stdout = resp.stdout ?? ''
  execDialog.stderr = resp.stderr ?? ''
}

const exitClass = (code?: number) => {
  if (code === undefined || code < 0) return 'exit-err'
  return code === 0 ? 'exit-ok' : 'exit-fail'
}

const copyExec = async() => {
  const text = `[$t('admin.cluster.action')=${actionLabel(execDialog.action)}] ${t('admin.cluster.exitCode')}=${execDialog.exitCode}\n--- ${t('admin.cluster.stdout')} ---\n${execDialog.stdout}\n--- ${t('admin.cluster.stderr')} ---\n${execDialog.stderr}`
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('admin.cluster.copyOk'))
  } catch {
    ElMessage.warning(t('admin.cluster.copyFailed'))
  }
}

// ── 复制 ────────────────────────────────────────────────

const copyText = async(text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('admin.cluster.copyOk'))
  } catch {
    ElMessage.warning(t('admin.cluster.copyFailed'))
  }
}
</script>

<style scoped>
/* ── 列表 ── */
.cell-node-id { display: flex; align-items: center; gap: 4px; min-width: 0; }
.node-id-tag { font-family: var(--el-font-family-mono, monospace); font-size: 12px; padding: 0 6px; }
.copy-btn { padding: 2px; color: var(--el-text-color-placeholder); }
.copy-btn:hover { color: var(--el-color-primary); }

.services { display: flex; flex-wrap: wrap; gap: 3px; }
.service-tag { font-size: 11px; }

.online-dot { display: inline-flex; align-items: center; gap: 4px; font-size: 12px; }
.online-dot .dot { width: 8px; height: 8px; border-radius: 50%; }
.online-dot--up { color: var(--el-color-success); }
.online-dot--up .dot { background: var(--el-color-success); box-shadow: 0 0 4px currentColor; }
.online-dot--down { color: var(--el-text-color-placeholder); }
.online-dot--down .dot { background: var(--el-text-color-placeholder); }

.muted { color: var(--el-text-color-placeholder); }

.ssh-chip { display: inline-flex; align-items: center; gap: 4px; font-family: var(--el-font-family-mono, monospace); font-size: 12px; color: var(--el-color-primary); }
.ssh-chip__icon { font-size: 14px; }

.sync-hint { margin-left: auto; font-size: 12px; color: var(--el-text-color-secondary); }

/* ── SSH 弹窗 ── */
.form-tip { margin-top: 2px; font-size: 12px; color: var(--el-text-color-secondary); line-height: 1.4; }

/* ── 执行回显弹窗 ── */
.exec-meta { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; font-size: 13px; }
.exec-meta__exit { font-family: var(--el-font-family-mono, monospace); font-weight: 500; margin-left: auto; padding: 2px 8px; border-radius: 4px; }
.exit-ok { background: var(--el-color-success-light-9); color: var(--el-color-success); }
.exit-fail { background: var(--el-color-danger-light-9); color: var(--el-color-danger); }
.exit-err { background: var(--el-color-warning-light-9); color: var(--el-color-warning); }

.exec-panes { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.exec-pane { border: 1px solid var(--el-border-color-lighter); border-radius: 6px; overflow: hidden; display: flex; flex-direction: column; max-height: 360px; }
.exec-pane__title { font-size: 11px; padding: 4px 10px; background: var(--el-fill-color-lighter); color: var(--el-text-color-secondary); letter-spacing: 0.5px; }
.exec-pane__title--err { color: var(--el-color-danger); }
.exec-pane__code { flex: 1; margin: 0; padding: 10px; background: #1e1e1e; color: #ddd; font-family: var(--el-font-family-mono, monospace); font-size: 12px; line-height: 1.5; overflow: auto; white-space: pre-wrap; word-break: break-all; }
.exec-pane__code--err { color: #ff9a9a; }

@media (max-width: 720px) {
  .exec-panes { grid-template-columns: 1fr; }
}
</style>
