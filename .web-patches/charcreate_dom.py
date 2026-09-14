import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# DOM make-character layer: sprite preview with hair style/color controls,
# the name field, the stat steppers (clicking a stat takes a point from its
# pair, mirroring bumpCreateStat) and the Make/Cancel footer — the DOM twin
# of the canvas CharacterCreateWindow. Family styling matches the character
# select panel (title gradient bar, classic table).

# 1) CSS
sub1(
    "  /* DOM character select */",
    """  /* DOM make character */
  #goro-charcreate {
    position: fixed; inset: 0; z-index: 3; display: none; overflow: hidden;
    font: 13px/1.4 Sarabun, 'Noto Sans Thai', system-ui, sans-serif; color: #2a2622;
  }
  #goro-charcreate .bg { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: cover; }
  #goro-charcreate .panel {
    position: absolute; left: 50%; top: 50%; transform: translate(-50%, -50%);
    width: 470px; max-width: calc(100vw - 24px); box-sizing: border-box;
    background: rgba(255, 255, 253, .96); border: 1px solid rgba(120, 110, 90, .55);
    border-radius: 10px; padding: 8px 14px 12px; user-select: none;
  }
  #goro-charcreate .title {
    text-align: center; font-weight: 600;
    margin: -8px -14px 10px; padding: 6px 10px;
    background: linear-gradient(180deg, #d6e8fa, #b8d6f2);
    border-bottom: 1px solid #76a0ce;
    border-radius: 9px 9px 0 0;
    color: #163658;
  }
  #goro-charcreate .body { display: flex; gap: 18px; align-items: flex-start; justify-content: center; }
  #goro-charcreate .left { width: 142px; flex: none; }
  #goro-charcreate .preview {
    width: 142px; height: 166px; box-sizing: border-box;
    border: 1px solid rgba(120, 110, 90, .45); border-radius: 6px;
    background: rgba(255,255,255,.5);
    display: flex; align-items: center; justify-content: center;
  }
  #goro-charcreate .preview img { max-width: 134px; max-height: 158px; image-rendering: pixelated; }
  #goro-charcreate .hairrow { display: flex; gap: 4px; margin-top: 6px; }
  #goro-charcreate .hairrow button {
    flex: 1; min-width: 0; padding: 2px 0; border-radius: 5px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .8);
    color: #2a2622; cursor: pointer; font: 12px/1.5 Sarabun, 'Noto Sans Thai', system-ui, sans-serif;
  }
  #goro-charcreate .namerow { margin-top: 10px; }
  #goro-charcreate .namerow .lbl { color: #6a6055; font-size: 12px; margin-bottom: 2px; }
  #goro-charcreate .namerow input {
    width: 100%; box-sizing: border-box; padding: 3px 6px; border-radius: 5px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .6);
    color: #2a2622; font: 13px Sarabun, monospace;
  }
  #goro-charcreate .stats { flex: none; }
  #goro-charcreate .stats table { border-collapse: collapse; font-size: 12px; }
  #goro-charcreate .stats td { padding: 1px 4px; }
  #goro-charcreate .stats td.h { background: #deedfc; color: #3a689c; width: 40px; }
  #goro-charcreate .stats td.p { color: #8a94a0; font-size: 11px; width: 66px; }
  #goro-charcreate .stats td.v { text-align: right; background: #fff; color: #1a1a1a; width: 30px; }
  #goro-charcreate .stats td.b { width: 26px; }
  #goro-charcreate .stats td.b button {
    width: 22px; height: 18px; padding: 0; border-radius: 4px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .8);
    color: #2a2622; cursor: pointer; font: 12px/1 monospace;
  }
  #goro-charcreate .stats td.b button:disabled { opacity: .35; cursor: default; }
  #goro-charcreate .foot { display: flex; justify-content: flex-end; gap: 8px; margin-top: 12px; }
  #goro-charcreate .foot button {
    min-width: 64px; padding: 3px 14px; border-radius: 5px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .8);
    color: #2a2622; cursor: pointer; font: 13px/1.6 Sarabun, 'Noto Sans Thai', system-ui, sans-serif;
  }
  #goro-charcreate .status { text-align: center; color: #6a6055; font-size: 12px; min-height: 16px; margin-top: 6px; }
  #goro-charcreate .status.err { color: #b04030; }
  /* DOM character select */""",
)

# 2) markup
sub1(
    '<div id="goro-charselect"></div>',
    '<div id="goro-charselect"></div>\n  <div id="goro-charcreate"></div>',
)

# 3) JS — renderer + action wiring beside the char select block
sub1(
    "  // ---- DOM character select (goroCharSelectSync) ----",
    """  // ---- DOM make character (goroCharCreateSync) ----
  (function () {
    var el = document.getElementById('goro-charcreate');
    if (!el) return;
    function act(a) { if (window.goroCharCreateAction) window.goroCharCreateAction(a); }
    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }
    var STAT_KEYS = ['str', 'agi', 'vit', 'int', 'dex', 'luk'];
    var STAT_PAIRS = ['INT', 'LUK', 'DEX', 'STR', 'AGI', 'VIT'];
    var built = '';
    window.goroCharCreateSync = function (s) {
      var show = s.phase === 'create' && s.fade < 0.999;
      el.style.display = show ? 'block' : 'none';
      el.style.opacity = String(1 - (s.fade || 0));
      if (!show) { built = ''; return; }
      // The name is deliberately NOT part of the rebuild key: it changes
      // with every keystroke and rebuilding would drop the input focus.
      var key = [(s.bg || '').slice(-24), (s.stats || []).join(','), s.hairStyle, s.hairColor, s.status,
        (s.preview || '').slice(-16), Math.round((s.fade || 0) * 100)].join('~');
      if (key !== built) {
        built = key;
        var html = (s.bg ? '<img class="bg" src="' + s.bg + '" draggable="false">' : '') +
          '<div class="panel"><div class="title">Make Character</div>' +
          '<div class="body">' +
          '<div class="left">' +
          '<div class="preview">' + (s.preview ? '<img src="' + s.preview + '" draggable="false">' : '') + '</div>' +
          '<div class="hairrow">' +
          '<button data-a="hairprev">\u2039</button>' +
          '<button data-a="haircolor">Color</button>' +
          '<button data-a="hairnext">\u203a</button>' +
          '</div>' +
          '<div class="namerow"><div class="lbl">Name</div>' +
          '<input type="text" id="goro-charcreate-name" maxlength="23" autocomplete="off" spellcheck="false" placeholder="Character name">' +
          '</div></div>' +
          '<div class="stats"><table>';
        for (var i = 0; i < 6; i++) {
          var v = (s.stats || [])[i];
          var maxed = v >= 9;
          html += '<tr><td class="h">' + STAT_KEYS[i].toUpperCase() + '</td>' +
            '<td class="p">\u21c4 ' + STAT_PAIRS[i] + '</td>' +
            '<td class="v">' + (v == null ? '' : v) + '</td>' +
            '<td class="b"><button data-stat="' + i + '"' + (maxed ? ' disabled' : '') + '>+</button></td></tr>';
        }
        html += '</table></div></div>' +
          '<div class="foot"><button data-a="submit">Make</button><button data-a="cancel">Cancel</button></div>' +
          '<div class="status">' + esc(s.status || '') + '</div></div>';
        el.innerHTML = html;
        el.querySelectorAll('.hairrow button, .foot button').forEach(function (b) {
          b.addEventListener('pointerdown', function (ev) {
            ev.stopPropagation();
            act(b.getAttribute('data-a'));
          });
        });
        el.querySelectorAll('.stats td.b button').forEach(function (b) {
          b.addEventListener('pointerdown', function (ev) {
            ev.stopPropagation();
            if (b.disabled) return;
            act('stat:' + b.getAttribute('data-stat'));
          });
        });
        var input = document.getElementById('goro-charcreate-name');
        if (input) {
          input.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
          input.addEventListener('input', function () { act('name:' + input.value); });
          input.addEventListener('keydown', function (ev) {
            ev.stopPropagation();
            if (ev.key === 'Enter') { ev.preventDefault(); act('submit'); }
          });
        }
      }
      // Reflect the name without clobbering the field being typed into.
      var nameInput = document.getElementById('goro-charcreate-name');
      if (nameInput && document.activeElement !== nameInput && nameInput.value !== (s.name || '')) {
        nameInput.value = s.name || '';
      }
    };
  })();
  // ---- DOM character select (goroCharSelectSync) ----""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
