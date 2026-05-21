var WaffleWebSocket = (function() {
  'use strict';

  var ws = null;
  var reconnectTimer = null;
  var reconnectDelay = 3000;
  var currentSlug = null;
  var messageHandler = null;
  var activityFlashTimer = null;

  function connect(slug, handler) {
    currentSlug = slug;
    messageHandler = handler;

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
    };

    ws.onmessage = function(event) {
      try {
        var msg = JSON.parse(event.data);
        if (msg.type === 'ACTIVITY_EVENT') {
          flashActivity(msg.payload);
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
    updateStatus('reconnecting');

    reconnectTimer = setTimeout(function() {
      reconnectTimer = null;
      if (currentSlug && messageHandler) {
        var protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        var url = protocol + '//' + window.location.host + '/ws/' + currentSlug;
        tryConnect(url);
      }
    }, reconnectDelay);

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
        dot.className += ' bg-green-500';
        label.textContent = 'live';
        label.className = 'text-xs text-green-600';
        break;
      case 'connecting':
        dot.className += ' bg-yellow-500 animate-pulse';
        label.textContent = 'connecting';
        label.className = 'text-xs text-yellow-600';
        break;
      case 'reconnecting':
        dot.className += ' bg-yellow-500 animate-pulse';
        label.textContent = 'reconnecting';
        label.className = 'text-xs text-yellow-600';
        break;
      case 'disconnected':
      default:
        dot.className += ' bg-gray-300';
        label.textContent = 'offline';
        label.className = 'text-xs text-gray-400';
        break;
    }
  }

  function flashActivity(payload) {
    var container = document.getElementById('ws-status');
    if (!container) return;

    var dot = container.querySelector('span:first-child');
    var label = container.querySelector('span:last-child');
    if (!dot || !label) return;

    dot.className = 'w-2 h-2 rounded-full inline-block bg-blue-500 animate-pulse';
    label.textContent = payload.message || 'activity';
    label.className = 'text-xs text-blue-600';

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

  return {
    connect: connect,
    disconnect: disconnect
  };
})();
