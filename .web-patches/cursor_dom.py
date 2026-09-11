import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new, count=1):
    global src
    if src.count(old) < count:
        print("ANCHOR NOT FOUND:", old[:90]); sys.exit(1)
    src = src.replace(old, new, count)

# 1) CSS: fixed element above the canvas but below every DOM window, so
#    panels cover it exactly like they covered the canvas cursor
sub1(
    "  #goro-fps {",
    """  #goro-cursor {
    position: fixed; left: 0; top: 0; z-index: 1; display: none;
    pointer-events: none; will-change: transform;
  }
  #goro-cursor img { display: block; image-rendering: pixelated; }
  #goro-fps {""",
)

# 2) markup (before the FPS meter element)
sub1(
    '<div id="goro-fps"></div>',
    '<div id="goro-cursor"><img alt="" draggable="false"></div><div id="goro-fps"></div>',
)

# 3) hook beside the version badge hook
sub1(
    """    var verEl = document.getElementById('goro-version');
    window.goroVersionSync = function (label) {
      verEl.textContent = label || '';
      verEl.style.display = label ? 'block' : 'none';
    };""",
    """    var verEl = document.getElementById('goro-version');
    window.goroVersionSync = function (label) {
      verEl.textContent = label || '';
      verEl.style.display = label ? 'block' : 'none';
    };
    var curEl = document.getElementById('goro-cursor');
    var curImg = curEl.querySelector('img');
    window.goroCursorSync = function (url, x, y) {
      if (!url) { curEl.style.display = 'none'; return; }
      curEl.style.display = 'block';
      if (curImg.getAttribute('src') !== url) curImg.setAttribute('src', url);
      curEl.style.transform = 'translate(' + x + 'px,' + y + 'px)';
    };""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
