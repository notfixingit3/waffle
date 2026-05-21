var SpotSelection = (function() {
  'use strict';

  var selectedSpots = new Set();
  var config = {};
  var _cacheKey = null;

  function init(opts) {
    config = opts || {};
    _cacheKey = config.slug || null;

    var grid = document.getElementById('spot-grid');
    if (!grid) return;

    grid.addEventListener('click', function(e) {
      var spot = e.target.closest('.spot-item');
      if (!spot) return;
      var status = spot.dataset.spotStatus;
      if (status !== 'available') return;

      var num = parseInt(spot.dataset.spotNumber, 10);
      if (isNaN(num)) return;

      toggleSpot(num, spot);
    });

    var handleInput = document.getElementById('instagram-handle');
    if (handleInput) {
      handleInput.addEventListener('input', function() {
        var clean = handleInput.value.replace(/^@/, '');
        if (clean !== handleInput.value) {
          handleInput.value = clean;
        }
      });
    }

    var claimBtn = document.getElementById('claim-btn');
    if (claimBtn) {
      claimBtn.addEventListener('click', submitClaim);
    }

    updateClaimButton();
    if (!navigator.onLine) {
      setOfflineClaim();
    }
    window.addEventListener('waffle:offline', function() { setOfflineClaim(); });
    window.addEventListener('waffle:online', function() { updateClaimButton(); });
  }

  function setOfflineClaim() {
    var btn = document.getElementById('claim-btn');
    if (!btn) return;
    btn.disabled = true;
    btn.textContent = 'Offline — cannot claim';
  }

  function toggleSpot(num, el) {
    if (selectedSpots.has(num)) {
      selectedSpots.delete(num);
      el.classList.remove('ring-2', 'ring-blue-500', 'bg-green-200');
      el.setAttribute('aria-checked', 'false');
    } else {
      selectedSpots.add(num);
      el.classList.add('ring-2', 'ring-blue-500', 'bg-green-200');
      el.setAttribute('aria-checked', 'true');
    }

    updateClaimButton();
  }

  function updateClaimButton() {
    var btn = document.getElementById('claim-btn');
    var summary = document.getElementById('claim-summary');
    var countEl = document.getElementById('claim-count');
    var totalEl = document.getElementById('claim-total');

    var count = selectedSpots.size;
    var total = count * (config.spotPrice || 0);

    if (count > 0) {
      btn.disabled = false;
      btn.textContent = 'Claim ' + count + ' Spot' + (count !== 1 ? 's' : '') + ' - $' + total;
      summary.classList.remove('hidden');
      countEl.textContent = count;
      totalEl.textContent = '$' + total;
    } else {
      btn.disabled = true;
      btn.textContent = 'Select Spots to Claim';
      summary.classList.add('hidden');
    }
  }

  function submitClaim() {
    var btn = document.getElementById('claim-btn');
    var errorEl = document.getElementById('claim-error');
    var successEl = document.getElementById('claim-success');
    var handleInput = document.getElementById('instagram-handle');

    errorEl.classList.add('hidden');
    successEl.classList.add('hidden');

    var handle = handleInput.value.trim().replace(/^@/, '');

    if (!handle) {
      showError('Please enter your Instagram handle');
      return;
    }

    if (!/^[a-zA-Z0-9_.]{1,30}$/.test(handle)) {
      showError('Invalid Instagram handle. Use letters, numbers, underscores, and periods only (max 30 chars)');
      return;
    }

    if (selectedSpots.size === 0) {
      showError('Please select at least one spot');
      return;
    }

    var spotsArray = Array.from(selectedSpots).sort(function(a, b) { return a - b; });

    btn.disabled = true;
    btn.textContent = 'Claiming...';

    fetch('/api/claims', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        waffle_id: config.waffleId,
        spots: spotsArray,
        instagram_handle: handle
      })
    })
    .then(function(res) {
      return res.json().then(function(data) {
        return { ok: res.ok, data: data };
      });
    })
    .then(function(result) {
      if (!result.ok) {
        showError(result.data.error || 'Claim failed');
        btn.disabled = false;
        updateClaimButton();
        return;
      }

      selectedSpots.forEach(function(num) {
        var el = document.querySelector('[data-spot-number="' + num + '"]');
        if (el) {
          el.dataset.spotStatus = 'pending';
          el.className = el.className.replace(/ring-\w+-\d+/g, '').replace(/bg-green-\w+/g, 'bg-yellow-50');
          el.className = el.className.replace(/border-green-\w+/g, 'border-yellow-400');
          el.className = el.className.replace(/cursor-pointer/g, 'cursor-default');
          el.classList.add('text-yellow-900');
          el.disabled = true;
          el.setAttribute('aria-checked', 'false');

          var handleSpan = document.createElement('span');
          handleSpan.className = 'text-[10px] truncate w-full mt-0.5 opacity-70';
          handleSpan.textContent = handle;
          el.appendChild(handleSpan);
        }
      });

      selectedSpots.clear();

      if (_cacheKey && typeof OfflineHandler !== 'undefined') {
        fetch('/api/waffles/' + _cacheKey + '/spots')
          .then(function(res) { return res.json(); })
          .then(function(data) {
            if (data && data.spots) {
              OfflineHandler.setCachedData(_cacheKey, data.spots);
            }
          })
          .catch(function() { /* cache update failed — non-critical */ });
      }

      successEl.textContent = spotsArray.length + ' spot' + (spotsArray.length !== 1 ? 's' : '') + ' claimed by @' + handle + '!';
      successEl.classList.remove('hidden');
      handleInput.value = '';
      updateClaimButton();

      setTimeout(function() {
        successEl.classList.add('hidden');
      }, 5000);
    })
    .catch(function(err) {
      showError('Network error. Please try again.');
      btn.disabled = false;
      updateClaimButton();
    });
  }

  function showError(msg) {
    var errorEl = document.getElementById('claim-error');
    if (errorEl) {
      errorEl.textContent = msg;
      errorEl.classList.remove('hidden');
    }
  }

  return {
    init: init
  };
})();
