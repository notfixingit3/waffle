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
    loading: false,
    monthlyVisibility: { waffles: true, claims: true, revenue: true }
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

  $$('#tab-bar .tab').forEach(function (btn) {
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
        var lastDate = e.last_entry_date ? new Date(e.last_entry_date).toLocaleDateString() : '—'
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
  }  function renderMonthly() {
    var container = $('#tab-monthly')
    var entries = state.monthlyData || []

    var html = '<div class="mb-6 flex flex-wrap items-center justify-between gap-4">' +
      '<div>' +
      '<h2 class="text-lg font-semibold text-base-content">Monthly Activity</h2>' +
      '<p class="text-sm text-base-content/60">Waffles, claims, and revenue by month</p>' +
      '</div>'
      
    if (entries.length > 0) {
      var wVis = state.monthlyVisibility.waffles ? 'btn-primary' : 'btn-outline border-base-content/10 text-base-content/50'
      var cVis = state.monthlyVisibility.claims ? 'btn-success text-success-content' : 'btn-outline border-base-content/10 text-base-content/50'
      var rVis = state.monthlyVisibility.revenue ? 'btn-secondary text-secondary-content' : 'btn-outline border-base-content/10 text-base-content/50'
      
      html += '<div class="flex items-center gap-2">' +
        '<button id="toggle-waffles" class="btn btn-xs ' + wVis + '">Waffles</button>' +
        '<button id="toggle-claims" class="btn btn-xs ' + cVis + '">Claims</button>' +
        '<button id="toggle-revenue" class="btn btn-xs ' + rVis + '">Revenue</button>' +
        '</div>'
    }
    
    html += '</div>'

    if (entries.length === 0) {
      html += '<div class="p-8 text-center text-base-content/60 text-sm">No activity data in this range.</div>'
      container.innerHTML = html
      return
    }

    html += '<div class="relative w-full mb-6 p-4 bg-base-300/30 border border-base-content/5 rounded-xl overflow-hidden">' +
      '<div id="chart-tooltip" class="absolute hidden bg-base-300 border border-base-content/15 rounded-lg p-3 shadow-xl pointer-events-none text-xs z-30 transition-all duration-100 ease-out"></div>' +
      '<div id="monthly-svg-container" class="w-full" style="height: 350px;"></div>' +
      '</div>'

    var totalWaffles = 0, totalClaims = 0, totalRevenue = 0
    for (var i = 0; i < entries.length; i++) {
      totalWaffles += entries[i].waffles
      totalClaims += entries[i].spots_claimed
      totalRevenue += entries[i].revenue
    }

    html += '<div class="mt-6 grid grid-cols-3 gap-4 mb-6">' +
      '<div class="card bg-base-300/50 border border-base-content/5 p-4 rounded-xl shadow-sm">' +
      '<p class="text-xs text-base-content/50 uppercase font-bold tracking-wider">Total Waffles</p>' +
      '<p class="text-xl font-black text-primary mt-1">' + totalWaffles + '</p>' +
      '</div>' +
      '<div class="card bg-base-300/50 border border-base-content/5 p-4 rounded-xl shadow-sm">' +
      '<p class="text-xs text-base-content/50 uppercase font-bold tracking-wider">Total Claims</p>' +
      '<p class="text-xl font-black text-success mt-1">' + totalClaims + '</p>' +
      '</div>' +
      '<div class="card bg-base-300/50 border border-base-content/5 p-4 rounded-xl shadow-sm">' +
      '<p class="text-xs text-base-content/50 uppercase font-bold tracking-wider">Total Revenue</p>' +
      '<p class="text-xl font-black text-secondary mt-1">' + formatMoney(totalRevenue) + '</p>' +
      '</div>' +
      '</div>'

    html += '<div class="overflow-x-auto border border-base-content/5 rounded-xl bg-base-300/40"><table class="w-full text-sm">' +
      '<thead class="bg-base-300"><tr>' +
      '<th class="text-left px-4 py-3 font-medium text-base-content/60">Month</th>' +
      '<th class="text-right px-4 py-3 font-medium text-base-content/60">Waffles</th>' +
      '<th class="text-right px-4 py-3 font-medium text-base-content/60">Claims</th>' +
      '<th class="text-right px-4 py-3 font-medium text-base-content/60">Revenue</th>' +
      '</tr></thead><tbody>'

    for (var i = 0; i < entries.length; i++) {
      var e = entries[i]
      html += '<tr class="border-t border-base-content/5 hover:bg-base-300/30">' +
        '<td class="px-4 py-3 font-medium text-base-content">' + escHtml(e.month) + '</td>' +
        '<td class="px-4 py-3 text-right text-base-content/75">' + e.waffles + '</td>' +
        '<td class="px-4 py-3 text-right text-base-content/75">' + e.spots_claimed + '</td>' +
        '<td class="px-4 py-3 text-right text-base-content/75 font-semibold text-secondary">' + formatMoney(e.revenue) + '</td>' +
        '</tr>'
    }
    html += '</tbody></table></div>'

    container.innerHTML = html

    renderMonthlySVG(entries)

    $('#toggle-waffles').addEventListener('click', function() {
      state.monthlyVisibility.waffles = !state.monthlyVisibility.waffles
      renderMonthly()
    })
    $('#toggle-claims').addEventListener('click', function() {
      state.monthlyVisibility.claims = !state.monthlyVisibility.claims
      renderMonthly()
    })
    $('#toggle-revenue').addEventListener('click', function() {
      state.monthlyVisibility.revenue = !state.monthlyVisibility.revenue
      renderMonthly()
    })
  }

  function renderMonthlySVG(entries) {
    var svgContainer = $('#monthly-svg-container')
    if (!svgContainer) return

    var width = svgContainer.clientWidth || 800
    var height = 350
    var padding = { top: 30, right: 65, bottom: 40, left: 55 }
    var chartWidth = width - padding.left - padding.right
    var chartHeight = height - padding.top - padding.bottom

    var xStep = chartWidth / entries.length

    var maxLeft = 1
    var maxRight = 1
    for (var i = 0; i < entries.length; i++) {
      var e = entries[i]
      if (state.monthlyVisibility.waffles && e.waffles > maxLeft) maxLeft = e.waffles
      if (state.monthlyVisibility.claims && e.spots_claimed > maxLeft) maxLeft = e.spots_claimed
      var revD = e.revenue / 100
      if (state.monthlyVisibility.revenue && revD > maxRight) maxRight = revD
    }

    function roundMax(val) {
      if (val <= 5) return 5
      if (val <= 10) return 10
      if (val <= 50) return Math.ceil(val / 10) * 10
      if (val <= 100) return Math.ceil(val / 20) * 20
      return Math.ceil(val / 50) * 50
    }
    maxLeft = roundMax(maxLeft)
    maxRight = roundMax(maxRight)

    function getX(idx) {
      return padding.left + idx * xStep + xStep / 2
    }
    var getLeftY = function(val) {
      return padding.top + chartHeight - (val / maxLeft) * chartHeight
    }
    var getRightY = function(val) {
      return padding.top + chartHeight - (val / maxRight) * chartHeight
    }

    var gridLinesHtml = ''
    for (var i = 0; i <= 4; i++) {
      var y = padding.top + (chartHeight / 4) * i
      var leftVal = Math.round(maxLeft - (maxLeft / 4) * i)
      var rightVal = Math.round(maxRight - (maxRight / 4) * i)
      gridLinesHtml += '<line x1="' + padding.left + '" y1="' + y + '" x2="' + (width - padding.right) + '" y2="' + y + '" stroke="currentColor" class="opacity-10" stroke-dasharray="4 4" />'
      
      if (state.monthlyVisibility.waffles || state.monthlyVisibility.claims) {
        gridLinesHtml += '<text x="' + (padding.left - 10) + '" y="' + (y + 4) + '" fill="currentColor" class="text-[10px] opacity-40 font-medium" text-anchor="end">' + leftVal + '</text>'
      }
      if (state.monthlyVisibility.revenue) {
        gridLinesHtml += '<text x="' + (width - padding.right + 10) + '" y="' + (y + 4) + '" fill="currentColor" class="text-[10px] opacity-40 font-medium" text-anchor="start">$' + rightVal + '</text>'
      }
    }

    var xLabelsHtml = ''
    var monthNames = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]
    for (var i = 0; i < entries.length; i++) {
      var e = entries[i]
      var x = getX(i)
      var parts = e.month.split('-')
      var label = e.month
      var monthIdx = parseInt(parts[1], 10) - 1
      if (monthIdx >= 0 && monthIdx < 12) {
        label = monthNames[monthIdx] + ' ' + parts[0].substring(2)
      }
      xLabelsHtml += '<text x="' + x + '" y="' + (height - padding.bottom + 20) + '" fill="currentColor" class="text-[10px] opacity-50" text-anchor="middle">' + label + '</text>'
    }

    var barsHtml = ''
    var showW = state.monthlyVisibility.waffles
    var showC = state.monthlyVisibility.claims
    for (var i = 0; i < entries.length; i++) {
      var e = entries[i]
      var xCenter = getX(i)

      if (showW && showC) {
        var barW = Math.max(xStep * 0.22, 4)
        var gap = Math.max(xStep * 0.04, 1)
        
        var wH = chartHeight - (getLeftY(e.waffles) - padding.top)
        var wY = getLeftY(e.waffles)
        var wX = xCenter - barW - gap/2
        barsHtml += '<rect x="' + wX + '" y="' + wY + '" width="' + barW + '" height="' + wH + '" fill="url(#waffles-grad)" rx="2" class="opacity-80 transition-all duration-200" />'

        var cH = chartHeight - (getLeftY(e.spots_claimed) - padding.top)
        var cY = getLeftY(e.spots_claimed)
        var cX = xCenter + gap/2
        barsHtml += '<rect x="' + cX + '" y="' + cY + '" width="' + barW + '" height="' + cH + '" fill="url(#claims-grad)" rx="2" class="opacity-80 transition-all duration-200" />'
      } else if (showW) {
        var barW = Math.max(xStep * 0.35, 6)
        var wH = chartHeight - (getLeftY(e.waffles) - padding.top)
        var wY = getLeftY(e.waffles)
        var wX = xCenter - barW/2
        barsHtml += '<rect x="' + wX + '" y="' + wY + '" width="' + barW + '" height="' + wH + '" fill="url(#waffles-grad)" rx="3" class="opacity-80 transition-all duration-200" />'
      } else if (showC) {
        var barW = Math.max(xStep * 0.35, 6)
        var cH = chartHeight - (getLeftY(e.spots_claimed) - padding.top)
        var cY = getLeftY(e.spots_claimed)
        var cX = xCenter - barW/2
        barsHtml += '<rect x="' + cX + '" y="' + cY + '" width="' + barW + '" height="' + cH + '" fill="url(#claims-grad)" rx="3" class="opacity-80 transition-all duration-200" />'
      }
    }

    var lineHtml = ''
    if (state.monthlyVisibility.revenue) {
      var points = []
      for (var i = 0; i < entries.length; i++) {
        points.push({ x: getX(i), y: getRightY(entries[i].revenue / 100) })
      }

      if (points.length > 0) {
        var lineD = makeBezierPath(points)
        var areaD = makeBezierAreaPath(points, height - padding.bottom)

        lineHtml += '<path d="' + areaD + '" fill="url(#revenue-area-grad)" />'
        lineHtml += '<path d="' + lineD + '" fill="none" stroke="var(--color-secondary)" stroke-width="2.5" stroke-linecap="round" />'
        
        for (var i = 0; i < points.length; i++) {
          lineHtml += '<circle cx="' + points[i].x + '" cy="' + points[i].y + '" r="3" fill="var(--color-secondary)" stroke="var(--color-base-100)" stroke-width="1" />'
        }
      }
    }

    var hoverZonesHtml = ''
    for (var i = 0; i < entries.length; i++) {
      var xStart = padding.left + i * xStep
      hoverZonesHtml += '<rect x="' + xStart + '" y="' + padding.top + '" width="' + xStep + '" height="' + chartHeight + '" fill="transparent" class="cursor-pointer chart-hover-zone" data-idx="' + i + '" />'
    }

    var svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg')
    svg.setAttribute('width', '100%')
    svg.setAttribute('height', '100%')
    svg.setAttribute('viewBox', '0 0 ' + width + ' ' + height)
    svg.setAttribute('preserveAspectRatio', 'xMidYMid meet')

    svg.innerHTML = `
      <defs>
        <linearGradient id="waffles-grad" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="var(--color-primary)" stop-opacity="0.9" />
          <stop offset="100%" stop-color="var(--color-primary)" stop-opacity="0.2" />
        </linearGradient>
        <linearGradient id="claims-grad" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="var(--color-success)" stop-opacity="0.9" />
          <stop offset="100%" stop-color="var(--color-success)" stop-opacity="0.2" />
        </linearGradient>
        <linearGradient id="revenue-area-grad" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="var(--color-secondary)" stop-opacity="0.25" />
          <stop offset="100%" stop-color="var(--color-secondary)" stop-opacity="0.0" />
        </linearGradient>
      </defs>
      <g>${gridLinesHtml}</g>
      <line x1="${padding.left}" y1="${height - padding.bottom}" x2="${width - padding.right}" y2="${height - padding.bottom}" stroke="currentColor" class="opacity-15" stroke-width="1.5" />
      <g>${barsHtml}</g>
      <g>${lineHtml}</g>
      <g>${xLabelsHtml}</g>
      <line id="hover-line" x1="0" y1="${padding.top}" x2="0" y2="${height - padding.bottom}" stroke="currentColor" class="opacity-20 hidden" stroke-width="1" stroke-dasharray="3 3" />
      <circle id="hover-dot-waffles" r="5" class="fill-primary stroke-base-100 stroke-2 hidden" />
      <circle id="hover-dot-claims" r="5" class="fill-success stroke-base-100 stroke-2 hidden" />
      <circle id="hover-dot-revenue" r="5" class="fill-secondary stroke-base-100 stroke-2 hidden" />
      <g>${hoverZonesHtml}</g>
    `

    svgContainer.innerHTML = ''
    svgContainer.appendChild(svg)

    var tooltip = $('#chart-tooltip')
    var hoverLine = svgContainer.querySelector('#hover-line')
    var dotW = svgContainer.querySelector('#hover-dot-waffles')
    var dotC = svgContainer.querySelector('#hover-dot-claims')
    var dotR = svgContainer.querySelector('#hover-dot-revenue')

    var zones = svgContainer.querySelectorAll('.chart-hover-zone')
    zones.forEach(function (zone) {
      zone.addEventListener('mouseenter', function () {
        var idx = parseInt(this.getAttribute('data-idx'), 10)
        var entry = entries[idx]
        if (!entry) return

        var parts = entry.month.split('-')
        var fullMonthNames = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"]
        var monthLabel = fullMonthNames[parseInt(parts[1], 10) - 1] + ' ' + parts[0]

        var tHtml = '<div class="font-bold border-b border-base-content/10 pb-1 mb-1.5 text-base-content">' + monthLabel + '</div>'
        if (state.monthlyVisibility.waffles) {
          tHtml += '<div class="flex items-center justify-between gap-4"><span class="flex items-center gap-1.5"><span class="w-2.5 h-2.5 rounded-full bg-primary"></span>Waffles</span><span class="font-bold text-primary">' + entry.waffles + '</span></div>'
        }
        if (state.monthlyVisibility.claims) {
          tHtml += '<div class="flex items-center justify-between gap-4"><span class="flex items-center gap-1.5"><span class="w-2.5 h-2.5 rounded-full bg-success"></span>Claims</span><span class="font-bold text-success">' + entry.spots_claimed + '</span></div>'
        }
        if (state.monthlyVisibility.revenue) {
          tHtml += '<div class="flex items-center justify-between gap-4"><span class="flex items-center gap-1.5"><span class="w-2.5 h-2.5 rounded-full bg-secondary"></span>Revenue</span><span class="font-bold text-secondary">' + formatMoney(entry.revenue) + '</span></div>'
        }

        tooltip.innerHTML = tHtml
        tooltip.classList.remove('hidden')

        var x = getX(idx)
        hoverLine.setAttribute('x1', x)
        hoverLine.setAttribute('x2', x)
        hoverLine.classList.remove('hidden')

        if (state.monthlyVisibility.waffles) {
          dotW.setAttribute('cx', x)
          dotW.setAttribute('cy', getLeftY(entry.waffles))
          dotW.classList.remove('hidden')
        } else {
          dotW.classList.add('hidden')
        }

        if (state.monthlyVisibility.claims) {
          dotC.setAttribute('cx', x)
          dotC.setAttribute('cy', getLeftY(entry.spots_claimed))
          dotC.classList.remove('hidden')
        } else {
          dotC.classList.add('hidden')
        }

        if (state.monthlyVisibility.revenue) {
          dotR.setAttribute('cx', x)
          dotR.setAttribute('cy', getRightY(entry.revenue / 100))
          dotR.classList.remove('hidden')
        } else {
          dotR.classList.add('hidden')
        }
      })

      zone.addEventListener('mousemove', function (e) {
        var rect = svgContainer.getBoundingClientRect()
        var x = e.clientX - rect.left
        var y = e.clientY - rect.top

        var tooltipWidth = tooltip.clientWidth || 120
        var tooltipHeight = tooltip.clientHeight || 80
        
        var leftPos = x + 15
        if (leftPos + tooltipWidth > rect.width) {
          leftPos = x - tooltipWidth - 15
        }
        
        var topPos = y - tooltipHeight / 2
        if (topPos < 0) {
          topPos = 10
        } else if (topPos + tooltipHeight > rect.height) {
          topPos = rect.height - tooltipHeight - 10
        }

        tooltip.style.left = leftPos + 'px'
        tooltip.style.top = topPos + 'px'
      })

      zone.addEventListener('mouseleave', function () {
        tooltip.classList.add('hidden')
        hoverLine.classList.add('hidden')
        dotW.classList.add('hidden')
        dotC.classList.add('hidden')
        dotR.classList.add('hidden')
      })
    })
  }

  function getControlPoints(p0, p1, p2, p3, t) {
    t = t || 0.15
    var cp1x = p1.x + (p2.x - p0.x) * t
    var cp1y = p1.y + (p2.y - p0.y) * t
    var cp2x = p2.x - (p3.x - p1.x) * t
    var cp2y = p2.y - (p3.y - p1.y) * t
    return { cp1x: cp1x, cp1y: cp1y, cp2x: cp2x, cp2y: cp2y }
  }

  function makeBezierPath(points) {
    if (points.length === 0) return ''
    if (points.length === 1) return 'M ' + points[0].x + ' ' + points[0].y
    if (points.length === 2) {
      return 'M ' + points[0].x + ' ' + points[0].y + ' L ' + points[1].x + ' ' + points[1].y
    }
    var d = 'M ' + points[0].x + ' ' + points[0].y
    for (var i = 0; i < points.length - 1; i++) {
      var p0 = points[i - 1] || points[i]
      var p1 = points[i]
      var p2 = points[i + 1]
      var p3 = points[i + 2] || p2
      var cp = getControlPoints(p0, p1, p2, p3)
      d += ' C ' + cp.cp1x + ' ' + cp.cp1y + ' ' + cp.cp2x + ' ' + cp.cp2y + ' ' + p2.x + ' ' + p2.y
    }
    return d
  }

  function makeBezierAreaPath(points, bottomY) {
    if (points.length === 0) return ''
    var d = 'M ' + points[0].x + ' ' + bottomY + ' L ' + points[0].x + ' ' + points[0].y
    if (points.length === 1) {
      d += ' L ' + points[0].x + ' ' + bottomY + ' Z'
      return d
    }
    if (points.length === 2) {
      d += ' L ' + points[1].x + ' ' + points[1].y
    } else {
      for (var i = 0; i < points.length - 1; i++) {
        var p0 = points[i - 1] || points[i]
        var p1 = points[i]
        var p2 = points[i + 1]
        var p3 = points[i + 2] || p2
        var cp = getControlPoints(p0, p1, p2, p3)
        d += ' C ' + cp.cp1x + ' ' + cp.cp1y + ' ' + cp.cp2x + ' ' + cp.cp2y + ' ' + p2.x + ' ' + p2.y
      }
    }
    d += ' L ' + points[points.length - 1].x + ' ' + bottomY + ' Z'
    return d
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
      html += '<div class="grid grid-cols-1 md:grid-cols-2 gap-6 p-6">'
      
      var activeEntry = null
      var completedEntry = null
      for (var i = 0; i < entries.length; i++) {
        if (entries[i].status === 'active') activeEntry = entries[i]
        if (entries[i].status === 'completed') completedEntry = entries[i]
      }
      
      if (activeEntry && (state.velocityFilter === '' || state.velocityFilter === 'active')) {
        html += '<div class="card bg-base-300 border border-base-content/5 p-6 rounded-xl shadow-md flex flex-col justify-between">' +
          '<div>' +
          '<div class="flex items-center justify-between mb-4">' +
          '<span class="text-xs uppercase font-bold tracking-wider text-success">Active Waffles</span>' +
          '<span class="badge bg-success/20 text-success border-success/30 font-medium">' + activeEntry.waffle_count + ' Waffles</span>' +
          '</div>' +
          '<p class="text-2xl font-black text-base-content mb-6">' + activeEntry.avg_first_claim_hours.toFixed(1) + 'h <span class="text-xs font-normal opacity-50">Avg to First Claim</span></p>' +
          
          '<!-- SVG Timeline -->' +
          '<div class="relative w-full h-12 flex items-center justify-between px-2 mb-6">' +
          '<div class="absolute left-6 right-6 h-1 bg-base-100 rounded-full"></div>' +
          '<div class="absolute left-6 w-1/2 h-1 bg-gradient-to-r from-primary to-primary/40 rounded-full"></div>' +
          
          '<!-- Created Node -->' +
          '<div class="relative z-10 flex flex-col items-center">' +
          '<div class="w-6 h-6 rounded-full bg-base-100 border-2 border-primary flex items-center justify-center">' +
          '<div class="w-2.5 h-2.5 rounded-full bg-primary"></div>' +
          '</div>' +
          '<span class="text-[10px] mt-1 opacity-60">Created</span>' +
          '</div>' +
          
          '<!-- First Claim Node -->' +
          '<div class="relative z-10 flex flex-col items-center">' +
          '<div class="w-6 h-6 rounded-full bg-base-100 border-2 border-primary flex items-center justify-center">' +
          '<div class="w-2.5 h-2.5 rounded-full bg-primary"></div>' +
          '</div>' +
          '<span class="text-[10px] mt-1 font-semibold text-primary">' + activeEntry.avg_first_claim_hours.toFixed(1) + 'h</span>' +
          '<span class="text-[9px] opacity-40">First Claim</span>' +
          '</div>' +
          
          '<!-- Filling Node -->' +
          '<div class="relative z-10 flex flex-col items-center">' +
          '<div class="w-6 h-6 rounded-full bg-base-100 border-2 border-dashed border-success flex items-center justify-center animate-pulse">' +
          '<div class="w-2.5 h-2.5 rounded-full bg-success"></div>' +
          '</div>' +
          '<span class="text-[10px] mt-1 text-success font-medium">Filling</span>' +
          '</div>' +
          '</div>' +
          '</div>' +
          '<div class="text-xs opacity-50 mt-2">Measures response time: the lag between creating a board and a buyer claiming spot #1.</div>' +
          '</div>'
      }
      
      if (completedEntry && (state.velocityFilter === '' || state.velocityFilter === 'completed')) {
        var firstClaimVal = completedEntry.avg_first_claim_hours
        var compVal = completedEntry.avg_completion_hours
        var firstClaimPercent = compVal > 0 ? (firstClaimVal / compVal * 100) : 0
        
        html += '<div class="card bg-base-300 border border-base-content/5 p-6 rounded-xl shadow-md flex flex-col justify-between">' +
          '<div>' +
          '<div class="flex items-center justify-between mb-4">' +
          '<span class="text-xs uppercase font-bold tracking-wider text-secondary">Completed Waffles</span>' +
          '<span class="badge bg-secondary/20 text-secondary border-secondary/30 font-medium">' + completedEntry.waffle_count + ' Waffles</span>' +
          '</div>' +
          
          '<div class="grid grid-cols-2 gap-4 mb-6">' +
          '<div>' +
          '<p class="text-xs opacity-50">Avg to First Claim</p>' +
          '<p class="text-xl font-bold text-base-content">' + firstClaimVal.toFixed(1) + 'h</p>' +
          '</div>' +
          '<div>' +
          '<p class="text-xs opacity-50">Avg to Complete</p>' +
          '<p class="text-xl font-bold text-secondary">' + compVal.toFixed(1) + 'h</p>' +
          '</div>' +
          '</div>' +
          
          '<!-- SVG Timeline -->' +
          '<div class="relative w-full h-12 flex items-center justify-between px-2 mb-6">' +
          '<div class="absolute left-6 right-6 h-1 bg-base-100 rounded-full"></div>' +
          '<div class="absolute left-6 h-1 bg-primary rounded-full" style="width: ' + Math.min(firstClaimPercent, 90) + '%"></div>' +
          '<div class="absolute h-1 bg-gradient-to-r from-primary to-secondary rounded-full" style="left: calc(1.5rem + ' + Math.min(firstClaimPercent, 90) + '%); right: 1.5rem"></div>' +
          
          '<!-- Created Node -->' +
          '<div class="relative z-10 flex flex-col items-center">' +
          '<div class="w-6 h-6 rounded-full bg-base-100 border-2 border-primary flex items-center justify-center">' +
          '<div class="w-2.5 h-2.5 rounded-full bg-primary"></div>' +
          '</div>' +
          '<span class="text-[10px] mt-1 opacity-60">0h</span>' +
          '</div>' +
          
          '<!-- First Claim Node -->' +
          '<div class="relative z-10 flex flex-col items-center">' +
          '<div class="w-6 h-6 rounded-full bg-base-100 border-2 border-primary flex items-center justify-center">' +
          '<div class="w-2.5 h-2.5 rounded-full bg-primary"></div>' +
          '</div>' +
          '<span class="text-[10px] mt-1 font-semibold text-primary">' + firstClaimVal.toFixed(1) + 'h</span>' +
          '<span class="text-[9px] opacity-40">First Claim</span>' +
          '</div>' +
          
          '<!-- Completed Node -->' +
          '<div class="relative z-10 flex flex-col items-center">' +
          '<div class="w-6 h-6 rounded-full bg-base-100 border-2 border-secondary flex items-center justify-center">' +
          '<div class="w-2.5 h-2.5 rounded-full bg-secondary"></div>' +
          '</div>' +
          '<span class="text-[10px] mt-1 font-semibold text-secondary">' + compVal.toFixed(1) + 'h</span>' +
          '<span class="text-[9px] opacity-40">Completed</span>' +
          '</div>' +
          '</div>' +
          '</div>' +
          '<div class="text-xs opacity-50 mt-2">Measures total cycle time: the total duration from board creation until all spots are paid and winner is marked.</div>' +
          '</div>'
      }

      html += '</div>'

      html += '<div class="px-6 pb-6"><div class="overflow-x-auto border border-base-content/5 rounded-xl bg-base-300/40"><table class="w-full text-sm">' +
        '<thead class="bg-base-300"><tr>' +
        '<th class="text-left px-4 py-3 font-medium text-base-content/60">Status</th>' +
        '<th class="text-right px-4 py-3 font-medium text-base-content/60">Waffles</th>' +
        '<th class="text-right px-4 py-3 font-medium text-base-content/60">Avg First Claim</th>' +
        '<th class="text-right px-4 py-3 font-medium text-base-content/60">Avg Completion</th>' +
        '</tr></thead><tbody>'

      for (var i = 0; i < entries.length; i++) {
        var v = entries[i]
        var statusBadge = v.status === 'active'
          ? '<span class="badge bg-success/20 text-success border-success/30 font-medium">' + escHtml(v.status) + '</span>'
          : '<span class="badge bg-secondary/20 text-secondary border-secondary/30 font-medium">' + escHtml(v.status) + '</span>'
        var completion = v.avg_completion_hours > 0 ? v.avg_completion_hours.toFixed(1) + 'h' : '\u2014'

        html += '<tr class="border-t border-base-content/5 hover:bg-base-300/30">' +
          '<td class="px-4 py-3">' + statusBadge + '</td>' +
          '<td class="px-4 py-3 text-right text-base-content/75">' + v.waffle_count + '</td>' +
          '<td class="px-4 py-3 text-right text-base-content/75">' + v.avg_first_claim_hours.toFixed(1) + 'h</td>' +
          '<td class="px-4 py-3 text-right text-base-content/75">' + completion + '</td>' +
          '</tr>'
      }

      html += '</tbody></table></div></div>'
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

    window.addEventListener('resize', function() {
      if (state.activeTab === 'monthly' && state.monthlyData) {
        renderMonthly()
      }
    })
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init)
  } else {
    init()
  }
})()
