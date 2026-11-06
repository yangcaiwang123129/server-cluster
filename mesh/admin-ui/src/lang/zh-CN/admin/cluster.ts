export default {
  title: '集群管理',
  hint: '服务器从 etcd 自动同步入库；SSH 配置运维手动录入',

  // ── 列表筛选 ──
  nodeName: '服务名',
  nodeNamePlaceholder: 'gate/node/mesh',
  nodeKind: '类型',
  kindGate: '网关',
  kindNode: '节点',
  kindMesh: '网格',
  kindMaster: '主控',
  online: '在线',
  onlineYes: '在线',
  onlineNo: '离线',

  // ── 列 ──
  nodeId: '节点ID',
  alias: '部署别名',
  services: '内部服务',
  // pb 内部服务名 → 中文（services 列渲染时 lookup）
  svc_GameAdminService: 'GM 管理',
  svc_GamePlayerService: '玩家服务',
  svc_GiftService: '礼包服务',
  svc_ExchangeService: '兑换服务',
  svc_RechargeService: '充值服务',
  endpoint: '内部端点',
  sshHost: 'SSH主机',
  sshPort: '端口',
  sshUser: 'SSH用户',
  sshKeyPath: '私钥路径',
  deployDir: '部署目录',
  remark: '备注',
  updatedAt: '更新时间',

  sshUnconfigured: '未配置 SSH',

  // ── 工具栏/提示 ──
  syncHint: '共 {n} 个节点',
  refreshSync: '立即同步',
  emptyTip: 'etcd 中暂无集群节点',

  copyNodeId: '复制节点ID',
  copyOk: '已复制',
  copyFailed: '复制失败，请手动复制',

  // ── 操作按钮 ──
  editSSH: 'SSH配置',
  actionStart: '开服',
  actionStop: '关服',
  actionRestart: '重启',
  actionDeploy: '部署',
  hotReload: '热更',

  // ── SSH 配置弹窗 ──
  sshTitle: 'SSH / 部署配置',
  sshKeyPathTip: '留空时走本机 SSH agent ($SSH_AUTH_SOCK)',
  deployDirTip: '服务器上含 start.sh / stop.sh 的目录，如 /opt/server/gate-1',
  sshRequired: '请填写 SSH 主机、用户和部署目录',
  sshSaved: '配置已保存',

  // ── 远程执行弹窗 ──
  execTitle: '远程运维：{action}',
  execConfirm: '在 {node} 上执行「{action}」？',
  execConfirmOk: '立即执行',
  sshConfigMissing: '请先配置 SSH 连接信息',
  execEmpty: '（无输出）',
  execNetworkError: 'SSH 连接失败，请检查网络与密钥',
  copyExec: '复制输出',
  exitCode: '退出码',
  stdout: '标准输出',
  stderr: '标准错误',
  action: '操作'
}
