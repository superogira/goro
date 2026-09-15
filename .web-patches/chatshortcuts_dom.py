import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# DOM chat shortcuts panel (Alt+M): ten command rows with the Alt+N labels,
# a View button (emote picker) and Close — the DOM twin of the canvas
# ChatShortcutsWindow. Family styling matches the character panels. While a
# command field is focused, incoming syncs are deferred to focus-out so
# typing never rebuilds the row under the caret.

# 1) CSS
sub1(
    "  /* DOM make character */",
    """  /* DOM chat shortcuts (Alt+M) */
  #goro-chatshortcuts {
    position: fixed; z-index: 21; display: none; width: 300px; box-sizing: border-box;
    left: 50%; top: 50%; transform: translate(-50%, -50%);
    background: rgba(255, 255, 253, .96); border: 1px solid rgba(120, 110, 90, .55);
    border-radius: 10px; padding: 8px 12px 12px; user-select: none;
    font: 13px/1.4 Sarabun, 'Noto Sans Thai', system-ui, sans-serif; color: #2a2622;
  }
  #goro-chatshortcuts .title {
    text-align: center; font-weight: 600;
    margin: -8px -12px 10px; padding: 6px 10px;
    background: linear-gradient(180deg, #d6e8fa, #b8d6f2);
    border-bottom: 1px solid #76a0ce;
    border-radius: 9px 9px 0 0;
    color: #163658;
  }
  #goro-chatshortcuts .row { display: flex; align-items: center; gap: 8px; height: 24px; margin-bottom: 4px; }
  #goro-chatshortcuts .row .k {
    width: 54px; flex: none; text-align: center; font-size: 12px;
    background: #deedfc; color: #3a689c; border-radius: 4px; padding: 1px 0;
  }
  #goro-chatshortcuts .row input {
    flex: 1; min-width: 0; padding: 2px 6px; border-radius: 4px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .6);
    color: #2a2622; font: 12px Sarabun, monospace;
  }
  #goro-chatshortcuts .foot { display: flex; justify-content: flex-end; gap: 8px; margin-top: 8px; }
  #goro-chatshortcuts .foot button {
    min-width: 56px; padding: 2px 12px; border-radius: 5px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .8);
    color: #2a2622; cursor: pointer; font: 12px/1.6 Sarabun, 'Noto Sans Thai', system-ui, sans-serif;
  }
  /* DOM make character */""",
)

# 2) markup
sub1(
    '<div id="goro-charcreate"></div>',
    '<div id="goro-charcreate"></div>\n  <div id="goro-chatshortcuts"></div>',
)

# 3) JS renderer beside the char create block
sub1(
    "  // ---- DOM make character (goroCharCreateSync) ----",
    """  // ---- DOM chat shortcuts (goroChatShortcutsSync) ----
  (function () {
    var el = document.getElementById('goro-chatshortcuts');
    if (!el) return;
    function act(a) { if (window.goroChatShortcutsAction) window.goroChatShortcutsAction(a); }
    var built = '';
    var pending = null;
    function focusedInput() {
      var a = document.activeElement;
      return a && a.tagName === 'INPUT' && el.contains(a) ? a : null;
    }
    function render(s) {
      var html = '<div class="title">Shortcut List</div>';
      for (var i = 0; i < (s.commands || []).length; i++) {
        html += '<div class="row"><span class="k">Alt + ' + ((i + 1) % 10) + '</span>' +
          '<input type="text" data-slot="' + i + '" maxlength="80" autocomplete="off" spellcheck="false"></div>';
      }
      html += '<div class="foot"><button data-a="view">View</button><button data-a="close">Close</button></div>';
      el.innerHTML = html;
      el.querySelectorAll('.row input').forEach(function (input) {
        input.value = (s.commands || [])[+input.getAttribute('data-slot')] || '';
        input.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
        ['keydown', 'keyup'].forEach(function (k) {
          input.addEventListener(k, function (ev) {
            ev.stopPropagation();
            if (ev.key === 'Escape') input.blur();
          });
        });
        input.addEventListener('input', function () {
          act('set:' + input.getAttribute('data-slot') + ':' + input.value);
        });
      });
      el.querySelectorAll('.foot button').forEach(function (b) {
        b.addEventListener('pointerdown', function (ev) {
          ev.stopPropagation();
          act(b.getAttribute('data-a'));
        });
      });
    }
    window.goroChatShortcutsSync = function (s) {
      if (!s.open) { el.style.display = 'none'; built = ''; pending = null; return; }
      el.style.display = 'block';
      // Typing echo would rebuild the focused row: defer until focus leaves.
      if (focusedInput()) { pending = s; return; }
      render(s);
      built = 'x';
    };
    el.addEventListener('focusout', function () {
      setTimeout(function () {
        if (pending && !focusedInput()) { render(pending); pending = null; }
      }, 0);
    });
  })();
  // ---- DOM make character (goroCharCreateSync) ----""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
