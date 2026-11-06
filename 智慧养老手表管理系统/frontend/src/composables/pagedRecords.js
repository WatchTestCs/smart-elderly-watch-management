import { api } from '../api/client'
import { apiErrorMessage } from '../api/httpContract'
import { createLatestRead } from './latestRead'

export function pagedRecords({ path, responseKey, recordsKey, searchKey, load, failure, trim, filters }) {
  return {
    data () {
      return { [recordsKey]: [], isLoading: false, loadError: '', total: 0, pageSize: 10, currentPage: 1 }
    },
    created () {
      this._recordReads = createLatestRead(
        (query, signal) => api.post(path, query, { signal }),
        ({ data }) => {
          const records = data[responseKey]
          const total = data.total ?? records?.length
          if (!Array.isArray(records) || !Number.isSafeInteger(total) || total < 0) throw new Error('invalid records')
          // 删除了末页最后一条时当前页会变空：回退到新的最后一页，而不是停在空白页。
          const lastPage = Math.max(1, Math.ceil(total / this.pageSize))
          if (!records.length && total > 0 && lastPage < this.currentPage) {
            this.currentPage = lastPage
            this.$nextTick(() => this[load](true))
            return
          }
          this[recordsKey] = records
          this.total = total
        },
        error => {
          if (!failure) {
            this.$message.error(apiErrorMessage(error, '请求未完成，请重试'))
            return
          }
          this.loadError = apiErrorMessage(error, failure)
          this[recordsKey] = []
          this.total = 0
        },
        loading => { this.isLoading = loading }
      )
    },
    beforeUnmount () { this._recordReads.dispose() },
    methods: {
      [load] (keepPage = false) {
        if (keepPage !== true) this.currentPage = 1
        this.loadError = ''
        const query = { ...filters, page: this.currentPage, pageSize: this.pageSize }
        if (searchKey) query[searchKey] = trim ? this.searchText.trim() : this.searchText
        return this._recordReads.run(query)
      },
      handleSizeChange (size) { this.pageSize = size; this[load]() },
      changePage (page) { this.currentPage = page; this[load](true) }
    }
  }
}
