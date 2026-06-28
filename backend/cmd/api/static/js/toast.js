var AdminToast = (function() {
  'use strict';

  var container = null;

  function getContainer() {
    if (!container) {
      container = document.createElement('div');
      container.className = 'toast toast-top toast-center z-[9999]';
      document.body.appendChild(container);
    }
    return container;
  }

  function show(message, type) {
    var c = getContainer();
    var el = document.createElement('div');
    var typeClass = {
      error: 'alert-error',
      success: 'alert-success',
      warning: 'alert-warning',
      info: 'alert-info'
    }[type] || 'alert-info';
    el.className = 'alert ' + typeClass + ' shadow-lg max-w-sm text-sm';
    el.textContent = message;
    c.appendChild(el);
    setTimeout(function() {
      if (el.parentNode) el.parentNode.removeChild(el);
    }, 4000);
  }

  return {
    show: show,
    error:   function(msg) { show(msg, 'error'); },
    success: function(msg) { show(msg, 'success'); },
    warning: function(msg) { show(msg, 'warning'); },
    info:    function(msg) { show(msg, 'info'); }
  };
})();
