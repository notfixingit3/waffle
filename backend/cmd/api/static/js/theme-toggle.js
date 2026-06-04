var ThemeToggle = (function() {
  'use strict';
  function init() {
    document.documentElement.setAttribute('data-theme', 'syrup');
  }
  function toggle() {}
  return { init: init, toggle: toggle };
})();
