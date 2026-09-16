import io, sys

# DOM tooltip for the in-world chat room boards: the board pill trims long
# titles, so hovering one shows the full label near the cursor. Fed by
# goroBoardTipSync from the game's cursor path; display-only.
# Idempotent.

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

if "goroBoardTipSync" in src:
    print("board tip already patched")
    sys.exit(0)

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# 1) CSS (end of the style block so it wins the cascade)
sub1(
    "</style>",
    """  /* DOM chat room board tooltip */
  #goro-boardtip {
    position: fixed; display: none; z-index: 24; pointer-events: none;
    max-width: min(420px, calc(100vw - 12px)); padding: 5px 10px; border-radius: 6px;
    background: rgba(255, 253, 248, .97); border: 1px solid rgba(120, 110, 90, .55);
    box-shadow: 0 2px 8px rgba(20, 24, 16, .25);
    font: 12px/1.5 Sarabun, 'Noto Sans Thai', system-ui, sans-serif; color: #2a2622;
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
</style>""",
)

# 2) markup
sub1(
    '  <div id="goro-chatroom" class="goro-social" style="left:24px;top:120px"></div>',
    '''  <div id="goro-chatroom" class="goro-social" style="left:24px;top:120px"></div>
  <div id="goro-boardtip"></div>''',
)

# 3) JS
sub1(
    "  // ---- DOM text prompt (goroTextPromptSync) ----",
    """  // ---- DOM chat room board tooltip (goroBoardTipSync) ----
  (function () {
    var tip = document.getElementById('goro-boardtip');
    window.goroBoardTipSync = function (d) {
      if (!d || !d.show || !d.text) { tip.style.display = 'none'; tip.textContent = ''; return; }
      tip.textContent = d.text;
      tip.style.display = 'block';
      var x = Math.max(6, Math.min(d.x + 14, window.innerWidth - tip.offsetWidth - 6));
      var y = d.y - tip.offsetHeight - 12;
      if (y < 4) y = Math.min(d.y + 18, window.innerHeight - tip.offsetHeight - 4);
      tip.style.left = x + 'px';
      tip.style.top = y + 'px';
    };
  })();
  // ---- DOM text prompt (goroTextPromptSync) ----""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("board tip patched, %d -> %d bytes" % (len(orig), len(src)))
