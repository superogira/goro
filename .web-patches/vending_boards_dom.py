import io, sys

# DOM twins of the in-world vending boards: keyed pills fed by
# goroVendingBoardsSync (Go syncs only when a board moves a full pixel or
# a title changes). pointer-events stay off — hover/click keep flowing to
# the game canvas, whose hit-testing reads the bounds captured at draw.
# Idempotent.

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

if "goroVendingBoardsSync" in src:
    print("vending boards already patched")
    sys.exit(0)

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# 1) CSS (end of the style block so it wins the cascade)
sub1(
    "</style>",
    """  /* DOM vending boards */
  #goro-vboards { position: fixed; inset: 0; z-index: 5; pointer-events: none; overflow: hidden; }
  #goro-vboards .vb {
    position: absolute; left: 0; top: 0; width: 158px; height: 36px;
    box-sizing: border-box; display: flex; align-items: center; gap: 5px;
    padding: 5px 5px 5px 5px; border-radius: 4px;
    background: rgba(255, 255, 255, .96); border: 1px solid rgba(74, 138, 202, .96);
    box-shadow: 0 0 0 3px rgba(255, 255, 255, .96);
    font: 12px/1 'Sarabun', 'Noto Sans Thai', system-ui, sans-serif; color: #1e2228;
    will-change: transform;
  }
  #goro-vboards .vb img {
    width: 24px; height: 24px; flex: none; object-fit: contain;
    image-rendering: pixelated;
  }
  #goro-vboards .vb span {
    flex: 1 1 auto; min-width: 0; white-space: nowrap;
    overflow: hidden; text-overflow: ellipsis;
  }
</style>""",
)

# 2) markup
sub1(
    '  <div id="goro-boardtip"></div>',
    '''  <div id="goro-boardtip"></div>
  <div id="goro-vboards"></div>''',
)

# 3) JS
sub1(
    "  // ---- DOM chat room board tooltip (goroBoardTipSync) ----",
    """  // ---- DOM vending boards (goroVendingBoardsSync) ----
  (function () {
    var root = document.getElementById('goro-vboards');
    var panels = {};   // id -> { el, x, y, title }
    var iconURL = encodeURI('data/texture/\\uc720\\uc800\\uc778\\ud130\\ud398\\uc774\\uc2a4/basic_interface/shop.bmp');
    window.goroVendingBoardsSync = function (boards) {
      var seen = {};
      (boards || []).forEach(function (b) {
        seen[b.id] = true;
        var p = panels[b.id];
        if (!p) {
          var el = document.createElement('div');
          el.className = 'vb';
          el.innerHTML = '<img alt="" src="' + iconURL + '"><span></span>';
          root.appendChild(el);
          p = panels[b.id] = { el: el, x: -1, y: -1, title: '' };
        }
        if (p.x !== b.x || p.y !== b.y) {
          p.x = b.x; p.y = b.y;
          p.el.style.transform = 'translate(' + b.x + 'px, ' + b.y + 'px)';
        }
        if (p.title !== b.title) {
          p.title = b.title;
          p.el.querySelector('span').textContent = b.title;
        }
      });
      Object.keys(panels).forEach(function (id) {
        if (!seen[id]) { panels[id].el.remove(); delete panels[id]; }
      });
    };
  })();
  // ---- DOM chat room board tooltip (goroBoardTipSync) ----""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("vending boards patched, %d -> %d bytes" % (len(orig), len(src)))
