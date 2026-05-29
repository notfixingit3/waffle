var ThemeToggle = (function() {
  'use strict';

  var STORAGE_KEY = 'syrup-theme';
  var DARK_THEME = 'syrup';
  var LIGHT_THEME = 'syrup-light';

  function getSavedTheme() {
    var saved;
    try {
      saved = localStorage.getItem(STORAGE_KEY);
      return saved || DARK_THEME;
    } catch (e) {
      return DARK_THEME;
    }
  }

  function setTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
  }

  function updateIcons() {
    var current = document.documentElement.getAttribute('data-theme');
    var isDark = current !== LIGHT_THEME;
    var sunEls = document.querySelectorAll('.theme-sun');
    var moonEls = document.querySelectorAll('.theme-moon');
    var i;

    for (i = 0; i < sunEls.length; i++) {
      if (isDark) {
        sunEls[i].classList.remove('hidden');
      } else {
        sunEls[i].classList.add('hidden');
      }
    }

    for (i = 0; i < moonEls.length; i++) {
      if (isDark) {
        moonEls[i].classList.add('hidden');
      } else {
        moonEls[i].classList.remove('hidden');
      }
    }
  }

  function init() {
    var theme = getSavedTheme();
    setTheme(theme);
    updateIcons();
  }

  function toggle() {
    var current = document.documentElement.getAttribute('data-theme');
    var next = current === DARK_THEME ? LIGHT_THEME : DARK_THEME;

    try {
      localStorage.setItem(STORAGE_KEY, next);
    } catch (e) {
      // localStorage unavailable — continue with in-memory toggle
    }

    setTheme(next);
    updateIcons();
  }

  return {
    init: init,
    toggle: toggle
  };
})();
