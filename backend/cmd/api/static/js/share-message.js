var ShareMessage = (function() {
  'use strict';

  var csrfToken = '';
  var csrfMeta = document.querySelector('meta[name="csrf-token"]');
  if (csrfMeta) {
    csrfToken = csrfMeta.getAttribute('content') || '';
  }

  var container = document.querySelector('.waffle-manage');
  var slug = container ? container.dataset.waffleSlug : '';

  var select = document.getElementById('share-template-select');
  var message = document.getElementById('share-message');
  var publicUrl = document.getElementById('share-public-url');
  var downloadLink = document.getElementById('share-download-link');
  var statusEl = document.getElementById('share-status');
  var errorEl = document.getElementById('share-error');
  var noTemplatesWarning = document.getElementById('share-no-templates-warning');
  var format = 'story';

  function showStatus(msg, isError) {
    if (statusEl) statusEl.classList.add('hidden');
    if (errorEl) errorEl.classList.add('hidden');
    if (!msg) return;
    if (isError) {
      if (errorEl) {
        errorEl.textContent = msg;
        errorEl.classList.remove('hidden');
      }
    } else {
      if (statusEl) {
        statusEl.textContent = msg;
        statusEl.classList.remove('hidden');
      }
    }
  }

  function copyText(text) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      return navigator.clipboard.writeText(text);
    }
    return new Promise(function(resolve, reject) {
      var ta = document.createElement('textarea');
      ta.value = text;
      ta.style.position = 'fixed';
      ta.style.opacity = '0';
      document.body.appendChild(ta);
      ta.select();
      try {
        if (document.execCommand('copy')) {
          resolve();
        } else {
          reject(new Error('execCommand copy failed'));
        }
      } catch (e) {
        reject(e);
      } finally {
        document.body.removeChild(ta);
      }
    });
  }

  function populateTemplates(templates, selectedId) {
    if (!select) return;
    select.innerHTML = '<option value="">— Select a template —</option>';
    templates.forEach(function(t) {
      var opt = document.createElement('option');
      opt.value = t.id;
      opt.textContent = t.name;
      if (t.id === selectedId) opt.selected = true;
      select.appendChild(opt);
    });

    if (noTemplatesWarning) {
      if (!templates || templates.length === 0) {
        noTemplatesWarning.classList.remove('hidden');
      } else {
        noTemplatesWarning.classList.add('hidden');
      }
    }
  }

  function updateDownloadLink() {
    if (!downloadLink || !slug) return;
    downloadLink.href = '/waffle/' + encodeURIComponent(slug) + '/card.png?format=' + format;
  }

  function setFormat(fmt) {
    format = fmt;
    var storyBtn = document.getElementById('share-format-story');
    var squareBtn = document.getElementById('share-format-square');
    if (storyBtn) storyBtn.classList.toggle('btn-active', fmt === 'story');
    if (squareBtn) squareBtn.classList.toggle('btn-active', fmt === 'square');
    updateDownloadLink();
  }

  function load() {
    if (!slug) return;
    fetch('/api/admin/waffles/' + encodeURIComponent(slug) + '/share-message', {
      headers: { 'Accept': 'application/json' }
    })
    .then(function(res) {
      return res.json().then(function(data) {
        return { res: res, data: data };
      });
    })
    .then(function(result) {
      if (!result.res.ok) {
        showStatus(result.data.error || 'Failed to load share message', true);
        return;
      }
      populateTemplates(result.data.templates || [], result.data.selected_template_id);
      if (message && result.data.message != null) {
        message.value = result.data.message;
      }
    })
    .catch(function() {
      showStatus('Failed to load share message', true);
    });
  }

  function renderPreview() {
    if (!select || !slug || !select.value) return;
    fetch('/api/admin/waffles/' + encodeURIComponent(slug) + '/share-message/render', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
        'X-CSRF-Token': csrfToken
      },
      body: JSON.stringify({ template_id: select.value })
    })
    .then(function(res) {
      return res.json().then(function(data) {
        return { res: res, data: data };
      });
    })
    .then(function(result) {
      if (!result.res.ok) {
        showStatus(result.data.error || 'Failed to render preview', true);
        return;
      }
      if (message) message.value = result.data.message || '';
    })
    .catch(function() {
      showStatus('Failed to render preview', true);
    });
  }

  function save() {
    if (!slug) return;
    var payload = {};
    if (select && select.value) payload.template_id = select.value;
    if (message) payload.message = message.value;

    showStatus('', false);
    var btn = document.getElementById('share-save-btn');
    if (btn) {
      btn.disabled = true;
      btn.textContent = 'Saving...';
    }

    fetch('/api/admin/waffles/' + encodeURIComponent(slug) + '/share-message', {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
        'X-CSRF-Token': csrfToken
      },
      body: JSON.stringify(payload)
    })
    .then(function(res) {
      return res.json().then(function(data) {
        return { res: res, data: data };
      });
    })
    .then(function(result) {
      if (!result.res.ok) {
        showStatus(result.data.error || 'Failed to save share message', true);
        return;
      }
      showStatus('Share message saved', false);
    })
    .catch(function() {
      showStatus('Failed to save share message', true);
    })
    .finally(function() {
      if (btn) {
        btn.disabled = false;
        btn.textContent = 'Save Message';
      }
    });
  }

  function regenerate() {
    if (!slug) return;
    var btn = document.getElementById('share-regenerate-btn');
    if (btn) {
      btn.disabled = true;
      btn.textContent = 'Regenerating...';
    }
    showStatus('', false);

    fetch('/api/admin/waffles/' + encodeURIComponent(slug) + '/share-message/regenerate-card', {
      method: 'POST',
      headers: {
        'Accept': 'application/json',
        'X-CSRF-Token': csrfToken
      }
    })
    .then(function(res) {
      return res.json().then(function(data) {
        return { res: res, data: data };
      });
    })
    .then(function(result) {
      if (!result.res.ok) {
        showStatus(result.data.error || 'Failed to regenerate card', true);
        return;
      }
      if (downloadLink && slug) {
        downloadLink.href = '/waffle/' + encodeURIComponent(slug) + '/card.png?format=' + format + '&t=' + Date.now();
      }
      showStatus('Card regenerated', false);
    })
    .catch(function() {
      showStatus('Failed to regenerate card', true);
    })
    .finally(function() {
      if (btn) {
        btn.disabled = false;
        btn.textContent = 'Regenerate Card';
      }
    });
  }

  function bind() {
    if (select) select.addEventListener('change', renderPreview);

    var saveBtn = document.getElementById('share-save-btn');
    if (saveBtn) saveBtn.addEventListener('click', save);

    var copyBtn = document.getElementById('share-copy-btn');
    if (copyBtn) {
      copyBtn.addEventListener('click', function() {
        if (!message) return;
        copyText(message.value)
          .then(function() { showStatus('Message copied', false); })
          .catch(function() { showStatus('Failed to copy message', true); });
      });
    }

    var copyUrlBtn = document.getElementById('share-copy-url-btn');
    if (copyUrlBtn) {
      copyUrlBtn.addEventListener('click', function() {
        if (!publicUrl) return;
        copyText(publicUrl.value)
          .then(function() { showStatus('URL copied', false); })
          .catch(function() { showStatus('Failed to copy URL', true); });
      });
    }

    var storyBtn = document.getElementById('share-format-story');
    var squareBtn = document.getElementById('share-format-square');
    if (storyBtn) storyBtn.addEventListener('click', function() { setFormat('story'); });
    if (squareBtn) squareBtn.addEventListener('click', function() { setFormat('square'); });

    var regenerateBtn = document.getElementById('share-regenerate-btn');
    if (regenerateBtn) regenerateBtn.addEventListener('click', regenerate);
  }

  bind();

  return {
    load: load
  };
})();

document.addEventListener('DOMContentLoaded', function() {
  ShareMessage.load();
});