import { defineStore } from 'pinia'

export const useUiStore = defineStore('ui', {
  state: () => ({ sidebarOpen: true, mobileDrawerOpen: false, topicPickerOpen: false }),
  actions: {
    toggleSidebar() { this.sidebarOpen = !this.sidebarOpen },
    openDrawer() { this.topicPickerOpen = false; this.mobileDrawerOpen = true },
    closeDrawer() { this.mobileDrawerOpen = false },
    openTopics() { this.mobileDrawerOpen = false; this.topicPickerOpen = true },
    closeTopics() { this.topicPickerOpen = false },
    closePanels() { this.mobileDrawerOpen = false; this.topicPickerOpen = false },
  },
})
