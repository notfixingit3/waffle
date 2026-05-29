var WaffleWebSocket = (function() {
  'use strict';

  var ws = null;
  var reconnectTimer = null;
  var reconnectDelay = 3000;
  var currentSlug = null;
  var messageHandler = null;
  var activityFlashTimer = null;
  var maxRetries = 10;
  var retryCount = 0;

  function connect(slug, handler) {
    currentSlug = slug;
    messageHandler = handler;

    if (!navigator.onLine) {
      updateStatus('disconnected');
      return;
    }

    var protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    var url = protocol + '//' + window.location.host + '/ws/' + slug;

    tryConnect(url);
  }

  function tryConnect(url) {
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
      return;
    }

    ws = new WebSocket(url);
    updateStatus('connecting');

    ws.onopen = function() {
      updateStatus('connected');
      reconnectDelay = 3000;
      retryCount = 0;
    };

    ws.onmessage = function(event) {
      try {
        var msg = JSON.parse(event.data);
        if (msg.type === 'ACTIVITY_EVENT') {
          flashActivity(msg.payload);
        }
        if (msg.type === 'SPOT_UPDATED' && msg.payload && currentSlug) {
          updateCachedSpot(currentSlug, msg.payload);
        }
        if (messageHandler) {
          messageHandler(msg);
        }
      } catch (e) {
        // Ignore parse errors
      }
    };

    ws.onclose = function() {
      updateStatus('disconnected');
      ws = null;
      scheduleReconnect();
    };

    ws.onerror = function() {
      // onclose will fire after onerror
    };
  }

  function scheduleReconnect() {
    if (reconnectTimer) return;

    retryCount++;
    if (retryCount >= maxRetries) {
      updateStatus('disconnected');
      return;
    }

    updateStatus('reconnecting');

    var delay = reconnectDelay + Math.random() * 1000;

    reconnectTimer = setTimeout(function() {
      reconnectTimer = null;
      if (currentSlug && messageHandler) {
        var protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        var url = protocol + '//' + window.location.host + '/ws/' + currentSlug;
        tryConnect(url);
      }
    }, delay);

    reconnectDelay = Math.min(reconnectDelay * 1.5, 30000);
  }

  function updateStatus(status) {
    var container = document.getElementById('ws-status');
    if (!container) return;

    var dot = container.querySelector('span:first-child');
    var label = container.querySelector('span:last-child');

    if (!dot || !label) return;

    if (activityFlashTimer) {
      clearTimeout(activityFlashTimer);
      activityFlashTimer = null;
    }

    dot.className = 'w-2 h-2 rounded-full inline-block';

    switch (status) {
      case 'connected':
        dot.className += ' bg-success';
        label.textContent = 'live';
        label.className = 'text-xs text-success';
        break;
      case 'connecting':
        dot.className += ' bg-warning animate-pulse';
        label.textContent = 'connecting';
        label.className = 'text-xs text-warning';
        break;
      case 'reconnecting':
        dot.className += ' bg-warning animate-pulse';
        label.textContent = 'reconnecting';
        label.className = 'text-xs text-warning';
        break;
      case 'disconnected':
      default:
        dot.className += ' bg-base-300';
        label.textContent = 'offline';
        label.className = 'text-xs text-base-content/40';
        break;
    }
  }

  function flashActivity(payload) {
    var container = document.getElementById('ws-status');
    if (!container) return;

    var dot = container.querySelector('span:first-child');
    var label = container.querySelector('span:last-child');
    if (!dot || !label) return;

    dot.className = 'w-2 h-2 rounded-full inline-block bg-info animate-pulse';
    label.textContent = payload.message || 'activity';
    label.className = 'text-xs text-info';

    if (activityFlashTimer) {
      clearTimeout(activityFlashTimer);
    }

    activityFlashTimer = setTimeout(function() {
      activityFlashTimer = null;
      var status = (ws && ws.readyState === WebSocket.OPEN) ? 'connected' : 'disconnected';
      if (reconnectTimer) status = 'reconnecting';
      updateStatus(status);
    }, 2000);
  }

  function updateCachedSpot(slug, payload) {
    if (typeof OfflineHandler === 'undefined') return;
    var cached = OfflineHandler.getCachedData(slug);
    if (!cached || !cached.data) return;
    var found = false;
    for (var i = 0; i < cached.data.length; i++) {
      if (cached.data[i].number === payload.spot_number) {
        cached.data[i].status = payload.status;
        if (payload.claimed_by_handle !== undefined) {
          cached.data[i].claimed_by_handle = payload.claimed_by_handle;
        }
        found = true;
        break;
      }
    }
    if (found) {
      localStorage.setItem('waffle-data-' + slug, JSON.stringify({
        data: cached.data,
        timestamp: Date.now()
      }));
    }
  }

  function disconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    if (ws) {
      ws.close();
      ws = null;
    }
    currentSlug = null;
    messageHandler = null;
  }

  window.addEventListener('beforeunload', function() {
    disconnect();
  });

  window.addEventListener('waffle:offline', function() {
    if (ws) {
      ws.close();
      ws = null;
    }
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    updateStatus('disconnected');
  });

  window.addEventListener('waffle:online', function() {
    if (currentSlug && messageHandler && !ws) {
      reconnectDelay = 3000;
      var protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      var url = protocol + '//' + window.location.host + '/ws/' + currentSlug;
      tryConnect(url);
    }
  });

  return {
    connect: connect,
    disconnect: disconnect
  };
})();
