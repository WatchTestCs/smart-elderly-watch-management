import { onMounted, ref } from 'vue'

const STORAGE_KEY = 'large_text'

function readStoredLargeText(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === 'true'
  } catch {
    return false
  }
}

const largeText = ref(readStoredLargeText())

function applyTextSizePreference(): void {
  if (typeof document === 'undefined') return
  document.documentElement.classList.toggle('large-text', largeText.value)
}

function setLargeText(value: boolean): void {
  largeText.value = value
  try {
    localStorage.setItem(STORAGE_KEY, String(value))
  } catch {
    // Keep the active session preference when storage is unavailable.
  }
  applyTextSizePreference()
}

export function useTextSizePreference() {
  onMounted(applyTextSizePreference)
  return {
    largeText,
    setLargeText,
    toggleTextSize: () => setLargeText(!largeText.value)
  }
}
