import { onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import { api } from '../api/client'
import { apiErrorMessage } from '../api/httpContract'
import { createLatestRead } from './latestRead'

export interface PagedOptions<TQuery extends Record<string, unknown>> {
  path: string
  responseKey: string
  buildQuery?: () => TQuery
  initialPageSize?: number
  failure?: string
}

export function usePagedRecords<TRecord, TQuery extends Record<string, unknown> = Record<string, unknown>>(
  options: PagedOptions<TQuery>
) {
  type PageQuery = TQuery & { page: number; pageSize: number }
  const records = shallowRef<TRecord[]>([])
  const isLoading = ref(false)
  const loadError = ref('')
  const total = ref(0)
  const pageSize = ref(options.initialPageSize ?? 10)
  const currentPage = ref(1)

  function pageQuery(): PageQuery {
    return { ...(options.buildQuery?.() ?? {}), page: currentPage.value, pageSize: pageSize.value } as PageQuery
  }

  const reader = createLatestRead(
    (query: PageQuery, signal: AbortSignal) =>
      api.post<Record<string, unknown>>(options.path, query, { signal }),
    ({ data }) => {
      const list = data[options.responseKey]
      const count = data.total ?? (Array.isArray(list) ? list.length : undefined)
      if (!Array.isArray(list) || typeof count !== 'number' || !Number.isSafeInteger(count) || count < 0) {
        throw new Error('invalid paged response')
      }
      // A deletion can invalidate the current page, including when the collection becomes empty.
      const lastPage = Math.max(1, Math.ceil(count / pageSize.value))
      if (currentPage.value > lastPage) {
        currentPage.value = lastPage
        void reader.run(pageQuery())
        return
      }
      records.value = list as TRecord[]
      total.value = count
      loadError.value = ''
    },
    error => {
      records.value = []
      total.value = 0
      loadError.value = apiErrorMessage(error, options.failure ?? '请求未完成，请重试')
    },
    loading => { isLoading.value = loading }
  )

  function load(keepPage = false): Promise<void> {
    if (keepPage !== true) currentPage.value = 1
    loadError.value = ''
    return reader.run(pageQuery())
  }

  function handleSizeChange(size: number): Promise<void> {
    pageSize.value = size
    return load()
  }

  function changePage(page: number): Promise<void> {
    currentPage.value = page
    return load(true)
  }

  onMounted(() => { void load(true) })
  onBeforeUnmount(reader.dispose)

  return { records, isLoading, loadError, total, pageSize, currentPage, load, handleSizeChange, changePage }
}
