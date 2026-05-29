var FooterClock = (function() {
  'use strict';

  var serverTimeStr = '';
  var serverDate = null;
  var tzName = '';
  var intervalId = null;

  function parseServerTime(str) {
    if (!str) return null;
    var parts = str.match(/^(\d{1,2}):(\d{2})\s+(AM|PM)$/i);
    if (!parts) return null;
    var hours = parseInt(parts[1], 10);
    var minutes = parseInt(parts[2], 10);
    var ampm = parts[3].toUpperCase();
    if (ampm === 'PM' && hours !== 12) hours += 12;
    if (ampm === 'AM' && hours === 12) hours = 0;
    var now = new Date();
    var d = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate(), hours, minutes, 0));
    return d;
  }

  function formatTime(date, timeZone) {
    var h, m, ampm, formatted;
    try {
      formatted = new Intl.DateTimeFormat('en-US', {
        hour: 'numeric',
        minute: '2-digit',
        hour12: true,
        timeZone: timeZone
      }).format(date);
      return formatted;
    } catch (e) {
      h = date.getUTCHours();
      m = date.getUTCMinutes();
      ampm = h >= 12 ? 'PM' : 'AM';
      h = h % 12;
      if (h === 0) h = 12;
      return h + ':' + (m < 10 ? '0' + m : m) + ' ' + ampm;
    }
  }

  function getTimeZoneAbbreviation(timeZone) {
    var parts, i;
    try {
      parts = new Intl.DateTimeFormat('en-US', {
        timeZone: timeZone,
        timeZoneName: 'short'
      }).formatToParts(new Date());
      for (i = 0; i < parts.length; i++) {
        if (parts[i].type === 'timeZoneName') {
          return parts[i].value;
        }
      }
    } catch (e) {
      // fall through
    }
    return 'UTC';
  }

  function updateClocks() {
    var el = document.getElementById('footer-clocks');
    var now, utcTime, localTime, localTz;
    if (!el) return;

    now = new Date();
    utcTime = formatTime(now, 'UTC');

    if (tzName) {
      localTime = formatTime(now, tzName);
      localTz = getTimeZoneAbbreviation(tzName);
    } else {
      localTime = utcTime;
      localTz = 'UTC';
    }

    el.textContent = 'UTC ' + utcTime + ' | Local ' + localTime + ' ' + localTz;
  }

  function init() {
    var footer = document.querySelector('footer[data-server-time]');
    if (footer) {
      serverTimeStr = footer.getAttribute('data-server-time') || '';
    }
    serverDate = parseServerTime(serverTimeStr);

    try {
      tzName = Intl.DateTimeFormat().resolvedOptions().timeZone || '';
    } catch (e) {
      tzName = '';
    }

    updateClocks();

    if (intervalId) {
      clearInterval(intervalId);
    }
    intervalId = setInterval(updateClocks, 60000);
  }

  return {
    init: init
  };
})();
