export default {
  title: 'Profile',
  username: 'Username',
  phone: 'Phone Number',
  email: 'Email',
  role: 'Roles',
  createdAt: 'Created',
  noRole: 'None',

  basic: 'Basic Information',
  tabs: {
    info: 'Basic Information',
    password: 'Change Password'
  },

  info: {
    nickName: 'Nickname',
    phone: 'Phone Number',
    email: 'Email',
    sex: 'Gender',
    male: 'Male',
    female: 'Female',
    rules: {
      nickName: 'Nickname is required',
      emailRequired: 'Email is required',
      emailFormat: 'Enter a valid email address',
      phoneRequired: 'Phone number is required',
      phoneFormat: 'Enter a valid phone number'
    }
  },

  password: {
    old: 'Current Password',
    oldPlaceholder: 'Enter your current password',
    new: 'New Password',
    newPlaceholder: 'Enter new password',
    confirm: 'Confirm Password',
    confirmPlaceholder: 'Confirm new password',
    rules: {
      oldRequired: 'Current password is required',
      newRequired: 'New password is required',
      length: 'Must be between 6 and 20 characters',
      confirmRequired: 'Please confirm the password',
      mismatch: 'The two passwords do not match'
    }
  },

  avatar: {
    hint: 'Avatar'
  },

  save: 'Save',
  close: 'Close'
}
