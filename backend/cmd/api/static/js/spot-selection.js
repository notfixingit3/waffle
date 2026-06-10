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

    initRandomClaim();

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
    var selectedClasses = SPOT_SELECTION_CLASSES.selected.split(' ');
    if (selectedSpots.has(num)) {
      selectedSpots.delete(num);
      selectedClasses.forEach(function(cls) { el.classList.remove(cls); });
      el.setAttribute('aria-checked', 'false');
    } else {
      selectedSpots.add(num);
      selectedClasses.forEach(function(cls) { el.classList.add(cls); });
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
          var pendingClasses = SPOT_STATUS_CLASSES.pending;
          el.className = 'spot-item relative rounded-lg border-2 text-center p-2 min-h-[44px] flex flex-col items-center justify-center transition-all duration-200 touch-manipulation select-none ' + pendingClasses.bg + ' ' + pendingClasses.border + ' ' + pendingClasses.text + ' cursor-default';
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

  function initRandomClaim() {
    var randomBtn = document.getElementById('claim-random-btn');
    var countInput = document.getElementById('random-spot-count');
    var availableCount;

    if (randomBtn) {
      randomBtn.addEventListener('click', submitRandomClaim);
    }

    if (countInput) {
      countInput.addEventListener('input', function() {
        availableCount = document.querySelectorAll('[data-spot-status="available"]').length;
        updateRandomClaimButtonState(availableCount);
      });
    }

    availableCount = document.querySelectorAll('[data-spot-status="available"]').length;
    updateRandomClaimButtonState(availableCount);
  }

  function submitRandomClaim() {
    var btn = document.getElementById('claim-random-btn');
    var errorEl = document.getElementById('random-claim-error');
    var successEl = document.getElementById('random-claim-success');
    var countInput = document.getElementById('random-spot-count');
    var handleInput = document.getElementById('instagram-handle');
    var handle;
    var count;
    var availableCount;

    if (errorEl) errorEl.classList.add('hidden');
    if (successEl) successEl.classList.add('hidden');

    handle = handleInput.value.trim().replace(/^@/, '');

    if (!handle) {
      showRandomClaimError('Please enter your Instagram handle');
      return;
    }

    if (!/^[a-zA-Z0-9_.]{1,30}$/.test(handle)) {
      showRandomClaimError('Invalid Instagram handle. Use letters, numbers, underscores, and periods only (max 30 chars)');
      return;
    }

    count = parseInt(countInput.value, 10);
    if (isNaN(count) || count <= 0) {
      showRandomClaimError('Please enter a valid number of spots');
      return;
    }

    availableCount = document.querySelectorAll('[data-spot-status="available"]').length;
    if (availableCount === 0) {
      showRandomClaimError('No spots available');
      return;
    }

    btn.disabled = true;
    btn.textContent = 'Claiming...';

    fetch('/api/claims/random', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        waffle_id: config.waffleId,
        count: count,
        instagram_handle: handle
      })
    })
    .then(function(res) {
      return res.json().then(function(data) {
        return { ok: res.ok, data: data };
      });
    })
    .then(function(result) {
      var spotNumbers;
      var newAvailableCount;

      if (!result.ok) {
        showRandomClaimError(result.data.error || 'Random claim failed');
        btn.disabled = false;
        btn.textContent = 'Claim Random Spots';
        availableCount = document.querySelectorAll('[data-spot-status="available"]').length;
        updateRandomClaimButtonState(availableCount);
        return;
      }

      spotNumbers = result.data.spot_numbers || [];
      spotNumbers.forEach(function(num) {
        var el = document.querySelector('[data-spot-number="' + num + '"]');
        var pendingClasses;
        var handleSpan;
        var existingHandle;
        if (el) {
          el.dataset.spotStatus = 'pending';
          pendingClasses = SPOT_STATUS_CLASSES.pending;
          el.className = 'spot-item relative rounded-lg border-2 text-center p-2 min-h-[44px] flex flex-col items-center justify-center transition-all duration-200 touch-manipulation select-none ' + pendingClasses.bg + ' ' + pendingClasses.border + ' ' + pendingClasses.text + ' cursor-default';
          el.disabled = true;
          el.setAttribute('aria-checked', 'false');

          existingHandle = el.querySelector('span');
          if (existingHandle) existingHandle.remove();

          handleSpan = document.createElement('span');
          handleSpan.className = 'text-[10px] truncate w-full mt-0.5 opacity-70';
          handleSpan.textContent = handle;
          el.appendChild(handleSpan);
        }
      });

      newAvailableCount = document.querySelectorAll('[data-spot-status="available"]').length;
      updateRandomClaimButtonState(newAvailableCount);

      showRandomClaimSuccess(spotNumbers, result.data.requested_count, result.data.claimed_count);

      if (countInput) countInput.value = '';

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
    })
    .catch(function(err) {
      showRandomClaimError('Network error. Please try again.');
      btn.disabled = false;
      btn.textContent = 'Claim Random Spots';
      availableCount = document.querySelectorAll('[data-spot-status="available"]').length;
      updateRandomClaimButtonState(availableCount);
    });
  }

  function updateRandomClaimButtonState(availableCount) {
    var btn = document.getElementById('claim-random-btn');
    var countInput = document.getElementById('random-spot-count');
    var count;
    var isValid;

    if (!btn) return;

    count = countInput ? parseInt(countInput.value, 10) : 0;
    isValid = !isNaN(count) && count > 0 && availableCount > 0;

    btn.disabled = !isValid;

    if (availableCount === 0) {
      btn.textContent = 'No Spots Available';
    } else {
      btn.textContent = 'Claim Random Spots';
    }
  }

  function showRandomClaimSuccess(spotNumbers, requestedCount, claimedCount) {
    var successEl = document.getElementById('random-claim-success');
    var msg;

    if (!successEl) return;

    msg = 'Claimed ' + claimedCount + ' spot' + (claimedCount !== 1 ? 's' : '');
    if (claimedCount < requestedCount) {
      msg += ' (requested ' + requestedCount + ', only ' + claimedCount + ' available)';
    }
    msg += ': #' + spotNumbers.join(', #');

    successEl.textContent = msg;
    successEl.classList.remove('hidden');

    setTimeout(function() {
      successEl.classList.add('hidden');
    }, 5000);
  }

  function showRandomClaimError(message) {
    var errorEl = document.getElementById('random-claim-error');
    if (errorEl) {
      errorEl.textContent = message;
      errorEl.classList.remove('hidden');
    }
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
