import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# AEOPD document registry window: opened by clicking the hidden NPC at the
# Prontera Library bookshelf (goroSarabanOpen from the game). Login posts
# to the webbridge /aeopd endpoints (same origin through nginx), then the
# paginated register table loads page by page. Family styling matches the
# other DOM panels.

# 1) CSS
sub1(
    "  /* DOM chat shortcuts (Alt+M) */",
    """  /* DOM AEOPD document registry */
  #goro-saraban {
    position: fixed; z-index: 21; display: none; width: 560px; max-width: calc(100vw - 24px);
    box-sizing: border-box; left: 50%; top: 50%; transform: translate(-50%, -50%);
    background: rgba(255, 255, 253, .96); border: 1px solid rgba(120, 110, 90, .55);
    border-radius: 10px; padding: 8px 12px 12px; user-select: none;
    font: 13px/1.4 Sarabun, 'Noto Sans Thai', system-ui, sans-serif; color: #2a2622;
  }
  #goro-saraban .title {
    text-align: center; font-weight: 600; cursor: move; touch-action: none;
    margin: -8px -12px 10px; padding: 6px 10px;
    background: linear-gradient(180deg, #d6e8fa, #b8d6f2);
    border-bottom: 1px solid #76a0ce;
    border-radius: 9px 9px 0 0;
    color: #163658;
    position: relative;
  }
  #goro-saraban .title .x {
    position: absolute; right: 4px; top: 3px; width: 18px; height: 18px;
    border-radius: 4px; background: rgba(232, 223, 200, .7); cursor: pointer;
    text-align: center; line-height: 16px; color: #6a6055; font-size: 11px;
  }
  #goro-saraban .login { display: flex; flex-direction: column; gap: 8px; width: 260px; margin: 10px auto 4px; }
  #goro-saraban .login input {
    padding: 4px 8px; border-radius: 5px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .6);
    color: #2a2622; font: 13px Sarabun, monospace;
  }
  #goro-saraban .login button {
    padding: 4px 14px; border-radius: 5px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .85);
    color: #2a2622; cursor: pointer; font: 13px/1.5 Sarabun, 'Noto Sans Thai', system-ui, sans-serif;
  }
  #goro-saraban .msg { text-align: center; color: #b04030; font-size: 12px; min-height: 16px; margin-top: 6px; }
  #goro-saraban .who { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; color: #6a6055; font-size: 12px; }
  #goro-saraban .who button {
    padding: 1px 10px; border-radius: 4px; font-size: 11px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .8); color: #2a2622; cursor: pointer;
  }
  #goro-saraban .docwrap { overflow-y: auto; max-height: min(46vh, 340px); border: 1px solid #76a0ce; border-radius: 6px; }
  #goro-saraban table { width: 100%; border-collapse: collapse; font-size: 12px; table-layout: fixed; }
  #goro-saraban th { position: sticky; top: 0; background: #deedfc; color: #3a689c; padding: 4px 6px; text-align: left; font-weight: 600; }
  #goro-saraban th.t, #goro-saraban td.t { width: 74px; }
  #goro-saraban th.d, #goro-saraban td.d { width: 76px; }
  #goro-saraban th.f, #goro-saraban td.f { width: 118px; }
  #goro-saraban th.o, #goro-saraban td.o { width: 118px; }
  #goro-saraban td { padding: 3px 6px; border-top: 1px solid rgba(120,110,90,.18); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; background: #fff; color: #1a1a1a; }
  #goro-saraban td.t { background: #deedfc; color: #3a689c; text-align: center; }
  #goro-saraban .pager { display: flex; justify-content: center; align-items: center; gap: 10px; margin-top: 8px; font-size: 12px; color: #6a6055; }
  #goro-saraban .pager button {
    min-width: 26px; padding: 2px 8px; border-radius: 4px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .8); color: #2a2622; cursor: pointer;
  }
  #goro-saraban .pager button:disabled { opacity: .35; cursor: default; }
  #goro-saraban .hint { text-align: center; color: #8a7f6a; font-size: 11px; margin-top: 5px; }
  #goro-saraban .rs {
    position: absolute; right: 0; bottom: 0; width: 16px; height: 16px;
    cursor: nwse-resize; touch-action: none;
    background: linear-gradient(135deg, transparent 46%, rgba(120,110,90,.55) 46%, rgba(120,110,90,.55) 54%, transparent 54%);
    border-radius: 0 0 9px 0;
  }
  /* DOM chat shortcuts (Alt+M) */""",
)

# 2) markup
sub1(
    '<div id="goro-chatshortcuts"></div>',
    '<div id="goro-chatshortcuts"></div>\n  <div id="goro-saraban"></div>',
)

# 3) JS
sub1(
    "  // ---- DOM chat shortcuts (goroChatShortcutsSync) ----",
    """  // ---- DOM AEOPD document registry (NPC portal) ----
  (function () {
    var el = document.getElementById('goro-saraban');
    if (!el) return;
    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }
    var state = { token: '', name: '', page: 1, pages: 1, total: 0, open: false };
    function render() {
      if (!state.open) { el.style.display = 'none'; return; }
      el.style.display = 'block';
      if (!state.token) {
        el.innerHTML = '<div class="title">\\u0e17\\u0e30\\u0e40\\u0e1a\\u0e35\\u0e22\\u0e19\\u0e2b\\u0e19\\u0e31\\u0e07\\u0e2a\\u0e37\\u0e2d (AEOPD)<div class="x">\\u2715</div></div>' +
          '<div class="login">' +
          '<input type="text" id="goro-saraban-user" placeholder="Username" autocomplete="off" spellcheck="false">' +
          '<input type="password" id="goro-saraban-pass" placeholder="Password">' +
          '<button id="goro-saraban-login">Login</button>' +
          '</div><div class="msg" id="goro-saraban-msg"></div>';
        wireLogin();
        wireMove();
        return;
      }
      el.innerHTML = '<div class="title">\\u0e17\\u0e30\\u0e40\\u0e1a\\u0e35\\u0e22\\u0e19\\u0e2b\\u0e19\\u0e31\\u0e07\\u0e2a\\u0e37\\u0e2d (AEOPD)<div class="x">\\u2715</div></div>' +
        '<div class="who"><span>' + esc(state.name) + ' \\u00b7 \\u0e17\\u0e31\\u0e49\\u0e07\\u0e2b\\u0e21\\u0e14 <span id="goro-saraban-count">' + state.total + '</span> \\u0e23\\u0e32\\u0e22\\u0e01\\u0e32\\u0e23</span>' +
        '<button id="goro-saraban-logout">Logout</button></div>' +
        '<div class="docwrap"><table><thead><tr>' +
        '<th class="t">\\u0e1b\\u0e23\\u0e30\\u0e40\\u0e20\\u0e17</th><th class="d">\\u0e25\\u0e07\\u0e27\\u0e31\\u0e19\\u0e17\\u0e35\\u0e48</th>' +
        '<th class="f">\\u0e08\\u0e32\\u0e01</th><th class="o">\\u0e16\\u0e36\\u0e07</th><th>\\u0e40\\u0e23\\u0e37\\u0e48\\u0e2d\\u0e07</th>' +
        '</tr></thead><tbody id="goro-saraban-rows"></tbody></table></div>' +
        '<div class="pager">' +
        '<button id="goro-saraban-prev">\\u2039</button>' +
        '<span id="goro-saraban-page"></span>' +
        '<button id="goro-saraban-next">\\u203a</button>' +
        '</div><div class="hint" id="goro-saraban-hint"></div><div class="rs"></div>';
      el.querySelector('.x').addEventListener('pointerdown', function (ev) { ev.stopPropagation(); closePanel(); });
      wireMove();
      var wrap = el.querySelector('.docwrap');
      if (state.wrapH) wrap.style.maxHeight = state.wrapH;
      keepInView();
      document.getElementById('goro-saraban-logout').addEventListener('pointerdown', function (ev) {
        ev.stopPropagation();
        state.token = ''; state.name = ''; state.page = 1;
        render();
      });
      document.getElementById('goro-saraban-prev').addEventListener('pointerdown', function (ev) { ev.stopPropagation(); loadPage(state.page - 1); });
      document.getElementById('goro-saraban-next').addEventListener('pointerdown', function (ev) { ev.stopPropagation(); loadPage(state.page + 1); });
      loadPage(state.page);
    }
    function wireLogin() {
      el.querySelector('.x').addEventListener('pointerdown', function (ev) { ev.stopPropagation(); closePanel(); });
      var user = document.getElementById('goro-saraban-user');
      var pass = document.getElementById('goro-saraban-pass');
      function submit() {
        var msg = document.getElementById('goro-saraban-msg');
        msg.textContent = '\\u0e01\\u0e33\\u0e25\\u0e31\\u0e07\\u0e15\\u0e23\\u0e27\\u0e08\\u0e2a\\u0e2d\\u0e1a...';
        msg.style.color = '#6a6055';
        fetch('/aeopd/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username: user.value.trim(), password: pass.value })
        }).then(function (r) { return r.json(); }).then(function (d) {
          if (d.token) { state.token = d.token; state.name = d.name; state.page = 1; render(); return; }
          msg.textContent = d.error || 'login failed';
          msg.style.color = '#b04030';
        }).catch(function () {
          msg.textContent = '\\u0e15\\u0e34\\u0e14\\u0e15\\u0e48\\u0e2d\\u0e40\\u0e0b\\u0e2d\\u0e23\\u0e4c\\u0e40\\u0e27\\u0e2d\\u0e23\\u0e4c\\u0e44\\u0e21\\u0e48\\u0e44\\u0e14\\u0e49';
          msg.style.color = '#b04030';
        });
      }
      document.getElementById('goro-saraban-login').addEventListener('pointerdown', function (ev) { ev.stopPropagation(); submit(); });
      [user, pass].forEach(function (input) {
        input.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
        ['keydown', 'keyup'].forEach(function (k) {
          input.addEventListener(k, function (ev) { ev.stopPropagation(); if (ev.key === 'Enter') { ev.preventDefault(); submit(); } });
        });
      });
      setTimeout(function () { user.focus(); }, 0);
    }
    // Drag by the title bar; the bottom-right handle resizes width and
    // list height (kept across re-renders through state.wrapH / el width).
    function wireMove() {
      var title = el.querySelector('.title');
      var drag = null;
      title.addEventListener('pointerdown', function (ev) {
        if (ev.target.closest('.x')) return;
        var r = el.getBoundingClientRect();
        el.style.transform = 'none';
        el.style.left = r.left + 'px';
        el.style.top = r.top + 'px';
        drag = { x: ev.clientX - r.left, y: ev.clientY - r.top };
        try { title.setPointerCapture(ev.pointerId); } catch (e) {}
        ev.preventDefault();
      });
      title.addEventListener('pointermove', function (ev) {
        if (!drag) return;
        el.style.left = Math.max(0, Math.min(ev.clientX - drag.x, window.innerWidth - el.offsetWidth)) + 'px';
        el.style.top = Math.max(0, Math.min(ev.clientY - drag.y, window.innerHeight - el.offsetHeight)) + 'px';
      });
      title.addEventListener('pointerup', function () { drag = null; });
      title.addEventListener('pointercancel', function () { drag = null; });
      var handle = el.querySelector('.rs');
      if (!handle) return;
      var wrap = el.querySelector('.docwrap');
      var rs = null;
      handle.addEventListener('pointerdown', function (ev) {
        ev.stopPropagation();
        rs = { x: ev.clientX, y: ev.clientY, w: el.offsetWidth, h: wrap.offsetHeight };
        try { handle.setPointerCapture(ev.pointerId); } catch (e) {}
        ev.preventDefault();
      });
      handle.addEventListener('pointermove', function (ev) {
        if (!rs) return;
        el.style.width = Math.max(420, Math.min(rs.w + ev.clientX - rs.x, window.innerWidth - 24)) + 'px';
        var h = Math.max(140, Math.min(rs.h + ev.clientY - rs.y, Math.round(window.innerHeight * 0.7)));
        wrap.style.maxHeight = h + 'px';
        state.wrapH = h + 'px';
      });
      handle.addEventListener('pointerup', function () { rs = null; });
      handle.addEventListener('pointercancel', function () { rs = null; });
    }
    // After a drag the panel keeps explicit left/top; re-clamp it into the
    // viewport whenever the content height changes (render + row loads).
    function keepInView() {
      if (el.style.transform !== 'none') return;
      el.style.left = Math.max(0, Math.min(parseFloat(el.style.left) || 0, window.innerWidth - el.offsetWidth)) + 'px';
      el.style.top = Math.max(0, Math.min(parseFloat(el.style.top) || 0, window.innerHeight - el.offsetHeight)) + 'px';
    }
    function loadPage(page) {
      var rowsEl = document.getElementById('goro-saraban-rows');
      var hint = document.getElementById('goro-saraban-hint');
      var prev = document.getElementById('goro-saraban-prev');
      var next = document.getElementById('goro-saraban-next');
      prev.disabled = next.disabled = true;
      hint.textContent = '\\u0e01\\u0e33\\u0e25\\u0e31\\u0e07\\u0e42\\u0e2b\\u0e25\\u0e14...';
      fetch('/aeopd/docs?token=' + encodeURIComponent(state.token) + '&page=' + page)
        .then(function (r) { return r.json().then(function (d) { return { ok: r.ok, d: d }; }); })
        .then(function (res) {
          if (!res.ok) { state.token = ''; render(); return; }
          var d = res.d;
          state.page = d.page; state.pages = d.pages; state.total = d.total;
          var cnt = document.getElementById('goro-saraban-count');
          if (cnt) cnt.textContent = d.total;
          var html = '';
          (d.rows || []).forEach(function (row) {
            html += '<tr><td class="t">' + esc(row.label) + '</td><td class="d">' + esc(row.date) + '</td>' +
              '<td class="f" title="' + esc(row.from) + '">' + esc(row.from) + '</td>' +
              '<td class="o" title="' + esc(row.to) + '">' + esc(row.to) + '</td>' +
              '<td title="' + esc(row.topic) + '">' + esc(row.topic) + '</td></tr>';
          });
          rowsEl.innerHTML = html || '';
          keepInView();
          document.getElementById('goro-saraban-page').textContent = d.page + ' / ' + d.pages;
          prev.disabled = d.page <= 1;
          next.disabled = d.page >= d.pages;
          hint.textContent = '';
        })
        .catch(function () { hint.textContent = '\\u0e42\\u0e2b\\u0e25\\u0e14\\u0e44\\u0e21\\u0e48\\u0e2a\\u0e33\\u0e40\\u0e23\\u0e47\\u0e08'; });
    }
    function closePanel() { state.open = false; render(); }
    window.goroSarabanOpen = function () {
      state.open = true;
      render();
    };
  })();
  // ---- DOM chat shortcuts (goroChatShortcutsSync) ----""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
