import io, sys

# DOM chat room windows: the "Make a Room" creation form and the in-room
# chat window. Family (.goro-social) styling; actions flow back through
# goroChatRoomCreateAction / goroChatRoomAction. Idempotent.

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

if "goroChatRoomCreateSync" in src:
    print("chat room panels already patched")
    sys.exit(0)

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# 1) CSS (appended at the end of the style block so it wins the cascade)
sub1(
    "</style>",
    """  /* DOM chat room windows */
  .goro-social .frm { display: flex; align-items: center; gap: 8px; margin: 6px 2px; }
  .goro-social .frm .lb { flex: 0 0 auto; min-width: 56px; font-size: 12px; color: #3a689c; }
  .goro-social .finput {
    flex: 1 1 auto; min-width: 0; padding: 4px 8px; border-radius: 5px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .6);
    color: #2a2622; font: 12px Sarabun, monospace;
  }
  .goro-social .frm label { margin: 0; }
  .goro-social .memline {
    margin: 0 0 6px; font-size: 12px; color: #3a689c;
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  .goro-social .chatlog div.sys { color: #a0bee6; }
</style>""",
)

# 2) markup
sub1(
    '  <div id="goro-whisper" style="position:fixed;left:0;bottom:0;z-index:22;pointer-events:none"></div>',
    '''  <div id="goro-whisper" style="position:fixed;left:0;bottom:0;z-index:22;pointer-events:none"></div>
  <div id="goro-chatroom-create" class="goro-social" style="left:50%;top:40%;transform:translate(-50%,-50%)"></div>
  <div id="goro-chatroom" class="goro-social" style="left:24px;top:120px"></div>''',
)

# 3) JS
sub1(
    "  // ---- DOM text prompt (goroTextPromptSync) ----",
    """  // ---- DOM chat room windows ----
  (function () {
    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }
    function stop(el) {
      ['pointerdown', 'pointerup', 'click', 'wheel'].forEach(function (k) {
        el.addEventListener(k, function (ev) { ev.stopPropagation(); });
      });
    }
    function wireDrag(panel, title) {
      var drag = null;
      title.addEventListener('pointerdown', function (ev) {
        if (ev.target.closest('.x')) return;
        var r = panel.getBoundingClientRect();
        if (panel.style.transform) { panel.style.transform = 'none'; panel.style.left = r.left + 'px'; panel.style.top = r.top + 'px'; }
        drag = { x: ev.clientX - r.left, y: ev.clientY - r.top };
        try { title.setPointerCapture(ev.pointerId); } catch (e) {}
        ev.preventDefault();
      });
      title.addEventListener('pointermove', function (ev) {
        if (!drag) return;
        panel.style.left = Math.max(0, Math.min(ev.clientX - drag.x, window.innerWidth - panel.offsetWidth)) + 'px';
        panel.style.top = Math.max(0, Math.min(ev.clientY - drag.y, window.innerHeight - panel.offsetHeight)) + 'px';
      });
      title.addEventListener('pointerup', function () { drag = null; });
      title.addEventListener('pointercancel', function () { drag = null; });
    }
    var US = '\\u001f';

    // -- make a room (#goro-chatroom-create): built fresh on every open so
    //    the form resets; field values never round-trip through Go --
    var cc = document.getElementById('goro-chatroom-create');
    var ccOpen = false;
    function ccAct(a) { if (window.goroChatRoomCreateAction) window.goroChatRoomCreateAction(a); }
    function ccBuild() {
      cc.innerHTML = '<div class="title">Make a Room<div class="x">\\u2715</div></div>' +
        '<div class="frm"><span class="lb">Title</span><input class="finput" data-k="title" maxlength="32" placeholder="Room Title"></div>' +
        '<div class="frm"><span class="lb">Limit</span><input class="finput" data-k="limit" maxlength="2" inputmode="numeric" style="flex:0 0 64px">' +
        '<span class="lb" style="min-width:0;margin-left:8px">Type</span>' +
        '<label><input type="radio" name="cc-type" value="public" checked> Public</label>' +
        '<label><input type="radio" name="cc-type" value="private"> Private</label></div>' +
        '<div class="frm"><span class="lb">Password</span><input class="finput" data-k="password" maxlength="8" placeholder="Optional"></div>' +
        '<div class="btns"><button class="ok">OK</button><button class="no">Cancel</button></div>';
      stop(cc);
      wireDrag(cc, cc.querySelector('.title'));
      cc.querySelector('[data-k="limit"]').value = '20';
      function submit() {
        var title = cc.querySelector('[data-k="title"]').value.trim();
        if (!title) { cc.querySelector('[data-k="title"]').focus(); return; }
        var pass = cc.querySelector('[data-k="password"]').value.trim();
        var limit = parseInt(cc.querySelector('[data-k="limit"]').value, 10) || 20;
        var pub = !cc.querySelector('input[name="cc-type"][value="private"]').checked;
        ccAct('ok' + US + title + US + pass + US + limit + US + (pub ? 'public' : 'private'));
      }
      cc.querySelectorAll('input.finput').forEach(function (i) {
        ['keydown', 'keyup', 'keypress'].forEach(function (k) {
          i.addEventListener(k, function (ev) {
            ev.stopPropagation();
            if (ev.key === 'Enter') { ev.preventDefault(); submit(); }
            if (ev.key === 'Escape') { ev.preventDefault(); ccAct('cancel'); }
          });
        });
        i.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
      });
      cc.querySelectorAll('input[type="radio"]').forEach(function (i) {
        i.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
      });
      cc.querySelector('.ok').addEventListener('pointerdown', function () { submit(); });
      cc.querySelector('.no').addEventListener('pointerdown', function () { ccAct('cancel'); });
      cc.querySelector('.x').addEventListener('pointerdown', function () { ccAct('cancel'); });
      cc.querySelector('[data-k="title"]').focus();
    }
    window.goroChatRoomCreateSync = function (d) {
      var open = !!d.open;
      if (open && !ccOpen) { ccBuild(); cc.style.display = 'block'; }
      else if (!open && ccOpen) { cc.style.display = 'none'; cc.innerHTML = ''; }
      ccOpen = open;
    };

    // -- chat room (#goro-chatroom): built once per room; only the member
    //    line and log update across syncs, so drags and input focus
    //    survive --
    var cr = document.getElementById('goro-chatroom');
    var crOpen = false, crSig = '', crDraft = '';
    function crAct(a) { if (window.goroChatRoomAction) window.goroChatRoomAction(a); }
    function crSigOf(d) {
      var last = d.lines.length ? d.lines[d.lines.length - 1].text : '';
      return [d.title, d.public ? 1 : 0, d.limit, d.count, d.owner, (d.members || []).join(','), d.lines.length, last].join('|');
    }
    function crBuild(d) {
      cr.innerHTML = '<div class="title"><span class="t"></span><div class="x">\\u2715</div></div>' +
        '<div class="memline"></div>' +
        '<div class="chatlog"></div>' +
        '<div class="chatrow"><input type="text" maxlength="100" placeholder="Message"><button>Send</button><button data-do="leave">Leave</button></div>';
      stop(cr);
      wireDrag(cr, cr.querySelector('.title'));
      var input = cr.querySelector('.chatrow input');
      input.value = crDraft;
      function send() {
        var text = input.value.trim();
        if (!text) return;
        crDraft = ''; input.value = '';
        crAct('send:' + text);
      }
      ['keydown', 'keyup', 'keypress'].forEach(function (k) {
        input.addEventListener(k, function (ev) {
          ev.stopPropagation();
          if (ev.key === 'Enter') { ev.preventDefault(); send(); }
        });
      });
      input.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
      input.addEventListener('input', function () { crDraft = input.value; });
      cr.querySelector('.chatrow button').addEventListener('pointerdown', function (ev) { ev.stopPropagation(); send(); });
      cr.querySelector('[data-do="leave"]').addEventListener('pointerdown', function (ev) { ev.stopPropagation(); crAct('leave'); });
      cr.querySelector('.x').addEventListener('pointerdown', function (ev) { ev.stopPropagation(); crAct('leave'); });
    }
    function crUpdate(d) {
      var count = d.count || (d.members || []).length;
      var limit = d.limit || count;
      var names = (d.members || []).map(function (m) { return m === d.owner ? m + ' (owner)' : m; }).join(', ') || 'No members';
      cr.querySelector('.title .t').textContent = d.title + ' - ' + (d.public ? 'Public' : 'Private');
      cr.querySelector('.memline').textContent = count + '/' + limit + '  ' + names;
      var log = cr.querySelector('.chatlog');
      var lines = d.lines.length ? d.lines : [{ text: 'No messages', kind: '' }];
      log.innerHTML = lines.map(function (l) {
        return '<div class="' + esc(l.kind) + '">' + esc(l.text) + '</div>';
      }).join('');
      log.scrollTop = log.scrollHeight;
    }
    window.goroChatRoomSync = function (d) {
      if (!d.open) {
        if (crOpen) { cr.style.display = 'none'; cr.innerHTML = ''; crOpen = false; crSig = ''; }
        return;
      }
      var sig = crSigOf(d);
      if (!crOpen) {
        crBuild(d);
        crUpdate(d);
        cr.style.display = 'block';
        crOpen = true; crSig = sig;
        return;
      }
      if (sig !== crSig) { crSig = sig; crUpdate(d); }
    };
  })();
  // ---- DOM text prompt (goroTextPromptSync) ----""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("chat room panels patched, %d -> %d bytes" % (len(orig), len(src)))
