import { defineStore } from 'pinia'

export const useUiStore = defineStore('ui', {
  state: () => ({ sidebarOpen: true, mobileDrawerOpen: false }),
  actions: {
    toggleSidebar() { this.sidebarOpen = !this.sidebarOpen },
    openDrawer() { this.mobileDrawerOpen = true },
    closeDrawer() { this.mobileDrawerOpen = false },
  },
})
