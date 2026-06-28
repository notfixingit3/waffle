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

    var clearWinnerBtn = document.getElementById('clear-winner-btn');
    if (clearWinnerBtn) {
      clearWinnerBtn.addEventListener('click', executeClearWinner);
    }

    var changeWinnerBtn = document.getElementById('change-winner-btn');
    if (changeWinnerBtn) {
      changeWinnerBtn.addEventListener('click', executeChangeWinner);
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

    var csrfToken = (document.querySelector('meta[name="csrf-token"]') || {}).content || '';

    var promises = spots.map(function(spotId) {
      return fetch('/api/admin/spots/' + spotId + '/pay', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }
      }).then(function(res) {
        return res.json().then(function(data) {
          return { ok: res.ok, data: data, spotId: spotId };
        });
      });
    });

    Promise.all(promises).then(function(results) {
      var succeeded = 0;
      var failed = 0;
      results.forEach(function(r) {
        if (r.ok) {
          succeeded++;
          var el = document.querySelector('[data-spot-id="' + r.spotId + '"]');
          if (el) {
            el.dataset.spotStatus = 'paid';
            el.className = config.spotBaseClasses + ' ' + config.spotPaidClasses;
          }
          var listItem = document.querySelector('#pending-list [data-spot-id="' + r.spotId + '"]');
          if (listItem) {
            listItem.remove();
          }
        } else {
          failed++;
        }
      });
      checkEmptyPendingList();
      selectedForBulk.clear();
      updateBulkUI();
      if (succeeded > 0) {
        AdminToast.success(succeeded + ' spot' + (succeeded === 1 ? '' : 's') + ' marked as paid.');
      }
      if (failed > 0) {
        AdminToast.error(failed + ' spot' + (failed === 1 ? '' : 's') + ' failed to update. Please try again.');
      }
    }).catch(function() {
      btn.disabled = false;
      updateBulkUI();
      AdminToast.error('Network error during bulk pay. Please try again.');
    });
  }

  function markSpotPaid(spotId, spotNumber, el, isListItem) {
    AdminConfirm.show('Mark spot #' + spotNumber + ' as paid?', function() {
      var button;
      if (el && el.querySelector) {
        button = el.querySelector('.pay-single-btn');
      }
      if (button) {
        button.disabled = true;
        button.textContent = '...';
      }

      var csrfToken = (document.querySelector('meta[name="csrf-token"]') || {}).content || '';
      fetch('/api/admin/spots/' + spotId + '/pay', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }
      })
      .then(function(res) {
        return res.json().then(function(data) {
          return { ok: res.ok, data: data };
        });
      })
      .then(function(result) {
        if (!result.ok) {
          AdminToast.error('Error: ' + (result.data.error || 'Failed to mark paid'));
          if (button) {
            button.disabled = false;
            button.textContent = 'Mark Paid';
          }
          return;
        }

        var gridSpot = document.querySelector('#spot-grid [data-spot-id="' + spotId + '"]');
        if (gridSpot) {
          gridSpot.dataset.spotStatus = 'paid';
          gridSpot.className = config.spotBaseClasses + ' ' + config.spotPaidClasses;
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
        AdminToast.error('Network error. Please try again.');
        if (button) {
          button.disabled = false;
          button.textContent = 'Mark Paid';
        }
      });
    }, { title: 'Mark Paid', confirmLabel: 'Mark Paid', confirmClass: 'btn-success' });
  }

  function releaseSpot(spotId, spotNumber, el) {
    AdminConfirm.show('Release spot #' + spotNumber + ' back to available? This cannot be undone.', function() {
      var btn = el.querySelector('.release-btn');
      if (btn) {
        btn.disabled = true;
        btn.textContent = '...';
      }

      var csrfToken = (document.querySelector('meta[name="csrf-token"]') || {}).content || '';
      fetch('/api/admin/spots/' + spotId + '/release', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }
      })
      .then(function(res) {
        return res.json().then(function(data) {
          return { ok: res.ok, data: data };
        });
      })
      .then(function(result) {
        if (!result.ok) {
          AdminToast.error('Error: ' + (result.data.error || 'Failed to release spot'));
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
        AdminToast.error('Network error. Please try again.');
        if (btn) {
          btn.disabled = false;
          btn.textContent = 'Release';
        }
      });
    }, { title: 'Release Spot', confirmLabel: 'Release', confirmClass: 'btn-warning' });
  }

  function executeSetWinner() {
    var errorEl = document.getElementById('winner-error');
    var successEl = document.getElementById('winner-success');

    errorEl.classList.add('hidden');
    successEl.classList.add('hidden');

    var selects = document.querySelectorAll('.winner-spot-select');
    var spotNumbers = [];
    for (var i = 0; i < selects.length; i++) {
      var val = parseInt(selects[i].value, 10);
      if (isNaN(val) || val < 1) {
        errorEl.textContent = 'Please select a winning spot for all items.';
        errorEl.classList.remove('hidden');
        return;
      }
      spotNumbers.push(val);
    }

    var uniqueSpots = new Set(spotNumbers);
    if (uniqueSpots.size < spotNumbers.length) {
      if (!confirm('You have selected the same spot number for multiple items. Are you sure you want to proceed?')) {
        return;
      }
    }

    if (!confirm('Set the selected spot(s) as winner(s)? This will complete the waffle and mark all other paid spots as losers. This cannot be undone.')) return;

    var btn = document.getElementById('set-winner-btn');
    btn.disabled = true;
    btn.textContent = 'Setting Winner...';

    var csrfToken = (document.querySelector('meta[name="csrf-token"]') || {}).content || '';
    fetch('/api/admin/waffles/' + config.waffleId + '/winner', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({ winning_spot_numbers: spotNumbers })
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

  function executeChangeWinner() {
    var errorEl = document.getElementById('winner-error');
    var successEl = document.getElementById('winner-success');

    errorEl.classList.add('hidden');
    successEl.classList.add('hidden');

    var selects = document.querySelectorAll('.winner-spot-select');
    var spotNumbers = [];
    for (var i = 0; i < selects.length; i++) {
      var val = parseInt(selects[i].value, 10);
      if (isNaN(val) || val < 1) {
        errorEl.textContent = 'Please select a winning spot for all items.';
        errorEl.classList.remove('hidden');
        return;
      }
      spotNumbers.push(val);
    }

    var uniqueSpots = new Set(spotNumbers);
    if (uniqueSpots.size < spotNumbers.length) {
      if (!confirm('You have selected the same spot number for multiple items. Are you sure you want to proceed?')) {
        return;
      }
    }

    if (!confirm('Change the winning spot(s) to the selected spot(s)? This will recalculate winners, losers, and buyer stats. This cannot be undone.')) return;

    var btn = document.getElementById('change-winner-btn');
    btn.disabled = true;
    btn.textContent = 'Changing Winner...';

    var csrfToken = (document.querySelector('meta[name="csrf-token"]') || {}).content || '';
    fetch('/api/admin/waffles/' + config.waffleId + '/change-winner', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({ winning_spot_numbers: spotNumbers })
    })
    .then(function(res) {
      return res.json().then(function(data) {
        return { ok: res.ok, data: data };
      });
    })
    .then(function(result) {
      if (!result.ok) {
        errorEl.textContent = result.data.error || 'Failed to change winner';
        errorEl.classList.remove('hidden');
        btn.disabled = false;
        btn.textContent = 'Change Winner';
        return;
      }

      successEl.textContent = 'Winner changed successfully!';
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
      btn.textContent = 'Change Winner';
    });
  }

  function executeClearWinner() {
    var errorEl = document.getElementById('winner-error');
    var successEl = document.getElementById('winner-success');

    errorEl.classList.add('hidden');
    successEl.classList.add('hidden');

    if (!confirm('Clear the winners and reopen this waffle? All spots will return to PAID status, and buyer win/loss stats will be reverted. This cannot be undone.')) return;

    var btn = document.getElementById('clear-winner-btn');
    btn.disabled = true;
    btn.textContent = 'Clearing Winner...';

    var csrfToken = (document.querySelector('meta[name="csrf-token"]') || {}).content || '';
    fetch('/api/admin/waffles/' + config.waffleId + '/clear-winner', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken }
    })
    .then(function(res) {
      return res.json().then(function(data) {
        return { ok: res.ok, data: data };
      });
    })
    .then(function(result) {
      if (!result.ok) {
        errorEl.textContent = result.data.error || 'Failed to clear winner';
        errorEl.classList.remove('hidden');
        btn.disabled = false;
        btn.textContent = 'Clear Winner & Reopen Waffle';
        return;
      }

      successEl.textContent = 'Winners cleared and waffle reopened!';
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
      btn.textContent = 'Clear Winner & Reopen Waffle';
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
