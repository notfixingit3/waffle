var OfflineHandler = (function() {
  'use strict';

  var PREFIX = 'waffle-data-';

  function getCachedData(key) {
    try {
      var raw = localStorage.getItem(PREFIX + key);
      if (!raw) return null;
      var parsed = JSON.parse(raw);
      if (!parsed || !parsed.data) return null;
      return parsed;
    } catch (e) {
      return null;
    }
  }

  function setCachedData(key, data) {
    try {
      localStorage.setItem(PREFIX + key, JSON.stringify({
        data: data,
        timestamp: Date.now()
      }));
    } catch (e) {
      // localStorage unavailable or full — silently ignore
    }
  }

  function getLastUpdated(key) {
    var cached = getCachedData(key);
    if (!cached || !cached.timestamp) return null;
    var diff = Date.now() - cached.timestamp;
    var minutes = Math.floor(diff / 60000);
    if (minutes < 1) return 'just now';
    if (minutes === 1) return '1 minute ago';
    if (minutes < 60) return minutes + ' minutes ago';
    var hours = Math.floor(minutes / 60);
    if (hours === 1) return '1 hour ago';
    return hours + ' hours ago';
  }

  return {
    getCachedData: getCachedData,
    setCachedData: setCachedData,
    getLastUpdated: getLastUpdated
  };
})();
