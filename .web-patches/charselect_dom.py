import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# DOM character select layer: slot row with preview sprites, stats panel,
# footer buttons and the delete confirm/email modals — the DOM twin of the
# canvas CharacterSelectWindow. The game keeps ownership of selection,
# previews and packets (goroCharSelectSync / goroCharSelectAction).

# 1) CSS — the panel family look (same as the DOM inventory), plus the
#    cursor fix: the sprite cursor must sit above every DOM layer, not
#    under the windows it was drawn beneath since z-index 1.
sub1(
    "  #goro-cursor {\n    position: fixed; left: 0; top: 0; z-index: 1; display: none;",
    "  #goro-cursor {\n    position: fixed; left: 0; top: 0; z-index: 9999; display: none;",
)
sub1(
    "  /* DOM loading label + status window */",
    """  /* DOM character select */
  #goro-charselect {
    position: fixed; inset: 0; z-index: 3; display: none; overflow: hidden;
    font: 13px/1.4 Sarabun, 'Noto Sans Thai', system-ui, sans-serif; color: #2a2622;
  }
  #goro-charselect .bg { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: cover; }
  #goro-charselect .panel {
    position: absolute; left: 50%; top: 50%; transform: translate(-50%, -50%);
    width: 580px; max-width: calc(100vw - 24px); box-sizing: border-box;
    background: rgba(255, 255, 253, .96); border: 1px solid rgba(120, 110, 90, .55);
    border-radius: 10px; padding: 8px 14px 12px; user-select: none;
  }
  #goro-charselect .title { text-align: center; font-weight: 600; margin-bottom: 8px; }
  #goro-charselect .slotsrow { display: flex; align-items: center; justify-content: center; gap: 13px; }
  #goro-charselect .slots { display: flex; gap: 25px; }
  #goro-charselect .slot {
    width: 139px; height: 144px; box-sizing: border-box; border-radius: 6px;
    border: 1px solid rgba(120, 110, 90, .55); background: rgba(255, 255, 255, .5);
    display: flex; align-items: center; justify-content: center;
    color: #8a7f6a; font-size: 14px; cursor: pointer; padding: 0;
  }
  #goro-charselect .slot img { max-width: 127px; max-height: 132px; image-rendering: pixelated; }
  #goro-charselect .slot.sel { background: #deedfc; border-color: #a6bede; box-shadow: 0 0 0 1px #a6bede; }
  #goro-charselect .arrow {
    width: 30px; height: 60px; border-radius: 6px; font-size: 22px; line-height: 1;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .8);
    color: #2a2622; cursor: pointer;
  }
  #goro-charselect .pageinfo { text-align: center; color: #6a6055; margin: 6px 0; }
  #goro-charselect .info {
    margin: 0 auto; width: 318px; max-width: 100%; box-sizing: border-box;
    border: 1px solid rgba(120, 110, 90, .45); border-radius: 6px; overflow: hidden;
    background: rgba(255,255,255,.35);
  }
  #goro-charselect .info table { width: 100%; border-collapse: collapse; font-size: 12px; }
  #goro-charselect .info td { padding: 1px 6px; }
  #goro-charselect .info td.h { color: #6a6055; width: 44px; }
  #goro-charselect .info td.s { color: #6a6055; width: 32px; }
  #goro-charselect .info td.v { text-align: right; }
  #goro-charselect .info .empty { text-align: center; padding: 18px 8px; color: #8a7f6a; }
  #goro-charselect .foot { display: flex; justify-content: center; gap: 8px; margin-top: 10px; }
  #goro-charselect .foot button, #goro-charselect .delmodal button {
    min-width: 64px; padding: 3px 14px; border-radius: 5px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .8);
    color: #2a2622; cursor: pointer; font: 13px/1.6 Sarabun, 'Noto Sans Thai', system-ui, sans-serif;
  }
  #goro-charselect .foot button:disabled { opacity: .45; cursor: default; }
  #goro-charselect .status { text-align: center; color: #6a6055; font-size: 12px; min-height: 16px; margin-top: 6px; }
  #goro-charselect .status.err { color: #b04030; }
  #goro-charselect .delmodal {
    position: absolute; left: 50%; top: 44%; transform: translate(-50%, -50%);
    background: rgba(255, 255, 253, .97); border: 1px solid rgba(120, 110, 90, .55);
    border-radius: 8px; padding: 10px 14px; text-align: center; min-width: 260px;
  }
  #goro-charselect .delmodal input {
    width: 200px; margin: 8px 0; padding: 3px 6px; border-radius: 5px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .6);
    color: #2a2622; font: 13px Sarabun, monospace; text-align: center;
  }
  /* DOM loading label + status window */""",
)

# 2) markup — container after the title layer element
sub1(
    '<div id="goro-alert" style="display:none"></div></div>',
    '<div id="goro-alert" style="display:none"></div></div>\n  <div id="goro-charselect"></div>',
)

# 3) JS — renderer + action wiring beside the title layer block
sub1(
    "  // ---- DOM title layer: background + login + service (goroTitleSync) ----",
    """  // ---- DOM character select (goroCharSelectSync) ----
  (function () {
    var el = document.getElementById('goro-charselect');
    if (!el) return;
    function act(a) { if (window.goroCharSelectAction) window.goroCharSelectAction(a); }
    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }
    var built = '';
    window.goroCharSelectSync = function (s) {
      var show = s.phase === 'character' && s.fade < 0.999;
      el.style.display = show ? 'block' : 'none';
      el.style.opacity = String(1 - (s.fade || 0));
      if (!show) { built = ''; return; }
      var page = s.page || 0, pageCount = s.pageCount || 1;
      var selected = s.selected || 0, pageStart = page * 3;
      var selHasChar = false;
      for (var i = 0; i < (s.slots || []).length; i++) {
        if (pageStart + i === selected && (s.slots[i] || {}).name) selHasChar = true;
      }
      var key = [s.bg && s.bg.slice(-24), page, pageCount, selected, selHasChar, s.status, s.deleteStep, s.deleteName,
        (s.slots || []).map(function (d) { return (d && (d.name + '|' + d.level + '|' + (d.preview || '').slice(-16))) || '-'; }).join(';')].join('~');
      if (key === built) return;
      built = key;
      var html = (s.bg ? '<img class="bg" src="' + s.bg + '" draggable="false">' : '') +
        '<div class="panel"><div class="title">Select Character</div>' +
        '<div class="slotsrow">' +
        '<button class="arrow" data-a="prev">\u2039</button><div class="slots">';
      for (var i = 0; i < 3; i++) {
        var d = (s.slots || [])[i] || {};
        var slot = pageStart + i;
        html += '<button class="slot' + (slot === selected ? ' sel' : '') + '" data-slot="' + slot + '">' +
          (d.preview ? '<img src="' + d.preview + '" draggable="false">' : (d.name ? '' : 'Create')) + '</button>';
      }
      html += '</div><button class="arrow" data-a="next">\u203a</button></div>' +
        '<div class="pageinfo">' + (page + 1) + ' / ' + pageCount + '</div>';
      var sel = null;
      for (var i = 0; i < (s.slots || []).length; i++) { if (pageStart + i === selected) sel = s.slots[i] || {}; }
      if (sel && sel.name) {
        html += '<div class="info"><table>' +
          '<tr><td class="h">Name</td><td>' + esc(sel.name) + '</td><td class="s">STR</td><td class="v">' + esc(sel.str) + '</td></tr>' +
          '<tr><td class="h">Job</td><td>' + esc(sel.job) + '</td><td class="s">AGI</td><td class="v">' + esc(sel.agi) + '</td></tr>' +
          '<tr><td class="h">Level</td><td>' + esc(sel.level) + '</td><td class="s">VIT</td><td class="v">' + esc(sel.vit) + '</td></tr>' +
          '<tr><td class="h">Exp</td><td>' + esc(sel.exp) + '</td><td class="s">INT</td><td class="v">' + esc(sel.int) + '</td></tr>' +
          '<tr><td class="h">HP</td><td>' + esc(sel.hp) + '</td><td class="s">DEX</td><td class="v">' + esc(sel.dex) + '</td></tr>' +
          '<tr><td class="h">SP</td><td>' + esc(sel.sp) + '</td><td class="s">LUK</td><td class="v">' + esc(sel.luk) + '</td></tr>' +
          '</table></div>';
      } else {
        html += '<div class="info"><div class="empty">Empty Slot \u2014 use Make to create a character.</div></div>';
      }
      html += '<div class="foot">' +
        '<button data-a="delete"' + (selHasChar ? '' : ' disabled') + '>Delete</button>' +
        '<button data-a="make"' + (selHasChar ? ' disabled' : '') + '>Make</button>' +
        '<button data-a="ok">OK</button>' +
        '<button data-a="cancel">Cancel</button></div>' +
        '<div class="status">' + esc(s.status || '') + '</div></div>';
      if (s.deleteStep === 1) {
        html += '<div class="delmodal"><div class="t">Delete Character</div>' +
          '<div class="m">Delete ' + esc(s.deleteName) + '?</div>' +
          '<div class="b"><button data-a="delok">OK</button><button data-a="delcancel">Cancel</button></div></div>';
      } else if (s.deleteStep === 2) {
        html += '<div class="delmodal"><div class="t">Delete Character</div>' +
          '<div class="m">Enter the account email to confirm deleting ' + esc(s.deleteName) + '.</div>' +
          '<input type="text" id="goro-charselect-email" placeholder="Email" autocomplete="off">' +
          '<div class="b"><button data-a="delmail">Delete</button><button data-a="delmailcancel">Cancel</button></div></div>';
      }
      el.innerHTML = html;
      el.querySelectorAll('.arrow, .foot button, .delmodal button').forEach(function (b) {
        b.addEventListener('pointerdown', function (ev) {
          ev.stopPropagation();
          if (b.disabled) return;
          var a = b.getAttribute('data-a');
          if (a === 'delmail') {
            var input = document.getElementById('goro-charselect-email');
            var v = input ? input.value.trim() : '';
            if (!v) { if (input) input.focus(); return; }
            a = 'delmail:' + v;
          }
          act(a);
        });
      });
      el.querySelectorAll('.slot').forEach(function (b) {
        b.addEventListener('pointerdown', function (ev) {
          ev.stopPropagation();
          var slot = +b.getAttribute('data-slot');
          act(slot === selected ? 'activate:' + slot : 'select:' + slot);
        });
      });
      if (s.deleteStep === 2) {
        var input = document.getElementById('goro-charselect-email');
        if (input) {
          input.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
          input.addEventListener('keydown', function (ev) {
            ev.stopPropagation();
            if (ev.key === 'Enter') act('delmail:' + input.value.trim());
          });
        }
      }
    };
  })();
  // ---- DOM title layer: background + login + service (goroTitleSync) ----""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
