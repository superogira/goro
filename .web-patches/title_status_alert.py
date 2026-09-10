import io, re, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new, count=1):
    global src
    if src.count(old) < count:
        print("ANCHOR NOT FOUND:", old[:90]); sys.exit(1)
    src = src.replace(old, new, count)

# 1) CSS: status rows + alert panel + panel lift transitions
sub1(
    "  #goro-service .foot button.ok { background: #ffe9b0; font-weight: 600; }",
    """  #goro-service .foot button.ok { background: #ffe9b0; font-weight: 600; }
  #goro-login .status, #goro-service .status {
    min-height: 15px; padding: 0 14px 8px; text-align: center;
    font-size: 12px; color: #6a6055; word-break: break-word;
  }
  #goro-login .status.err, #goro-service .status.err { color: #b23b3b; font-weight: 600; }
  #goro-login { transition: transform .18s ease-out; }
  #goro-alert {
    position: absolute; left: 50%; top: 42%; transform: translate(-50%, -50%);
    width: 286px; max-width: 92vw; background: rgba(255, 255, 252, .96);
    border: 1px solid rgba(120, 110, 90, .55); border-radius: 8px;
    color: #2a2622; user-select: none; overflow: hidden;
    font: 13px/1.3 Sarabun,'Noto Sans Thai',system-ui,sans-serif; pointer-events: auto;
  }
  #goro-alert .title {
    height: 24px; display: flex; align-items: center; justify-content: center; font-weight: 600;
    background: linear-gradient(180deg, #9cc9f0 0%, #5b95d4 52%, #3d72b5 100%);
    color: #fff; text-shadow: 0 1px 1px rgba(15, 35, 65, .55);
  }
  #goro-alert .msg { padding: 14px 16px; white-space: pre-wrap; word-break: break-word; }
  #goro-alert .foot { display: flex; justify-content: flex-end; gap: 6px; padding: 0 12px 12px; }
  #goro-alert .foot button {
    min-width: 74px; height: 26px; border: 1px solid rgba(120, 110, 90, .5); border-radius: 6px;
    background: rgba(232, 223, 200, .95); color: #2a2622; cursor: pointer;
    font: 13px Sarabun,'Noto Sans Thai',system-ui,sans-serif;
  }
  #goro-alert .foot button:active { background: #d8cba8; }
  #goro-alert .foot button.ok { background: #ffe9b0; font-weight: 600; }""",
)

# 2) console transition gains transform
sub1(
    "  #goro-console { transition: opacity .4s; }",
    "  #goro-console { transition: opacity .4s, transform .18s ease-out; }",
)

# 3) markup: alert panel inside the title layer
sub1(
    '<div id="goro-title"><img alt=""><div id="goro-login" style="display:none"></div><div id="goro-service" style="display:none"></div></div>',
    '<div id="goro-title"><img alt=""><div id="goro-login" style="display:none"></div><div id="goro-service" style="display:none"></div><div id="goro-alert" style="display:none"></div></div>',
)

# 4) titleAct helper
sub1(
    "    var svcAct = function (a) { if (window.goroServiceAction) window.goroServiceAction(a); };",
    """    var svcAct = function (a) { if (window.goroServiceAction) window.goroServiceAction(a); };
    var titleAct = function (a) { if (window.goroTitleAction) window.goroTitleAction(a); };""",
)

# 5) goroTitleSync: status line + modal alert
sub1(
    """    window.goroTitleSync = function (s) {
      if (s.bg && img.getAttribute('src') !== s.bg) img.setAttribute('src', s.bg);
      var show = s.phase === 'account' && s.fade < 0.999;
      layer.style.display = show ? 'block' : 'none';
      layer.style.opacity = String(1 - s.fade);
    };""",
    """    window.goroTitleSync = function (s) {
      if (s.bg && img.getAttribute('src') !== s.bg) img.setAttribute('src', s.bg);
      var show = s.phase === 'account' && s.fade < 0.999;
      layer.style.display = show ? 'block' : 'none';
      layer.style.opacity = String(1 - s.fade);
      var err = !!(s.alert && s.alert.message);
      ['goro-login', 'goro-service'].forEach(function (id) {
        var row = document.querySelector('#' + id + ' .status');
        if (row) { row.textContent = s.status || ''; row.classList.toggle('err', err); }
      });
      var alertEl = document.getElementById('goro-alert');
      if (s.alert && show) {
        var html = '<div class="title">' + esc(s.alert.title || 'Notice') + '</div>' +
          '<div class="msg">' + esc(s.alert.message || '') + '</div>' +
          '<div class="foot"><button class="ok" data-a="alertok">OK</button>' +
          (s.alert.okOnly ? '' : '<button data-a="alertcancel">Cancel</button>') + '</div>';
        if (alertEl.getAttribute('data-sig') !== s.alert.title + '|' + s.alert.message + '|' + !!s.alert.okOnly) {
          alertEl.setAttribute('data-sig', s.alert.title + '|' + s.alert.message + '|' + !!s.alert.okOnly);
          alertEl.innerHTML = html;
          alertEl.querySelectorAll('.foot button').forEach(function (el) {
            el.addEventListener('pointerdown', function (ev) {
              ev.stopPropagation();
              titleAct(el.getAttribute('data-a'));
            });
          });
        }
        alertEl.style.display = 'block';
      } else {
        alertEl.style.display = 'none';
      }
    };""",
)

# 6) login panel gains a status row
sub1(
    "'</div><div class=\"foot\"><button data-a=\"submit\">Login</button></div>';",
    "'</div><div class=\"status\"></div><div class=\"foot\"><button data-a=\"submit\">Login</button></div>';",
)

# 7) service panel gains a status row
sub1(
    "html += '</div><div class=\"foot\"><button class=\"ok\" data-a=\"ok\">OK</button><button data-a=\"cancel\">Cancel</button></div>';",
    "html += '</div><div class=\"status\"></div><div class=\"foot\"><button class=\"ok\" data-a=\"ok\">OK</button><button data-a=\"cancel\">Cancel</button></div>';",
)

# 8) DOM panel keyboard lift: definition block before goroKeyboardPan
sub1(
    "  window.goroKeyboardPan = () => { panRetries = 0; panCompute(); };",
    """  // DOM text-field panels (login form, chat console): lift the focused
  // panel above the OS keyboard with a pure transform on the panel. The
  // game canvas is never touched — resizing or transforming it makes the
  // renderer rebuild and hitch.
  let kbPanEl = null, kbPanRetry = 0, kbPanZeroTicks = 0;
  const kbPanelOf = (el) => (el && el.closest ? el.closest('#goro-login, #goro-console') : null);
  const kbPanApply = (el, pan) => {
    const base = el.id === 'goro-login' ? 'translate(-50%, -50%)' : '';
    el.style.transform = pan > 0 ? base + ' translateY(' + (-pan) + 'px)' : base;
  };
  const panDomPanels = () => {
    const kb = keyboardHeight();
    if (!kbPanEl || kb <= 0) { if (kbPanEl) kbPanApply(kbPanEl, 0); return; }
    const r = kbPanEl.getBoundingClientRect();
    const overlap = r.bottom - (window.innerHeight - kb) + 28;
    kbPanApply(kbPanEl, Math.max(0, overlap));
  };
  const panDomPanelsSoon = () => {
    if (!kbPanEl) return;
    panDomPanels();
    if (keyboardHeight() === 0 && kbPanRetry < 20) { kbPanRetry++; setTimeout(panDomPanelsSoon, 250); }
  };
  document.addEventListener('focusin', (ev) => {
    const panel = kbPanelOf(ev.target);
    if (!panel) return;
    kbPanEl = panel; kbPanRetry = 0; panDomPanelsSoon();
  });
  document.addEventListener('focusout', () => {
    setTimeout(() => {
      const panel = kbPanelOf(document.activeElement);
      if (panel === kbPanEl) return;
      if (kbPanEl) kbPanApply(kbPanEl, 0);
      kbPanEl = panel;
      if (kbPanEl) { kbPanRetry = 0; panDomPanelsSoon(); }
    }, 60);
  });
  window.goroKeyboardPan = () => { panRetries = 0; panCompute(); panDomPanels(); };""",
)

# 9) collapse watchdog also restores DOM panels
sub1(
    """  let kbZeroTicks = 0;
  setInterval(() => {
    const canvas = document.querySelector('canvas');
    if (!canvas || !canvas.style.transform) { kbZeroTicks = 0; return; }
    if (keyboardHeight() === 0) {
      if (++kbZeroTicks >= 2) { canvas.style.transform = ''; kbZeroTicks = 0; }
    } else {
      kbZeroTicks = 0;
    }
  }, 300);""",
    """  let kbZeroTicks = 0;
  setInterval(() => {
    const zero = keyboardHeight() === 0;
    const canvas = document.querySelector('canvas');
    if (canvas && canvas.style.transform) {
      if (zero) {
        if (++kbZeroTicks >= 2) { canvas.style.transform = ''; kbZeroTicks = 0; }
      } else {
        kbZeroTicks = 0;
      }
    } else {
      kbZeroTicks = 0;
    }
    if (kbPanEl && zero && kbPanEl.style.transform) {
      if (++kbPanZeroTicks >= 2) { kbPanApply(kbPanEl, 0); kbPanZeroTicks = 0; }
    } else {
      kbPanZeroTicks = 0;
    }
  }, 300);""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
