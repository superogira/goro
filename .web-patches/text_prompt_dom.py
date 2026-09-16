import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# DOM text prompt (goroTextPromptSync): the twin of the canvas text
# prompt window, used by party invitations, guild reasons, the character
# delete email and skill ground messages. Family styling matches the
# other DOM panels.

# 1) CSS
sub1(
    "  /* DOM AEOPD document registry */",
    """  /* DOM text prompt */
  #goro-textprompt {
    position: fixed; z-index: 24; display: none; width: 320px; max-width: calc(100vw - 24px);
    box-sizing: border-box; left: 50%; top: 42%; transform: translate(-50%, -50%);
    background: rgba(255, 255, 253, .97); border: 1px solid rgba(120, 110, 90, .55);
    border-radius: 10px; padding: 8px 12px 12px; user-select: none;
    font: 13px/1.4 Sarabun, 'Noto Sans Thai', system-ui, sans-serif; color: #2a2622;
  }
  #goro-textprompt .title {
    text-align: center; font-weight: 600;
    margin: -8px -12px 10px; padding: 6px 10px;
    background: linear-gradient(180deg, #d6e8fa, #b8d6f2);
    border-bottom: 1px solid #76a0ce;
    border-radius: 9px 9px 0 0;
    color: #163658;
    position: relative;
  }
  #goro-textprompt .x {
    position: absolute; right: 4px; top: 3px; width: 18px; height: 18px;
    border-radius: 4px; background: rgba(232, 223, 200, .7); cursor: pointer;
    text-align: center; line-height: 16px; color: #6a6055; font-size: 11px;
  }
  #goro-textprompt .lbl { display: block; margin-bottom: 6px; color: #3a5a7c; font-size: 12px; }
  #goro-textprompt input {
    width: 100%; box-sizing: border-box; padding: 5px 8px; border-radius: 5px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .6);
    color: #2a2622; font: 13px Sarabun, monospace;
  }
  #goro-textprompt .btns { display: flex; justify-content: center; gap: 10px; margin-top: 10px; }
  #goro-textprompt .btns button {
    min-width: 74px; padding: 4px 14px; border-radius: 5px;
    border: 1px solid rgba(120,110,90,.5); background: rgba(232, 223, 200, .85);
    color: #2a2622; cursor: pointer; font: 13px/1.5 Sarabun, 'Noto Sans Thai', system-ui, sans-serif;
  }
  /* DOM AEOPD document registry */""",
)

# 2) markup
sub1(
    '  <div id="goro-saraban"></div>',
    '  <div id="goro-saraban"></div>\n  <div id="goro-textprompt"></div>',
)

# 3) JS renderer
sub1(
    "  // ---- DOM AEOPD document registry (NPC portal) ----",
    """  // ---- DOM text prompt (goroTextPromptSync) ----
  (function () {
    var el = document.getElementById('goro-textprompt');
    if (!el) return;
    var state = { open: false, title: '', label: '', placeholder: '', maxLength: 0 };
    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }
    function act(a) { if (window.goroTextPromptAction) window.goroTextPromptAction(a); }
    function submit() {
      var input = el.querySelector('input');
      var text = (input ? input.value : '').trim();
      if (!text) return;
      state.open = false;
      render();
      act('submit:' + text);
    }
    function cancel() {
      state.open = false;
      render();
      act('cancel');
    }
    function render() {
      if (!state.open) { el.style.display = 'none'; el.innerHTML = ''; return; }
      el.style.display = 'block';
      var max = state.maxLength > 0 ? ' maxlength="' + state.maxLength + '"' : '';
      el.innerHTML = '<div class="title">' + esc(state.title || '') + '<div class="x">\\u2715</div></div>' +
        '<label class="lbl">' + esc(state.label || '') + '</label>' +
        '<input type="text" placeholder="' + esc(state.placeholder || '') + '"' + max + ' autocomplete="off" spellcheck="false">' +
        '<div class="btns"><button class="ok">OK</button><button class="no">Cancel</button></div>';
      el.querySelector('.x').addEventListener('pointerdown', function (ev) { ev.stopPropagation(); cancel(); });
      var input = el.querySelector('input');
      ['keydown', 'keyup', 'keypress'].forEach(function (k) {
        input.addEventListener(k, function (ev) {
          ev.stopPropagation();
          if (ev.key === 'Enter') { ev.preventDefault(); submit(); }
          if (ev.key === 'Escape') { ev.preventDefault(); cancel(); }
        });
      });
      input.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
      el.querySelector('.ok').addEventListener('pointerdown', function (ev) { ev.stopPropagation(); submit(); });
      el.querySelector('.no').addEventListener('pointerdown', function (ev) { ev.stopPropagation(); cancel(); });
      setTimeout(function () { input.focus(); }, 0);
    }
    window.goroTextPromptSync = function (d) {
      state.open = !!d.open;
      state.title = d.title || '';
      state.label = d.label || '';
      state.placeholder = d.placeholder || '';
      state.maxLength = d.maxLength || 0;
      render();
    };
  })();
  // ---- DOM AEOPD document registry (NPC portal) ----""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
