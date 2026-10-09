import { describe, expect, it } from 'vitest'
import { createPinia } from 'pinia'
import { useUiStore } from '../ui'

describe('reading panels', () => {
  it('keeps theme and directory panels exclusive without changing desktop preference', () => {
    const ui = useUiStore(createPinia())
    ui.sidebarOpen = false
    ui.openDrawer()
    expect(ui.mobileDrawerOpen).toBe(true)
    ui.openTopics()
    expect(ui.mobileDrawerOpen).toBe(false)
    expect(ui.topicPickerOpen).toBe(true)
    ui.openDrawer()
    expect(ui.topicPickerOpen).toBe(false)
    expect(ui.mobileDrawerOpen).toBe(true)
    ui.closePanels()
    expect(ui.mobileDrawerOpen).toBe(false)
    expect(ui.topicPickerOpen).toBe(false)
    expect(ui.sidebarOpen).toBe(false)
  })
})
