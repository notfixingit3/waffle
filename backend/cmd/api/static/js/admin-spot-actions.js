var AdminSpotActions = (function() {
  'use strict';

  var bulkMode = false;
  var selectedForBulk = new Set();
  var config = {};

  function init(opts) {
    config = opts || {};

    var grid = document.getElementById('spot-grid');
    if (grid) {
      grid.addEventListener('click', handleGridClick);
    }

    var pendingList = document.getElementById('pending-list');
    if (pendingList) {
      pendingList.addEventListener('click', handlePendingListClick);
    }

    var bulkToggle = document.getElementById('bulk-mode-toggle');
    if (bulkToggle) {
      bulkToggle.addEventListener('click', toggleBulkMode);
    }

    var bulkPayBtn = document.getElementById('bulk-pay-btn');
    if (bulkPayBtn) {
      bulkPayBtn.addEventListener('click', executeBulkPay);
    }

    var setWinnerBtn = document.getElementById('set-winner-btn');
    if (setWinnerBtn) {
      setWinnerBtn.addEventListener('click', executeSetWinner);
    }
  }

  function handleGridClick(e) {
    var spot = e.target.closest('.admin-spot-item');
    if (!spot) return;

    var status = spot.dataset.spotStatus;

    if (bulkMode && status === 'pending') {
      toggleBulkSelect(spot);
      return;
    }

    if (status === 'pending') {
      markSpotPaid(spot.dataset.spotId, spot.dataset.spotNumber, spot, false);
    }
  }

  function handlePendingListClick(e) {
    var payBtn = e.target.closest('.pay-single-btn');
    if (payBtn) {
      var spotId = payBtn.dataset.spotId;
      var spotNumber = payBtn.dataset.spotNumber;
      var listItem = payBtn.closest('.pending-claim-item');
      markSpotPaid(spotId, spotNumber, listItem, true);
      return;
    }

    var releaseBtn = e.target.closest('.release-btn');
    if (releaseBtn) {
      var spotId = releaseBtn.dataset.spotId;
      var spotNumber = releaseBtn.dataset.spotNumber;
      var listItem = releaseBtn.closest('.pending-claim-item');
      releaseSpot(spotId, spotNumber, listItem);
    }
  }

  function toggleBulkMode() {
    bulkMode = !bulkMode;
    var btn = document.getElementById('bulk-mode-toggle');

    if (bulkMode) {
      btn.textContent = 'Exit Bulk Mode';
      btn.classList.remove('btn-outline');
      btn.classList.add('btn-warning');
      document.getElementById('bulk-actions').classList.remove('hidden');
    } else {
      btn.textContent = 'Bulk Pay Mode';
      btn.classList.remove('btn-warning');
      btn.classList.add('btn-outline');
      document.getElementById('bulk-actions').classList.add('hidden');
      clearBulkSelection();
    }
  }

  function toggleBulkSelect(spot) {
    var spotId = spot.dataset.spotId;
    var bulkClasses = SPOT_SELECTION_CLASSES.bulk_selected.split(' ');

    if (selectedForBulk.has(spotId)) {
      selectedForBulk.delete(spotId);
      bulkClasses.forEach(function(cls) { spot.classList.remove(cls); });
    } else {
      selectedForBulk.add(spotId);
      bulkClasses.forEach(function(cls) { spot.classList.add(cls); });
    }

    updateBulkUI();
  }

  function clearBulkSelection() {
    var bulkClasses = SPOT_SELECTION_CLASSES.bulk_selected.split(' ');
    selectedForBulk.forEach(function(spotId) {
      var el = document.querySelector('[data-spot-id="' + spotId + '"]');
      if (el) {
        bulkClasses.forEach(function(cls) { el.classList.remove(cls); });
      }
    });
    selectedForBulk.clear();
    updateBulkUI();
  }

  function updateBulkUI() {
    var count = selectedForBulk.size;
    document.getElementById('bulk-count').textContent = count;

    var btn = document.getElementById('bulk-pay-btn');
    if (count > 0) {
      btn.disabled = false;
      btn.textContent = 'Mark ' + count + ' Paid';
    } else {
      btn.disabled = true;
      btn.textContent = 'Mark Paid';
    }
  }

  function executeBulkPay() {
    if (selectedForBulk.size === 0) return;

    var spots = Array.from(selectedForBulk);
    var btn = document.getElementById('bulk-pay-btn');
    btn.disabled = true;
    btn.textContent = 'Processing...';

    var promises = spots.map(function(spotId) {
      return fetch('/api/admin/spots/' + spotId + '/pay', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
      }).then(function(res) {
        return res.json().then(function(data) {
          return { ok: res.ok, data: data, spotId: spotId };
        });
      });
    });

    Promise.all(promises).then(function(results) {
      results.forEach(function(r) {
        if (r.ok) {
          var el = document.querySelector('[data-spot-id="' + r.spotId + '"]');
          if (el) {
            el.dataset.spotStatus = 'paid';
            el.className = 'admin-spot-item relative rounded-lg border-2 text-center p-2 min-h-[44px] flex flex-col items-center justify-center transition-all duration-200 touch-manipulation select-none ' + SPOT_STATUS_CLASSES.paid.bg + ' ' + SPOT_STATUS_CLASSES.paid.border + ' ' + SPOT_STATUS_CLASSES.paid.text + ' cursor-default';
          }
          var listItem = document.querySelector('#pending-list [data-spot-id="' + r.spotId + '"]');
          if (listItem) {
            listItem.remove();
          }
        }
      });
      checkEmptyPendingList();
      selectedForBulk.clear();
      updateBulkUI();
    }).catch(function() {
      btn.disabled = false;
      updateBulkUI();
    });
  }

  function markSpotPaid(spotId, spotNumber, el, isListItem) {
    if (!confirm('Mark spot #' + spotNumber + ' as paid?')) return;

    var button;
    if (el && el.querySelector) {
      button = el.querySelector('.pay-single-btn');
    }
    if (button) {
      button.disabled = true;
      button.textContent = '...';
    }

    fetch('/api/admin/spots/' + spotId + '/pay', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    })
    .then(function(res) {
      return res.json().then(function(data) {
        return { ok: res.ok, data: data };
      });
    })
    .then(function(result) {
      if (!result.ok) {
        alert('Error: ' + (result.data.error || 'Failed to mark paid'));
        if (button) {
          button.disabled = false;
          button.textContent = 'Mark Paid';
        }
        return;
      }

      var gridSpot = document.querySelector('#spot-grid [data-spot-id="' + spotId + '"]');
      if (gridSpot) {
        gridSpot.dataset.spotStatus = 'paid';
        gridSpot.className = 'admin-spot-item relative rounded-lg border-2 text-center p-2 min-h-[44px] flex flex-col items-center justify-center transition-all duration-200 touch-manipulation select-none ' + SPOT_STATUS_CLASSES.paid.bg + ' ' + SPOT_STATUS_CLASSES.paid.border + ' ' + SPOT_STATUS_CLASSES.paid.text + ' cursor-default';
      }

      if (isListItem && el) {
        el.remove();
        checkEmptyPendingList();
      } else {
        var listItem = document.querySelector('#pending-list [data-spot-id="' + spotId + '"]');
        if (listItem) {
          listItem.remove();
          checkEmptyPendingList();
        }
      }
    })
    .catch(function() {
      alert('Network error. Please try again.');
      if (button) {
        button.disabled = false;
        button.textContent = 'Mark Paid';
      }
    });
  }

  function releaseSpot(spotId, spotNumber, el) {
    if (!confirm('Release spot #' + spotNumber + ' back to available? This cannot be undone.')) return;

    var btn = el.querySelector('.release-btn');
    if (btn) {
      btn.disabled = true;
      btn.textContent = '...';
    }

    fetch('/api/admin/spots/' + spotId + '/release', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    })
    .then(function(res) {
      return res.json().then(function(data) {
        return { ok: res.ok, data: data };
      });
    })
    .then(function(result) {
      if (!result.ok) {
        alert('Error: ' + (result.data.error || 'Failed to release spot'));
        if (btn) {
          btn.disabled = false;
          btn.textContent = 'Release';
        }
        return;
      }

      el.remove();
      checkEmptyPendingList();
    })
    .catch(function() {
      alert('Network error. Please try again.');
      if (btn) {
        btn.disabled = false;
        btn.textContent = 'Release';
      }
    });
  }

  function executeSetWinner() {
    var input = document.getElementById('winner-spot-input');
    var errorEl = document.getElementById('winner-error');
    var successEl = document.getElementById('winner-success');

    errorEl.classList.add('hidden');
    successEl.classList.add('hidden');

    var spotNumber = parseInt(input.value, 10);
    if (isNaN(spotNumber) || spotNumber < 1) {
      errorEl.textContent = 'Please enter a valid spot number.';
      errorEl.classList.remove('hidden');
      return;
    }

    if (!confirm('Set spot #' + spotNumber + ' as the WINNER? This will complete the waffle and mark all other paid spots as losers. This cannot be undone.')) return;

    var btn = document.getElementById('set-winner-btn');
    btn.disabled = true;
    btn.textContent = 'Setting Winner...';

    fetch('/api/admin/waffles/' + config.waffleId + '/winner', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ winning_spot_number: spotNumber })
    })
    .then(function(res) {
      return res.json().then(function(data) {
        return { ok: res.ok, data: data };
      });
    })
    .then(function(result) {
      if (!result.ok) {
        errorEl.textContent = result.data.error || 'Failed to set winner';
        errorEl.classList.remove('hidden');
        btn.disabled = false;
        btn.textContent = 'Set Winner';
        return;
      }

      successEl.textContent = 'Winner set! The waffle is now completed.';
      successEl.classList.remove('hidden');
      btn.disabled = true;

      setTimeout(function() {
        window.location.reload();
      }, 2000);
    })
    .catch(function() {
      errorEl.textContent = 'Network error. Please try again.';
      errorEl.classList.remove('hidden');
      btn.disabled = false;
      btn.textContent = 'Set Winner';
    });
  }

  function checkEmptyPendingList() {
    var list = document.getElementById('pending-list');
    if (!list.querySelector('.pending-claim-item')) {
      var p = document.createElement('p');
      p.className = 'text-sm text-base-content/40 text-center py-4';
      p.textContent = 'No pending claims';
      list.appendChild(p);
    }
  }

  return {
    init: init
  };
})();
