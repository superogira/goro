import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# Sprite cursor everywhere, like the original client: once the pointer has
# been seen anywhere on the page, the OS cursor is hidden on the whole page
# (not just the canvas) and the DOM sprite follows the pointer over DOM
# layers too — the page tracks the real position and only trusts the game's
# position while the pointer is over the canvas (where magnet snapping and
# per-action anchors apply; the anchor delta is remembered and reused while
# hovering DOM overlays).

# 1) CSS: hide the OS cursor everywhere once the pointer is tracked; keep
#    the drag feedback on top of that.
sub1(
    """  body:not(.goro-pointer-seen) canvas { cursor: default !important; }
  body:not(.goro-pointer-seen) #goro-cursor { display: none !important; }""",
    """  body:not(.goro-pointer-seen) canvas { cursor: default !important; }
  body:not(.goro-pointer-seen) #goro-cursor { display: none !important; }
  body.goro-pointer-seen, body.goro-pointer-seen * { cursor: none !important; }
  body.goro-pointer-seen.goro-inv-dragging, body.goro-pointer-seen.goro-inv-dragging * { cursor: grabbing !important; }""",
)

# 2) JS: window-level position tracking + canvas-trusted game coordinates
sub1(
    """    window.goroCursorSync = function (url, x, y) {
      if (!url) { curEl.style.display = 'none'; return; }
      curEl.style.display = 'block';
      if (curImg.getAttribute('src') !== url) curImg.setAttribute('src', url);
      curEl.style.transform = 'translate(' + x + 'px,' + y + 'px)';
    };
    window.addEventListener('mousemove', function (ev) {
      if (ev.target && ev.target.tagName === 'CANVAS') {
        document.body.classList.add('goro-pointer-seen');
      }
    }, true);""",
    """    var curTrack = { seen: false, onCanvas: false, x: 0, y: 0, dx: 0, dy: 0 };
    window.addEventListener('mousemove', function (ev) {
      curTrack.x = ev.clientX; curTrack.y = ev.clientY;
      curTrack.onCanvas = !!(ev.target && ev.target.tagName === 'CANVAS');
      if (!curTrack.seen) {
        curTrack.seen = true;
        document.body.classList.add('goro-pointer-seen');
      }
    }, true);
    window.goroCursorSync = function (url, x, y) {
      if (!url || !curTrack.seen) { curEl.style.display = 'none'; return; }
      if (curTrack.onCanvas) {
        // Game coordinates are live here: remember the anchor (and magnet)
        // delta so DOM-layer positions land with the same hotspot.
        curTrack.dx = curTrack.x - x; curTrack.dy = curTrack.y - y;
        curEl.style.transform = 'translate(' + x + 'px,' + y + 'px)';
      } else {
        curEl.style.transform = 'translate(' + (curTrack.x - curTrack.dx) + 'px,' + (curTrack.y - curTrack.dy) + 'px)';
      }
      curEl.style.display = 'block';
      if (curImg.getAttribute('src') !== url) curImg.setAttribute('src', url);
    };""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
