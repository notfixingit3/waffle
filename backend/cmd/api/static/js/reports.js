(function () {
  'use strict'

  var state = {
    activeTab: 'drought',
    datePreset: 'this-month',
    customFrom: '',
    customTo: '',
    velocityFilter: '',
    droughtData: null,
    powerBuyerData: null,
    monthlyData: null,
    velocityData: null,
    loading: false
  }

  var cache = {}

  function $(sel) { return document.querySelector(sel) }
  function $$(sel) { return document.querySelectorAll(sel) }

  function fmtDate(d) {
    var y = d.getFullYear()
    var m = String(d.getMonth() + 1).padStart(2, '0')
    var day = String(d.getDate()).padStart(2, '0')
    return y + '-' + m + '-' + day
  }

  function getDateRange() {
    var now = new Date()
    var to = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 23, 59, 59)

    switch (state.datePreset) {
      case 'this-month': {
        var from = new Date(now.getFullYear(), now.getMonth(), 1)
        return { from: fmtDate(from), to: fmtDate(to) }
      }
      case 'last-month': {
        var from = new Date(now.getFullYear(), now.getMonth() - 1, 1)
        var lastTo = new Date(now.getFullYear(), now.getMonth(), 0)
        return { from: fmtDate(from), to: fmtDate(lastTo) }
      }
      case 'last-3-months': {
        var from = new Date(now.getFullYear(), now.getMonth() - 3, 1)
        return { from: fmtDate(from), to: fmtDate(to) }
      }
      case 'all-time':
        return { from: '2000-01-01', to: fmtDate(to) }
      case 'custom':
        return {
          from: state.customFrom || '2024-01-01',
          to: state.customTo || fmtDate(to)
        }
      default:
        return { from: fmtDate(new Date(now.getFullYear(), now.getMonth(), 1)), to: fmtDate(to) }
    }
  }

  function formatMoney(cents) {
    return '$' + cents.toLocaleString()
  }

  function showLoading() {
    state.loading = true
    $('#reports-loading').classList.remove('hidden')
    $('#reports-error').classList.add('hidden')
  }

  function hideLoading() {
    state.loading = false
    $('#reports-loading').classList.add('hidden')
  }

  function showError(msg) {
    $('#reports-error').classList.remove('hidden')
    $('#reports-error-msg').textContent = msg
  }

  function apiURL(path, params) {
    var base = '/api/admin/reports/' + path
    if (params) {
      var qs = []
      for (var k in params) {
        if (params.hasOwnProperty(k) && params[k] !== '' && params[k] !== undefined) {
          qs.push(encodeURIComponent(k) + '=' + encodeURIComponent(params[k]))
        }
      }
      if (qs.length) base += '?' + qs.join('&')
    }
    return base
  }

  function fetchJSON(url) {
    return fetch(url, { credentials: 'same-origin' }).then(function (res) {
      if (!res.ok) return res.json().then(function (e) { throw new Error(e.error || 'Request failed') })
      return res.json()
    })
  }

  function loadTabData(tab) {
    if (tab === 'drought') return loadDrought()
    if (tab === 'power-buyers') return loadPowerBuyers()
    if (tab === 'monthly') return loadMonthly()
    if (tab === 'velocity') return loadVelocity()
    return Promise.resolve()
  }

  function loadDrought() {
    var dr = getDateRange()
    var cacheKey = 'drought:' + dr.from + ':' + dr.to
    if (cache[cacheKey]) {
      state.droughtData = cache[cacheKey]
      renderDrought()
      return Promise.resolve()
    }
    return fetchJSON(apiURL('drought', { from: dr.from, to: dr.to })).then(function (data) {
      var entries = data.entries || []
      cache[cacheKey] = entries
      state.droughtData = entries
      renderDrought()
    }).catch(function (err) {
      showError('Failed to load drought data: ' + err.message)
    })
  }

  function loadPowerBuyers() {
    var dr = getDateRange()
    var cacheKey = 'power-buyers:' + dr.from + ':' + dr.to
    if (cache[cacheKey]) {
      state.powerBuyerData = cache[cacheKey]
      renderPowerBuyers()
      return Promise.resolve()
    }
    return fetchJSON(apiURL('power-buyers', { from: dr.from, to: dr.to, limit: 20 })).then(function (data) {
      var entries = data.entries || []
      cache[cacheKey] = entries
      state.powerBuyerData = entries
      renderPowerBuyers()
    }).catch(function (err) {
      showError('Failed to load power buyer data: ' + err.message)
    })
  }

  function loadMonthly() {
    var dr = getDateRange()
    var cacheKey = 'monthly:' + dr.from + ':' + dr.to
    if (cache[cacheKey]) {
      state.monthlyData = cache[cacheKey]
      renderMonthly()
      return Promise.resolve()
    }
    return fetchJSON(apiURL('monthly-activity', { from: dr.from, to: dr.to })).then(function (data) {
      var entries = data.entries || []
      cache[cacheKey] = entries
      state.monthlyData = entries
      renderMonthly()
    }).catch(function (err) {
      showError('Failed to load monthly activity: ' + err.message)
    })
  }

  function loadVelocity() {
    var cacheKey = 'velocity:' + (state.velocityFilter || 'all')
    if (cache[cacheKey]) {
      state.velocityData = cache[cacheKey]
      renderVelocity()
      return Promise.resolve()
    }
    var params = state.velocityFilter ? { status: state.velocityFilter } : {}
    return fetchJSON(apiURL('spot-velocity', params)).then(function (data) {
      var entries = data.entries || []
      cache[cacheKey] = entries
      state.velocityData = entries
      renderVelocity()
    }).catch(function (err) {
      showError('Failed to load velocity data: ' + err.message)
    })
  }

  function invalidateDateCache() {
    cache = {}
    state.droughtData = null
    state.powerBuyerData = null
    state.monthlyData = null
  }

  function switchTab(tab) {
    state.activeTab = tab
    $$('.tab').forEach(function (btn) {
      var isActive = btn.getAttribute('data-tab') === tab
      if (isActive) {
        btn.classList.add('tab-active')
      } else {
        btn.classList.remove('tab-active')
      }
    })
    $$('.tab-content').forEach(function (el) { el.classList.add('hidden') })
    $('#tab-' + tab).classList.remove('hidden')

    if (tab === 'velocity') {
      renderVelocityShell()
    }

    showLoading()
    loadTabData(tab).then(hideLoading).catch(function (err) {
      hideLoading()
      showError(err.message)
    })
  }

  // --- Date preset handlers ---

  function setDatePreset(preset) {
    state.datePreset = preset
    $$('.date-preset-btn').forEach(function (btn) {
      var isActive = btn.getAttribute('data-date-preset') === preset
      if (isActive) {
        btn.classList.remove('btn-outline')
        btn.classList.add('btn-primary')
      } else {
        btn.classList.remove('btn-primary')
        btn.classList.add('btn-outline')
      }
    })

    if (preset === 'custom') {
      $('#custom-date-range').classList.remove('hidden')
    } else {
      $('#custom-date-range').classList.add('hidden')
    }

    var dr = getDateRange()
    $('#date-range-label').textContent = dr.from + ' \u2014 ' + dr.to

    invalidateDateCache()
    showLoading()
    loadTabData(state.activeTab).then(hideLoading).catch(function (err) {
      hideLoading()
      showError(err.message)
    })
  }

  $$('.date-preset-btn').forEach(function (btn) {
    btn.addEventListener('click', function () {
      setDatePreset(btn.getAttribute('data-date-preset'))
    })
  })

  $('#custom-from').addEventListener('change', function () {
    state.customFrom = this.value
    if (state.datePreset === 'custom') {
      $('#date-range-label').textContent = getDateRange().from + ' \u2014 ' + getDateRange().to
      invalidateDateCache()
      showLoading()
      loadTabData(state.activeTab).then(hideLoading)
    }
  })

  $('#custom-to').addEventListener('change', function () {
    state.customTo = this.value
    if (state.datePreset === 'custom') {
      $('#date-range-label').textContent = getDateRange().from + ' \u2014 ' + getDateRange().to
      invalidateDateCache()
      showLoading()
      loadTabData(state.activeTab).then(hideLoading)
    }
  })

  // --- Tab switching ---

  $$('.tab-btn').forEach(function (btn) {
    btn.addEventListener('click', function () {
      switchTab(btn.getAttribute('data-tab'))
    })
  })

  // --- Renderers ---

  function renderDrought() {
    var container = $('#tab-drought')
    var entries = state.droughtData || []

    var html = '<div class="p-4 border-b border-base-200">' +
      '<h2 class="text-lg font-semibold text-base-content">Drought List</h2>' +
      '<p class="text-sm text-base-content/60">Users with entries but no recent wins</p>' +
      '</div>'

    if (entries.length === 0) {
      html += '<div class="p-8 text-center text-base-content/60 text-sm">No users with drought data in this range.</div>'
    } else {
      html += '<div class="overflow-x-auto"><table class="w-full text-sm">' +
        '<thead class="bg-base-200"><tr>' +
        '<th class="text-left px-4 py-3 font-medium text-base-content/60">Instagram Handle</th>' +
        '<th class="text-right px-4 py-3 font-medium text-base-content/60">Total Entries</th>' +
        '<th class="text-right px-4 py-3 font-medium text-base-content/60">Last Entry</th>' +
        '<th class="text-right px-4 py-3 font-medium text-base-content/60">Drought (Days)</th>' +
        '</tr></thead><tbody>'

      for (var i = 0; i < entries.length; i++) {
        var e = entries[i]
        var droughtClass = e.longest_drought >= 30 ? 'text-error' : (e.longest_drought >= 14 ? 'text-warning' : 'text-base-content/70')
        var droughtLabel = e.longest_drought >= 99999 ? 'Never won' : e.longest_drought
        var lastDate = new Date(e.last_entry_date).toLocaleDateString()
        html += '<tr class="border-t border-base-200 hover:bg-base-200">' +
          '<td class="px-4 py-3 font-medium text-base-content">@' + escHtml(e.instagram_handle) + '</td>' +
          '<td class="px-4 py-3 text-right text-base-content/70">' + e.total_entries + '</td>' +
          '<td class="px-4 py-3 text-right text-base-content/60">' + lastDate + '</td>' +
          '<td class="px-4 py-3 text-right"><span class="font-semibold ' + droughtClass + '">' + droughtLabel + '</span></td>' +
          '</tr>'
      }

      html += '</tbody></table></div>'
    }

    container.innerHTML = html
  }

  function renderPowerBuyers() {
    var container = $('#tab-power-buyers')
    var entries = state.powerBuyerData || []

    var html = '<div class="p-4 border-b border-base-200">' +
      '<h2 class="text-lg font-semibold text-base-content">Power Buyers</h2>' +
      '<p class="text-sm text-base-content/60">Top users by activity and spending</p>' +
      '</div>'

    if (entries.length === 0) {
      html += '<div class="p-8 text-center text-base-content/60 text-sm">No power buyer data in this range.</div>'
    } else {
      html += '<div class="overflow-x-auto"><table class="w-full text-sm">' +
        '<thead class="bg-base-200"><tr>' +
        '<th class="text-left px-4 py-3 font-medium text-base-content/60">Instagram Handle</th>' +
        '<th class="text-right px-4 py-3 font-medium text-base-content/60">Spots Claimed</th>' +
        '<th class="text-right px-4 py-3 font-medium text-base-content/60">Total Spent</th>' +
        '<th class="text-right px-4 py-3 font-medium text-base-content/60">Win Rate</th>' +
        '</tr></thead><tbody>'

      for (var i = 0; i < entries.length; i++) {
        var e = entries[i]
        var rateClass = e.win_rate >= 50 ? 'text-success' : (e.win_rate >= 20 ? 'text-warning' : 'text-base-content/60')
        html += '<tr class="border-t border-base-200 hover:bg-base-200">' +
          '<td class="px-4 py-3 font-medium text-base-content">@' + escHtml(e.instagram_handle) + '</td>' +
          '<td class="px-4 py-3 text-right text-base-content/70">' + e.total_spots_claimed + '</td>' +
          '<td class="px-4 py-3 text-right text-base-content/70">' + formatMoney(e.total_spent) + '</td>' +
          '<td class="px-4 py-3 text-right"><span class="font-semibold ' + rateClass + '">' + e.win_rate + '%</span></td>' +
          '</tr>'
      }

      html += '</tbody></table></div>'
    }

    container.innerHTML = html
  }

  function renderMonthly() {
    var container = $('#tab-monthly')
    var entries = state.monthlyData || []

    var html = '<div class="mb-4">' +
      '<h2 class="text-lg font-semibold text-base-content">Monthly Activity</h2>' +
      '<p class="text-sm text-base-content/60">Waffles, claims, and revenue by month</p>' +
      '</div>'

    if (entries.length === 0) {
      html += '<div class="p-8 text-center text-base-content/60 text-sm">No activity data in this range.</div>'
    } else {
      var barMax = 1
      for (var i = 0; i < entries.length; i++) {
        var m = entries[i]
        var v = Math.max(m.waffles, m.spots_claimed, m.revenue / 10)
        if (v > barMax) barMax = v
      }

      html += '<div class="space-y-4"><div class="space-y-2">'

      var totalWaffles = 0, totalClaims = 0, totalRevenue = 0

      for (var i = 0; i < entries.length; i++) {
        var m = entries[i]
        totalWaffles += m.waffles
        totalClaims += m.spots_claimed
        totalRevenue += m.revenue

        var wafflePct = (m.waffles / barMax * 30).toFixed(1)
        var claimPct = (m.spots_claimed / barMax * 30).toFixed(1)
        var revPct = (m.revenue / 10 / barMax * 30).toFixed(1)

        html += '<div class="flex items-center gap-3">' +
          '<span class="text-xs text-base-content/60 w-16 flex-shrink-0">' + escHtml(m.month) + '</span>' +
          '<div class="flex-1 flex items-end gap-1" style="height:32px">' +
          '<div class="bg-primary rounded-t" style="width:' + wafflePct + '%;min-width:' + (m.waffles > 0 ? '4px' : '0') + ';height:100%" title="' + m.waffles + ' waffles"></div>' +
          '<div class="bg-success rounded-t" style="width:' + claimPct + '%;min-width:' + (m.spots_claimed > 0 ? '4px' : '0') + ';height:100%" title="' + m.spots_claimed + ' spots claimed"></div>' +
          '<div class="bg-secondary rounded-t" style="width:' + revPct + '%;min-width:' + (m.revenue > 0 ? '4px' : '0') + ';height:100%" title="' + formatMoney(m.revenue) + ' revenue"></div>' +
          '</div>' +
          '<div class="flex gap-2 text-xs text-base-content/60 w-24 flex-shrink-0 justify-end">' +
          '<span class="text-primary font-medium">' + m.waffles + ' waff</span>' +
          '<span class="text-success font-medium">' + m.spots_claimed + ' clm</span>' +
          '<span class="text-secondary font-medium">' + formatMoney(m.revenue) + '</span>' +
          '</div>' +
          '</div>'
      }

      html += '</div>'

      html += '<div class="flex gap-4 pt-2 border-t border-base-200 text-xs">' +
        '<div class="flex items-center gap-1"><div class="w-3 h-3 bg-primary rounded"></div><span>Waffles</span></div>' +
        '<div class="flex items-center gap-1"><div class="w-3 h-3 bg-success rounded"></div><span>Claims</span></div>' +
        '<div class="flex items-center gap-1"><div class="w-3 h-3 bg-secondary rounded"></div><span>Revenue</span></div>' +
        '</div></div>'

      html += '<div class="mt-6 grid grid-cols-3 gap-4 p-4 bg-base-200 rounded-lg">' +
        '<div><p class="text-xs text-base-content/60">Total Waffles</p><p class="text-lg font-bold text-base-content">' + totalWaffles + '</p></div>' +
        '<div><p class="text-xs text-base-content/60">Total Claims</p><p class="text-lg font-bold text-base-content">' + totalClaims + '</p></div>' +
        '<div><p class="text-xs text-base-content/60">Total Revenue</p><p class="text-lg font-bold text-base-content">' + formatMoney(totalRevenue) + '</p></div>' +
        '</div>'
    }

    container.innerHTML = html
  }

  function renderVelocityShell() {
    var container = $('#tab-velocity')
    container.innerHTML = '<div class="p-4 border-b border-base-200 flex flex-wrap items-center justify-between gap-3">' +
      '<div><h2 class="text-lg font-semibold text-base-content">Spot Velocity</h2>' +
      '<p class="text-sm text-base-content/60">How fast waffles fill (hours)</p></div>' +
      '<select id="velocity-filter-sel" class="select select-bordered select-sm">' +
      '<option value="">All Statuses</option>' +
      '<option value="active"' + (state.velocityFilter === 'active' ? ' selected' : '') + '>Active</option>' +
      '<option value="completed"' + (state.velocityFilter === 'completed' ? ' selected' : '') + '>Completed</option>' +
      '</select></div>'

    document.getElementById('velocity-filter-sel').addEventListener('change', function () {
      state.velocityFilter = this.value
      cache = {}
      state.velocityData = null
      showLoading()
      loadVelocity().then(hideLoading)
    })
  }

  function renderVelocity() {
    var container = $('#tab-velocity')
    var entries = state.velocityData || []

    var html = '<div class="p-4 border-b border-base-200 flex flex-wrap items-center justify-between gap-3">' +
      '<div><h2 class="text-lg font-semibold text-base-content">Spot Velocity</h2>' +
      '<p class="text-sm text-base-content/60">How fast waffles fill (hours)</p></div>' +
      '<select id="velocity-filter-sel" class="select select-bordered select-sm">' +
      '<option value="">All Statuses</option>' +
      '<option value="active"' + (state.velocityFilter === 'active' ? ' selected' : '') + '>Active</option>' +
      '<option value="completed"' + (state.velocityFilter === 'completed' ? ' selected' : '') + '>Completed</option>' +
      '</select></div>'

    if (entries.length === 0) {
      html += '<div class="p-8 text-center text-base-content/60 text-sm">No velocity data available.</div>'
    } else {
      html += '<div class="overflow-x-auto"><table class="w-full text-sm">' +
        '<thead class="bg-base-200"><tr>' +
        '<th class="text-left px-4 py-3 font-medium text-base-content/60">Status</th>' +
        '<th class="text-right px-4 py-3 font-medium text-base-content/60">Waffles</th>' +
        '<th class="text-right px-4 py-3 font-medium text-base-content/60">Avg First Claim (hrs)</th>' +
        '<th class="text-right px-4 py-3 font-medium text-base-content/60">Avg Completion (hrs)</th>' +
        '</tr></thead><tbody>'

      for (var i = 0; i < entries.length; i++) {
        var v = entries[i]
        var statusBadge = v.status === 'active'
          ? '<span class="badge badge-success">' + escHtml(v.status) + '</span>'
          : '<span class="badge badge-secondary">' + escHtml(v.status) + '</span>'
        var completion = v.avg_completion_hours > 0 ? v.avg_completion_hours.toFixed(1) + 'h' : '\u2014'

        html += '<tr class="border-t border-base-200 hover:bg-base-200">' +
          '<td class="px-4 py-3">' + statusBadge + '</td>' +
          '<td class="px-4 py-3 text-right text-base-content/70">' + v.waffle_count + '</td>' +
          '<td class="px-4 py-3 text-right text-base-content/70">' + v.avg_first_claim_hours.toFixed(1) + 'h</td>' +
          '<td class="px-4 py-3 text-right text-base-content/70">' + completion + '</td>' +
          '</tr>'
      }

      html += '</tbody></table></div>'
    }

    container.innerHTML = html

    document.getElementById('velocity-filter-sel').addEventListener('change', function () {
      state.velocityFilter = this.value
      cache = {}
      state.velocityData = null
      showLoading()
      loadVelocity().then(hideLoading)
    })
  }

  function escHtml(str) {
    var div = document.createElement('div')
    div.appendChild(document.createTextNode(str))
    return div.innerHTML
  }

  // --- Init ---

  function init() {
    var dr = getDateRange()
    $('#date-range-label').textContent = dr.from + ' \u2014 ' + dr.to
    switchTab('drought')
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init)
  } else {
    init()
  }
})()
