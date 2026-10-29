/**
 * 客户端版本管理：发布 U3D 安装包清单。菜单标题来自数据库，不在这里。
 */
export default {
  versionId: '版本ID',
  version: '版本号',
  versionPlaceholder: '请输入版本号',
  platform: '平台',
  platformPlaceholder: '请选择平台',
  platformAndroid: 'Android',
  platformIOS: 'iOS',
  platformWindows: 'Windows',
  downloadUrl: '下载地址',
  downloadUrlPlaceholder: '请输入安装包下载地址',
  forceUpdate: '强制更新',
  status: '状态',
  yes: '是',
  no: '否',
  statusOk: '状态已更新',
  releaseNote: '更新说明',
  releaseNotePlaceholder: '请输入更新说明',
  addTitle: '新增版本',
  editTitle: '修改版本',
  rules: {
    version: '版本号不能为空',
    platform: '平台不能为空',
    downloadUrl: '下载地址不能为空'
  }
}
