import Vue from 'vue'
import Clipboard from 'clipboard'

/**
 * 复制成功
 * @returns {Void}
 */
function clipboardSuccess() {
  Vue.prototype.$message({
    message: 'Copy successfully',
    type: 'success',
    duration: 1500
  })
}

/**
 * 复制失败
 * @returns {Void}
 */
function clipboardError() {
  Vue.prototype.$message({
    message: 'Copy failed',
    type: 'error'
  })
}

/**
 * 复制
 * @param {String} text 文本
 * @param {Object} event 事件
 * @returns {Void}
 */
export default function handleClipboard(text, event) {
  const clipboard = new Clipboard(event.target, {
    text: () => text
  })
  clipboard.on('success', () => {
    clipboardSuccess()
    clipboard.destroy()
  })
  clipboard.on('error', () => {
    clipboardError()
    clipboard.destroy()
  })
  clipboard.onClick(event)
}
