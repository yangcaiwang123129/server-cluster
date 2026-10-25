/**
 * The account pages: the summary card and its two tabs.
 *
 * Every Chinese value is byte-for-byte what the interface renders today, except
 * one that was not: the email rule used to read `'请输入正确的邮箱地址`, with a
 * stray leading apostrophe that reached the screen. It was carried across as-is
 * during the migration and corrected afterwards, so that the change to the
 * wording and the change to where the wording lives stayed separate.
 */
export default {
  title: '个人信息',
  username: '用户名称',
  phone: '手机号码',
  email: '用户邮箱',
  role: '所属角色',
  createdAt: '创建日期',
  noRole: '暂无',

  basic: '基本资料',
  tabs: {
    info: '基本资料',
    password: '修改密码'
  },

  info: {
    nickName: '用户昵称',
    phone: '手机号码',
    email: '邮箱',
    sex: '性别',
    male: '男',
    female: '女',
    rules: {
      nickName: '用户昵称不能为空',
      emailRequired: '邮箱地址不能为空',
      emailFormat: '请输入正确的邮箱地址',
      phoneRequired: '手机号码不能为空',
      phoneFormat: '请输入正确的手机号码'
    }
  },

  password: {
    old: '旧密码',
    oldPlaceholder: '请输入旧密码',
    new: '新密码',
    newPlaceholder: '请输入新密码',
    confirm: '确认密码',
    confirmPlaceholder: '请确认密码',
    rules: {
      oldRequired: '旧密码不能为空',
      newRequired: '新密码不能为空',
      length: '长度在 6 到 20 个字符',
      confirmRequired: '确认密码不能为空',
      mismatch: '两次输入的密码不一致'
    }
  },

  avatar: {
    hint: '头像'
  },

  save: '保存',
  close: '关闭'
}
