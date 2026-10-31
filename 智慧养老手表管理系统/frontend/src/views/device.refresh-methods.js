import axios from 'axios'

export const deviceRefreshMethods = {
    loadOldPeople () {
      var _this = this
      axios.post('api/v1/inviteCode/oldPersonList', {}, {
        headers: { 'content-type': 'application/json' },
        withCredentials: true
      }).then(function (response) {
        _this.oldPeople = response.data.oldPeople || []
      }).catch(function (error) {
        console.log(error)
      })
    },
    startTimer () {
      this.stopTimer()
      if (!this.autoRefresh || document.hidden || this.refreshDisposed) return
      this.timer = setTimeout(() => {
        this.timer = null
        Promise.resolve(this.loadDevices()).then(() => { this.startTimer() })
      }, 5000)
    },
    handleVisibilityChange () {
      this.stopTimer()
      if (!document.hidden && this.autoRefresh) Promise.resolve(this.loadDevices()).then(() => { this.startTimer() })
    },
    stopTimer () {
      if (this.timer) {
        clearTimeout(this.timer)
        this.timer = null
      }
    },
}
