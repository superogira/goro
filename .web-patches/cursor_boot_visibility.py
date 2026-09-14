import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# Title screen boot cursor gap: the game hides the OS cursor on the canvas
# immediately, but until the first real mousemove it still believes the
# pointer sits at (0,0) — its DOM sprite renders in the top-left corner
# while nothing marks the real pointer position. Keep the OS arrow (and
# hide the stray sprite) until the canvas has actually seen the mouse once;
# that same event delivers the true position, so the sprite takes over
# exactly under the pointer on the next frame.

# 1) CSS overrides while the handover has not happened yet. !important is
#    required: the game writes cursor:none as an inline style on the canvas.
sub1(
    "  #goro-cursor img { display: block; image-rendering: pixelated; }",
    """  #goro-cursor img { display: block; image-rendering: pixelated; }
  body:not(.goro-pointer-seen) canvas { cursor: default !important; }
  body:not(.goro-pointer-seen) #goro-cursor { display: none !important; }""",
)

# 2) handover watcher beside the cursor sync hook. The canvas is created at
#    runtime, so a window-level capture listener checking the event target
#    survives canvas replacement; classList.add is idempotent.
sub1(
    """    var curEl = document.getElementById('goro-cursor');
    var curImg = curEl.querySelector('img');
    window.goroCursorSync = function (url, x, y) {
      if (!url) { curEl.style.display = 'none'; return; }
      curEl.style.display = 'block';
      if (curImg.getAttribute('src') !== url) curImg.setAttribute('src', url);
      curEl.style.transform = 'translate(' + x + 'px,' + y + 'px)';
    };""",
    """    var curEl = document.getElementById('goro-cursor');
    var curImg = curEl.querySelector('img');
    window.goroCursorSync = function (url, x, y) {
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
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
