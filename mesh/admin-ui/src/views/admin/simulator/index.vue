<template>
  <PageContainer card>
    <div class="sim-page">
      <div class="sim-head">
        <div>
          <div class="sim-title">{{ t('admin.simulator.title') }}</div>
          <div class="sim-hint">{{ t('admin.simulator.hint') }}</div>
        </div>
        <div class="head-actions">
          <el-button :disabled="runningAll" @click="reset">{{ t('admin.simulator.reset') }}</el-button>
          <el-button
            :loading="runningAll"
            :disabled="runningAll || steps.login.status === 'running'"
            @click="runAll"
          >{{ t('admin.simulator.runAll') }}</el-button>
          <el-button
            type="primary"
            :loading="runningAll"
            :disabled="runningAll || steps.login.status === 'running'"
            @click="runToLobby"
          >{{ t('admin.simulator.runLobby') }}</el-button>
        </div>
      </div>

      <el-alert :title="t('admin.simulator.note')" type="info" :closable="false" show-icon class="sim-note" />

      <section class="sim-section">
        <div class="section-title">{{ t('admin.simulator.formTitle') }}</div>
        <el-form label-width="110px" @submit.prevent>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="24" :md="12">
              <el-form-item :label="t('admin.simulator.loginAddr')">
                <el-input v-model="form.loginAddr" :placeholder="t('admin.simulator.loginAddrPlaceholder')" clearable />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="24" :md="12">
              <el-form-item :label="t('admin.simulator.mode')">
                <el-radio-group v-model="form.mode">
                  <el-radio value="login">{{ t('admin.simulator.modeLogin') }}</el-radio>
                  <el-radio value="register">{{ t('admin.simulator.modeRegister') }}</el-radio>
                </el-radio-group>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :md="6">
              <el-form-item :label="t('admin.simulator.username')">
                <el-input v-model="form.username" :placeholder="t('admin.simulator.usernamePlaceholder')" clearable />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :md="6">
              <el-form-item :label="t('admin.simulator.password')">
                <el-input
                  v-model="form.password"
                  type="password"
                  show-password
                  :placeholder="t('admin.simulator.passwordPlaceholder')"
                />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="24" :md="12">
              <el-form-item :label="t('admin.simulator.gateAddr')">
                <el-input v-model="form.gateAddr" :placeholder="t('admin.simulator.gateAddrPlaceholder')" clearable />
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
      </section>

      <section class="sim-section">
        <div class="section-title">{{ t('admin.simulator.sessionTitle') }}</div>
        <el-descriptions v-if="session.id" :column="2" border size="small">
          <el-descriptions-item :label="t('admin.simulator.playerId')">
            {{ session.playerId || '-' }}
          </el-descriptions-item>
          <el-descriptions-item :label="t('admin.simulator.gateAddrLabel')">
            {{ session.gateAddr }}
          </el-descriptions-item>
          <el-descriptions-item :label="t('admin.simulator.token')" :span="2">
            <span class="token">{{ session.token }}</span>
          </el-descriptions-item>
        </el-descriptions>
        <div v-else class="empty">{{ t('admin.simulator.noSession') }}</div>
      </section>

      <div class="step-list">
        <div v-for="(def, index) in stepDefs" :key="def.key" class="step" :class="`step--${steps[def.key].status}`">
          <div class="step-head">
            <span class="step-no">{{ index + 1 }}</span>
            <div class="step-main">
              <div class="step-title">{{ def.title }}</div>
              <div class="step-desc">{{ def.desc }}</div>
            </div>
            <el-tag v-if="def.route" size="small" type="info" effect="plain">route {{ def.route }}</el-tag>
            <span v-if="steps[def.key].takenMs !== undefined" class="step-taken">
              {{ t('admin.simulator.taken', { ms: steps[def.key].takenMs }) }}
            </span>
            <el-tag size="small" :type="statusTagType(steps[def.key].status)">
              {{ statusText(steps[def.key].status) }}
            </el-tag>
            <el-button
              size="small"
              type="primary"
              plain
              :disabled="!canRun(def.key)"
              :loading="steps[def.key].status === 'running'"
              @click="runStep(def.key)"
            >{{ steps[def.key].status === 'success' ? t('admin.simulator.rerun') : t('admin.simulator.run') }}</el-button>
          </div>

          <div v-if="steps[def.key].error" class="step-error">
            {{ t('admin.simulator.error') }}: {{ steps[def.key].error }}
          </div>

          <div v-if="steps[def.key].request || steps[def.key].response" class="step-io">
            <div class="io-pane">
              <div class="io-title">{{ t('admin.simulator.request') }}</div>
              <pre class="io-body">{{ pretty(steps[def.key].request) || t('admin.simulator.emptyIo') }}</pre>
            </div>
            <div class="io-pane">
              <div class="io-title">{{ t('admin.simulator.response') }}</div>
              <pre class="io-body">{{ pretty(steps[def.key].response) || t('admin.simulator.emptyIo') }}</pre>
            </div>
          </div>
        </div>
      </div>
    </div>
  </PageContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import PageContainer from '@/components/PageContainer/index.vue'
import { getSimOptions, simClose, simConnect, simLogin, simStep } from '@/api/admin/sim'
import { asReportedError } from '@/utils/request'

defineOptions({ name: 'ClientSimulator' })

type StepKey = 'login' | 'connect' | 'verify' | 'enter' | 'leave'
type StepStatus = 'idle' | 'running' | 'success' | 'error'

interface StepState {
  status: StepStatus
  request?: unknown
  response?: unknown
  error?: string
  takenMs?: number
}

const { t } = useI18n()

const form = reactive({
  mode: 'login' as 'login' | 'register',
  username: '',
  password: '',
  loginAddr: '',
  gateAddr: ''
})

const session = reactive({
  id: '',
  playerId: 0,
  token: '',
  gateAddr: ''
})

const connected = ref(false)
const verified = ref(false)
const entered = ref(false)
const runningAll = ref(false)

const steps = reactive<Record<StepKey, StepState>>({
  login: { status: 'idle' },
  connect: { status: 'idle' },
  verify: { status: 'idle' },
  enter: { status: 'idle' },
  leave: { status: 'idle' }
})

const stepDefs = computed(() => [
  { key: 'login' as StepKey, title: t('admin.simulator.stepLogin'), desc: t('admin.simulator.stepLoginDesc'), route: 0 },
  { key: 'connect' as StepKey, title: t('admin.simulator.stepConnect'), desc: t('admin.simulator.stepConnectDesc'), route: 0 },
  { key: 'verify' as StepKey, title: t('admin.simulator.stepVerify'), desc: t('admin.simulator.stepVerifyDesc'), route: 10000 },
  { key: 'enter' as StepKey, title: t('admin.simulator.stepEnter'), desc: t('admin.simulator.stepEnterDesc'), route: 10001 },
  { key: 'leave' as StepKey, title: t('admin.simulator.stepLeave'), desc: t('admin.simulator.stepLeaveDesc'), route: 10002 }
])

const flow: StepKey[] = ['login', 'connect', 'verify', 'enter', 'leave']
/** 进入大厅到「进入游戏」为止，不走退出。 */
const lobbyFlow: StepKey[] = ['login', 'connect', 'verify', 'enter']

const statusText = (status: StepStatus) =>
  t(`admin.simulator.status${status.charAt(0).toUpperCase()}${status.slice(1)}`)

const statusTagType = (status: StepStatus) => {
  switch (status) {
    case 'success':
      return 'success'
    case 'error':
      return 'danger'
    case 'running':
      return 'primary'
    default:
      return 'info'
  }
}

const pretty = (value: unknown) => {
  if (value === undefined || value === null || value === '') return ''
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

const clearSteps = (keys: StepKey[]) => {
  for (const key of keys) {
    Object.assign(steps[key], {
      status: 'idle',
      request: undefined,
      response: undefined,
      error: undefined,
      takenMs: undefined
    } as StepState)
  }
}

const canRun = (key: StepKey) => {
  if (runningAll.value || steps[key].status === 'running') return false
  switch (key) {
    case 'login':
      return true
    case 'connect':
      return !!session.id && !connected.value
    case 'verify':
      return connected.value && !verified.value
    case 'enter':
      return verified.value && !entered.value
    case 'leave':
      return entered.value
  }
  return false
}

const closeSession = async (silent = false) => {
  if (!session.id) return
  const id = session.id
  session.id = ''
  connected.value = false
  verified.value = false
  entered.value = false
  await simClose(id).catch(() => null)
  if (!silent) ElMessage.success(t('admin.simulator.resetDone'))
}

const reset = async () => {
  await closeSession(true)
  session.playerId = 0
  session.token = ''
  session.gateAddr = ''
  clearSteps(flow)
}

const runStep = async (key: StepKey): Promise<boolean> => {
  if (!canRun(key)) return false

  const state = steps[key]
  state.status = 'running'
  state.error = undefined
  state.request = undefined
  state.response = undefined
  state.takenMs = undefined

  try {
    if (key === 'login') {
      if (!form.username.trim() || !form.password) {
        ElMessage.warning(t('admin.simulator.credentialsRequired'))
        state.status = 'idle'
        return false
      }

      await closeSession(true)
      const res = await simLogin({
        mode: form.mode,
        username: form.username.trim(),
        password: form.password,
        loginAddr: form.loginAddr.trim(),
        gateAddr: form.gateAddr.trim()
      })
      const data = res.data!
      session.id = data.sessionId
      session.playerId = data.playerId
      session.token = data.token
      session.gateAddr = data.gateAddr
      connected.value = false
      verified.value = false
      entered.value = false
      clearSteps(['connect', 'verify', 'enter', 'leave'])
      state.request = data.request
      state.response = data.response
      state.takenMs = data.takenMs
    } else if (key === 'connect') {
      const res = await simConnect(session.id)
      const data = res.data!
      connected.value = true
      verified.value = false
      entered.value = false
      clearSteps(['verify', 'enter', 'leave'])
      state.request = { sessionId: session.id, gateAddr: session.gateAddr }
      state.response = {
        gateAddr: data.gateAddr,
        remoteAddr: data.remoteAddr,
        localAddr: data.localAddr
      }
      state.takenMs = data.takenMs
    } else {
      const res = await simStep(session.id, key)
      const data = res.data!
      state.request = data.request
      state.response = data.response
      state.takenMs = data.takenMs
      if (key === 'verify') {
        verified.value = true
      } else if (key === 'enter') {
        entered.value = true
      } else {
        // 退出游戏后服务端已断开连接并销毁会话，后续步骤需要重新登录。
        connected.value = false
        verified.value = false
        entered.value = false
        session.id = ''
      }
    }
    state.status = 'success'
    return true
  } catch (error) {
    state.status = 'error'
    state.error = asReportedError(error).message
    return false
  }
}

const runFlow = async (keys: StepKey[], doneMessage: string) => {
  if (runningAll.value) return
  if (!form.username.trim() || !form.password) {
    ElMessage.warning(t('admin.simulator.credentialsRequired'))
    return
  }
  runningAll.value = true
  try {
    for (const key of keys) {
      if (!canRun(key)) break
      const ok = await runStep(key)
      if (!ok) return
    }
    ElMessage.success(doneMessage)
  } finally {
    runningAll.value = false
  }
}

const runAll = () => runFlow(flow, t('admin.simulator.flowDone'))

const runToLobby = () => runFlow(lobbyFlow, t('admin.simulator.lobbyDone'))

const loadOptions = async () => {
  const res = await getSimOptions().catch(() => null)
  if (!res?.data) return
  if (!form.loginAddr) form.loginAddr = res.data.loginAddr || ''
  if (!form.gateAddr) form.gateAddr = res.data.gateAddr || ''
}

onMounted(loadOptions)
onUnmounted(() => {
  void closeSession(true)
})
</script>

<style lang="scss" scoped>
.sim-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.sim-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.sim-title {
  font-size: 16px;
  font-weight: 600;
}

.sim-hint {
  margin-top: 4px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.head-actions {
  display: flex;
  gap: 8px;
  flex: none;
}

.sim-note {
  margin-bottom: 0;
}

.sim-section {
  padding: 12px 16px 4px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
}

.section-title {
  margin-bottom: 12px;
  font-size: 14px;
  font-weight: 600;
}

.empty {
  padding-bottom: 12px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.token {
  word-break: break-all;
  font-family: var(--el-font-family-mono, monospace);
  font-size: 12px;
}

.step-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.step {
  padding: 12px 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  background: var(--el-fill-color-blank);
}

.step--success {
  border-color: var(--el-color-success-light-5);
}

.step--error {
  border-color: var(--el-color-danger-light-5);
}

.step-head {
  display: flex;
  align-items: center;
  gap: 12px;
}

.step-no {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  flex: none;
  border-radius: 50%;
  background: var(--el-fill-color-light);
  font-size: 12px;
  font-weight: 600;
}

.step-main {
  flex: 1;
  min-width: 0;
}

.step-title {
  font-size: 14px;
  font-weight: 600;
}

.step-desc {
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.step-taken {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.step-error {
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-color-danger);
  word-break: break-all;
}

.step-io {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 12px;
  margin-top: 12px;
}

.io-title {
  margin-bottom: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.io-body {
  margin: 0;
  padding: 8px 10px;
  max-height: 320px;
  overflow: auto;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  font-family: var(--el-font-family-mono, monospace);
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}

@media (max-width: 900px) {
  .step-io {
    grid-template-columns: minmax(0, 1fr);
  }

  .step-head {
    flex-wrap: wrap;
  }
}
</style>
