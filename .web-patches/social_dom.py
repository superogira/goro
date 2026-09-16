import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# DOM social windows: confirm modal, friends (party/friend tabs), friend
# setup, party settings and 1:1 whisper chats. Family styling.

# 1) CSS
sub1(
    "  /* DOM text prompt */",
    """  /* DOM social windows */
  .goro-social {
    position: fixed; z-index: 22; display: none; width: 300px; max-width: calc(100vw - 24px);
    box-sizing: border-box;
    background: rgba(255, 255, 253, .97); border: 1px solid rgba(120, 110, 90, .55);
    border-radius: 10px; padding: 8px 12px 12px; user-select: none;
    font: 13px/1.4 Sarabun, 'Noto Sans Thai', system-ui, sans-serif; color: #2a2622;
  }
  .goro-social .title {
    text-align: center; font-weight: 600;
    margin: -8px -12px 10px; padding: 6px 10px;
    background: linear-gradient(180deg, #d6e8fa, #b8d6f2);
    border-bottom: 1px solid #76a0ce;
    border-radius: 9px 9px 0 0;
    color: #163658;
    position: relative;
    cursor: move; touch-action: none;
  }
  .goro-social .x {
    position: absolute; right: 4px; top: 3px; width: 18px; height: 18px;
    border-radius: 4px; background: rgba(232, 223, 200, .7); cursor: pointer;
    text-align: center; line-height: 16px; color: #6a6055; font-size: 11px;
  }
  .goro-social .msg { text-align: center; color: #2a2622; font-size: 13px; margin: 8px 4px 4px; white-space: pre-line; }
  .goro-social .btns { display: flex; justify-content: center; gap: 10px; margin-top: 12px; }
  .goro-social .btns button, .goro-social .rowbtn {
    min-width: 64px; padding: 3px 12px; border-radius: 5px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .85);
    color: #2a2622; cursor: pointer; font: 12px/1.6 Sarabun, 'Noto Sans Thai', system-ui, sans-serif;
  }
  .goro-social label { display: flex; align-items: center; gap: 8px; margin: 6px 2px; cursor: pointer; }
  .goro-social .tabs { display: flex; gap: 2px; margin: -2px 0 8px; }
  .goro-social .tabs button {
    flex: 0 0 auto; min-width: 80px; padding: 3px 12px; border-radius: 6px 6px 0 0;
    border: 1px solid #76a0ce; background: rgba(222, 237, 252, .55); color: #3a689c; cursor: pointer;
    font: 12px/1.6 Sarabun, 'Noto Sans Thai', system-ui, sans-serif;
  }
  .goro-social .tabs button.active { background: #deedfc; font-weight: 600; }
  .goro-social .list { border: 1px solid rgba(120,110,90,.35); border-radius: 6px; overflow-y: auto; max-height: 240px; background: #fff; }
  .goro-social .frow { display: flex; align-items: center; gap: 6px; padding: 3px 8px; border-bottom: 1px solid rgba(120,110,90,.12); }
  .goro-social .frow:nth-child(even) { background: rgba(222, 237, 252, .35); }
  .goro-social .frow .nm { flex: 1 1 auto; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; cursor: pointer; }
  .goro-social .frow .st { flex: 0 0 auto; font-size: 11px; color: #6a6055; }
  .goro-social .frow .st.on { color: #1d7a3e; }
  .goro-social .rowbtn { min-width: 22px; padding: 0 6px; border-radius: 4px; }
  .goro-social .hpbar { flex: 0 0 70px; height: 5px; border: 1px solid rgba(120,110,90,.4); border-radius: 2px; background: #e0e8f2; overflow: hidden; }
  .goro-social .hpbar i { display: block; height: 100%; background: #34a853; }
  .goro-social .foot { display: flex; justify-content: flex-end; gap: 8px; margin-top: 10px; }
  .goro-social .menu { display: none; position: absolute; right: 6px; z-index: 5; min-width: 120px;
    background: rgba(255,255,253,.98); border: 1px solid rgba(120,110,90,.5); border-radius: 6px; padding: 4px; }
  .goro-social .menu.open { display: block; }
  .goro-social .menu button { display: block; width: 100%; text-align: left; background: none; border: 0;
    padding: 4px 8px; border-radius: 4px; cursor: pointer; color: #2a2622;
    font: 12px/1.5 Sarabun, 'Noto Sans Thai', system-ui, sans-serif; }
  .goro-social .menu button:hover { background: rgba(222, 237, 252, .8); }
  .goro-social .chatlog { height: 130px; overflow-y: auto; background: rgba(14,18,24,.82);
    border: 1px solid rgba(180,198,218,.4); border-radius: 6px; padding: 6px 8px; font-size: 12px; }
  .goro-social .chatlog div { color: #b5deef; }
  .goro-social .chatlog div.out { color: #ffff78; }
  .goro-social .chatlog div.err { color: #e08070; }
  .goro-social .chatrow { display: flex; gap: 6px; margin-top: 8px; }
  .goro-social .chatrow input { flex: 1 1 auto; padding: 4px 8px; border-radius: 5px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .6); color: #2a2622; font: 12px Sarabun, monospace; }
  /* DOM text prompt */""",
)

# 2) markup: five panels + a whisper container
sub1(
    '  <div id="goro-textprompt"></div>',
    '''  <div id="goro-textprompt"></div>
  <div id="goro-confirm" class="goro-social" style="left:50%;top:38%;transform:translate(-50%,-50%)"></div>
  <div id="goro-friends" class="goro-social" style="right:18px;top:56px"></div>
  <div id="goro-friendsetup" class="goro-social" style="left:50%;top:40%;transform:translate(-50%,-50%)"></div>
  <div id="goro-partysetup" class="goro-social" style="left:50%;top:40%;transform:translate(-50%,-50%)"></div>
  <div id="goro-whisper" style="position:fixed;left:0;bottom:0;z-index:22;pointer-events:none"></div>''',
)

# 3) JS
sub1(
    "  // ---- DOM text prompt (goroTextPromptSync) ----",
    """  // ---- DOM social windows ----
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

    // -- confirm modal --
    var cfm = document.getElementById('goro-confirm');
    var cfmState = { open: false, title: '', message: '', okOnly: false };
    function cfmAct(a) { if (window.goroConfirmAction) window.goroConfirmAction(a); }
    function cfmRender() {
      if (!cfmState.open) { cfm.style.display = 'none'; cfm.innerHTML = ''; return; }
      cfm.style.display = 'block';
      cfm.innerHTML = '<div class="title">' + esc(cfmState.title) + '</div>' +
        '<div class="msg">' + esc(cfmState.message) + '</div>' +
        '<div class="btns"><button class="ok">OK</button>' + (cfmState.okOnly ? '' : '<button class="no">Cancel</button>') + '</div>';
      stop(cfm);
      cfm.querySelector('.ok').addEventListener('pointerdown', function () { cfmState.open = false; cfmRender(); cfmAct('ok'); });
      var no = cfm.querySelector('.no');
      if (no) no.addEventListener('pointerdown', function () { cfmState.open = false; cfmRender(); cfmAct('cancel'); });
    }
    window.goroConfirmSync = function (d) {
      cfmState.open = !!d.open; cfmState.title = d.title || ''; cfmState.message = d.message || ''; cfmState.okOnly = !!d.okOnly;
      cfmRender();
    };

    // -- friends window --
    var fw = document.getElementById('goro-friends');
    var fwState = { open: false, tab: 'friends', title: 'Friends', friends: [], party: { active: false, name: '', members: [] }, canManage: false };
    var fwMenuAid = null;
    function fwAct(a) { if (window.goroFriendsAction) window.goroFriendsAction(a); }
    function fwRender() {
      if (!fwState.open) { fw.style.display = 'none'; fw.innerHTML = ''; return; }
      fw.style.display = 'block';
      var html = '<div class="title">' + esc(fwState.title) + '<div class="x">\\u2715</div></div>' +
        '<div class="tabs">' +
        '<button data-tab="friends" class="' + (fwState.tab === 'friends' ? 'active' : '') + '">Friends</button>' +
        '<button data-tab="party" class="' + (fwState.tab === 'party' ? 'active' : '') + '">Party</button></div>';
      if (fwState.tab === 'friends') {
        html += '<div class="list">';
        if (!fwState.friends.length) {
          html += '<div class="frow"><span class="nm" style="color:#8a7f6a">No friends</span></div>';
        } else {
          fwState.friends.forEach(function (f) {
            html += '<div class="frow" data-aid="' + esc(f.aid) + '">' +
              '<span class="nm" data-do="whisper">' + esc(f.name) + '</span>' +
              '<span class="st ' + (f.online ? 'on' : '') + '">' + (f.online ? 'Online' : 'Offline') + '</span>' +
              '<button class="rowbtn" data-menu="1">\\u22ef</button></div>';
          });
        }
        html += '</div><div class="foot"><button data-do="fsetup">Setup</button></div>';
      } else {
        html += '<div class="list">';
        if (!fwState.party.active || !fwState.party.members.length) {
          html += '<div class="frow"><span class="nm" style="color:#8a7f6a">' + (fwState.party.active ? 'No party' : 'No party') + '</span></div>';
        } else {
          fwState.party.members.forEach(function (m) {
            var hp = m.maxHp > 0 ? Math.round(100 * m.hp / m.maxHp) : 0;
            html += '<div class="frow" data-aid="' + esc(m.aid) + '">' +
              '<span class="nm" data-pm="whisper">' + esc(m.name) + (m.leader ? ' *' : '') + '</span>' +
              '<span class="st ' + (m.online ? 'on' : '') + '">' + (m.online ? (esc(m.map) || 'Online') : 'Offline') + '</span>' +
              '<span class="hpbar"><i style="width:' + hp + '%"></i></span>' +
              '<button class="rowbtn" data-pmenu="1">\\u22ef</button></div>';
          });
        }
        html += '</div><div class="foot">';
        if (!fwState.party.active) {
          html += '<button data-do="pcreate">Create</button>';
        } else {
          if (fwState.canManage) html += '<button data-do="pinvite">Invite</button>';
          html += '<button data-do="psettings">Settings</button><button data-do="pleave">Leave</button>';
        }
        html += '</div>';
      }
      html += '<div class="menu" id="goro-fwmenu"></div>';
      fw.innerHTML = html;
      stop(fw);
      wireDrag(fw, fw.querySelector('.title'));
      fw.querySelector('.x').addEventListener('pointerdown', function () { fwState.open = false; fwRender(); fwAct('close'); });
      fw.querySelectorAll('.tabs button').forEach(function (b) {
        b.addEventListener('pointerdown', function () { fwAct('tab:' + b.getAttribute('data-tab')); });
      });
      fw.querySelectorAll('.foot button').forEach(function (b) {
        b.addEventListener('pointerdown', function () { fwAct(b.getAttribute('data-do')); });
      });
      fw.querySelectorAll('.frow .nm[data-do="whisper"]').forEach(function (el) {
        el.addEventListener('pointerdown', function () { fwAct('fwhisper:' + el.closest('.frow').getAttribute('data-aid')); });
      });
      fw.querySelectorAll('.frow .nm[data-pm="whisper"]').forEach(function (el) {
        el.addEventListener('pointerdown', function () { fwAct('pwhisper:' + el.closest('.frow').getAttribute('data-aid')); });
      });
      var menu = fw.querySelector('#goro-fwmenu');
      function closeMenu() { menu.classList.remove('open'); fwMenuAid = null; }
      fw.querySelectorAll('.rowbtn[data-menu="1"]').forEach(function (b) {
        b.addEventListener('pointerdown', function (ev) {
          ev.stopPropagation();
          var aid = b.closest('.frow').getAttribute('data-aid');
          fwMenuAid = fwMenuAid === aid ? null : aid;
          if (fwMenuAid) {
            menu.innerHTML = '<button data-a="fwhisper">1:1 Chat</button><button data-a="fdelete">Delete Friend</button><button data-a="fblock">Block Whisper</button>';
            menu.classList.add('open');
            menu.querySelectorAll('button').forEach(function (mb) {
              mb.addEventListener('pointerdown', function () { fwAct(mb.getAttribute('data-a') + ':' + aid); closeMenu(); });
            });
          } else closeMenu();
        });
      });
      fw.querySelectorAll('.rowbtn[data-pmenu="1"]').forEach(function (b) {
        b.addEventListener('pointerdown', function (ev) {
          ev.stopPropagation();
          var aid = b.closest('.frow').getAttribute('data-aid');
          fwMenuAid = fwMenuAid === aid ? null : aid;
          if (fwMenuAid) {
            menu.innerHTML = '<button data-a="pinfo">Information</button><button data-a="pwhisper">1:1 Chat</button><button data-a="pexpel">Expel</button>';
            menu.classList.add('open');
            menu.querySelectorAll('button').forEach(function (mb) {
              mb.addEventListener('pointerdown', function () { fwAct(mb.getAttribute('data-a') + ':' + aid); closeMenu(); });
            });
          } else closeMenu();
        });
      });
      fw.addEventListener('pointerdown', function (ev) {
        if (fwMenuAid && !ev.target.closest('.menu')) closeMenu();
      }, true);
    }
    window.goroFriendsSync = function (d) {
      fwState.open = !!d.open; fwState.tab = d.tab || 'friends'; fwState.title = d.title || 'Friends';
      fwState.friends = d.friends || []; fwState.party = d.party || { active: false, name: '', members: [] };
      fwState.canManage = !!d.canManage;
      fwRender();
    };

    // -- friend setup --
    var fs = document.getElementById('goro-friendsetup');
    var fsState = { open: false, strangers: false, friends: false, alert: false };
    function fsAct(a) { if (window.goroFriendSetupAction) window.goroFriendSetupAction(a); }
    function fsRender() {
      if (!fsState.open) { fs.style.display = 'none'; fs.innerHTML = ''; return; }
      fs.style.display = 'block';
      fs.innerHTML = '<div class="title">Friend Setup<div class="x">\\u2715</div></div>' +
        '<label><input type="checkbox" data-k="strangers"' + (fsState.strangers ? ' checked' : '') + '> 1:1 Chat from Strangers</label>' +
        '<label><input type="checkbox" data-k="friends"' + (fsState.friends ? ' checked' : '') + '> 1:1 Chat from Friends</label>' +
        '<label><input type="checkbox" data-k="alert"' + (fsState.alert ? ' checked' : '') + '> 1:1 Chat Alert</label>' +
        '<div class="btns"><button class="ok">OK</button><button class="no">Cancel</button></div>';
      stop(fs);
      fs.querySelector('.x').addEventListener('pointerdown', function () { fsState.open = false; fsRender(); fsAct('cancel'); });
      fs.querySelectorAll('input').forEach(function (i) {
        i.addEventListener('change', function () { fsState[i.getAttribute('data-k')] = i.checked; });
        i.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
      });
      fs.querySelector('.ok').addEventListener('pointerdown', function () {
        fsAct('ok:' + (fsState.strangers ? 1 : 0) + ':' + (fsState.friends ? 1 : 0) + ':' + (fsState.alert ? 1 : 0));
        fsState.open = false; fsRender();
      });
      fs.querySelector('.no').addEventListener('pointerdown', function () { fsState.open = false; fsRender(); fsAct('cancel'); });
    }
    window.goroFriendSetupSync = function (d) {
      fsState.open = !!d.open; fsState.strangers = !!d.strangers; fsState.friends = !!d.friends; fsState.alert = !!d.alert;
      fsRender();
    };

    // -- party settings --
    var ps = document.getElementById('goro-partysetup');
    var psState = { open: false, expShare: 0, refuseInvites: false };
    function psAct(a) { if (window.goroPartySetupAction) window.goroPartySetupAction(a); }
    function psRender() {
      if (!psState.open) { ps.style.display = 'none'; ps.innerHTML = ''; return; }
      ps.style.display = 'block';
      ps.innerHTML = '<div class="title">Party Settings<div class="x">\\u2715</div></div>' +
        '<div style="margin:4px 2px;color:#3a689c;font-size:12px">EXP</div>' +
        '<label><input type="radio" name="ps-exp" value="0"' + (psState.expShare === 0 ? ' checked' : '') + '> Each Take</label>' +
        '<label><input type="radio" name="ps-exp" value="1"' + (psState.expShare === 1 ? ' checked' : '') + '> Even Share</label>' +
        '<label><input type="checkbox" data-k="refuseInvites"' + (psState.refuseInvites ? ' checked' : '') + '> Refuse party invites</label>' +
        '<div class="btns"><button class="ok">OK</button><button class="no">Cancel</button></div>';
      stop(ps);
      ps.querySelector('.x').addEventListener('pointerdown', function () { psState.open = false; psRender(); psAct('cancel'); });
      ps.querySelectorAll('input[name="ps-exp"]').forEach(function (i) {
        i.addEventListener('change', function () { if (i.checked) psState.expShare = parseInt(i.value, 10) || 0; });
        i.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
      });
      var cb = ps.querySelector('input[data-k="refuseInvites"]');
      cb.addEventListener('change', function () { psState.refuseInvites = cb.checked; });
      cb.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
      ps.querySelector('.ok').addEventListener('pointerdown', function () {
        psAct('ok:' + psState.expShare + ':' + (psState.refuseInvites ? 1 : 0));
        psState.open = false; psRender();
      });
      ps.querySelector('.no').addEventListener('pointerdown', function () { psState.open = false; psRender(); psAct('cancel'); });
    }
    window.goroPartySetupSync = function (d) {
      psState.open = !!d.open; psState.expShare = d.expShare | 0; psState.refuseInvites = !!d.refuseInvites;
      psRender();
    };

    // -- whisper windows --
    var whRoot = document.getElementById('goro-whisper');
    var whState = { windows: [], drafts: {}, pos: {} };
    function whAct(a) { if (window.goroWhisperAction) window.goroWhisperAction(a); }
    function whRender() {
      whRoot.innerHTML = '';
      whState.windows.forEach(function (win, idx) {
        var panel = document.createElement('div');
        panel.className = 'goro-social';
        var pos = whState.pos[win.target];
        panel.style.display = 'block';
        panel.style.pointerEvents = 'auto';
        panel.style.right = 'auto';
        panel.style.left = (pos && pos.x != null ? pos.x : Math.min(24 + idx * 34, window.innerWidth - 320)) + 'px';
        panel.style.bottom = 'auto';
        panel.style.top = (pos && pos.y != null ? pos.y : window.innerHeight - 300 - idx * 26) + 'px';
        var lines = win.lines.length ? win.lines : [{ text: 'No messages', kind: '' }];
        panel.innerHTML = '<div class="title">1:1 ' + esc(win.target) + '<div class="x">\\u2715</div></div>' +
          '<div class="chatlog">' + lines.map(function (l) {
            return '<div class="' + esc(l.kind) + '">' + esc(l.text) + '</div>';
          }).join('') + '</div>' +
          '<div class="chatrow"><input type="text" maxlength="100" placeholder="Message"><button>Send</button></div>';
        whRoot.appendChild(panel);
        stop(panel);
        wireDrag(panel, panel.querySelector('.title'));
        var log = panel.querySelector('.chatlog');
        log.scrollTop = log.scrollHeight;
        var input = panel.querySelector('input');
        input.value = whState.drafts[win.target] || '';
        function send() {
          var text = input.value.trim();
          if (!text) return;
          delete whState.drafts[win.target];
          input.value = '';
          whAct('send:' + win.target + ':' + text);
        }
        ['keydown', 'keyup', 'keypress'].forEach(function (k) {
          input.addEventListener(k, function (ev) {
            ev.stopPropagation();
            if (ev.key === 'Enter') { ev.preventDefault(); send(); }
          });
        });
        input.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
        input.addEventListener('input', function () { whState.drafts[win.target] = input.value; });
        panel.querySelector('.chatrow button').addEventListener('pointerdown', function () { send(); });
        panel.querySelector('.x').addEventListener('pointerdown', function () { whAct('close:' + win.target); });
        // remember drag position between syncs
        var title = panel.querySelector('.title');
        title.addEventListener('pointerup', function () {
          var r = panel.getBoundingClientRect();
          whState.pos[win.target] = { x: r.left, y: r.top };
        });
      });
    }
    window.goroWhisperSync = function (windows) {
      whState.windows = windows || [];
      whRender();
    };
  })();
  // ---- DOM text prompt (goroTextPromptSync) ----""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("social panels patched, %d -> %d bytes" % (len(orig), len(src)))
