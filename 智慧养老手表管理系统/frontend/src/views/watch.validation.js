export function checkThreshold (rule, value, callback) {
  if (value === '' || value === null || value === undefined) {
    return callback(new Error('请输入阈值'))
  }
  var n = Number(value)
  if (!Number.isInteger(n) || n < 1 || n > 300) {
    return callback(new Error('阈值须为1-300的整数'))
  }
  callback()
}
