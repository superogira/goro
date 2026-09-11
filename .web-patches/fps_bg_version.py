import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new, count=1):
    global src
    if src.count(old) < count:
        print("ANCHOR NOT FOUND:", old[:90]); sys.exit(1)
    src = src.replace(old, new, count)

# 1) FPS meter gains the faint black backdrop the canvas meter had;
#    version badge DOM twin at the old canvas spot (right, 64px up)
sub1(
    """  #goro-fps {
    position: fixed; left: 6px; top: 6px; z-index: 31; display: none;
    font: 12px/1.4 monospace; color: #ffe9a0; pointer-events: none;
    text-shadow: 0 1px 2px rgba(0, 0, 0, .85); letter-spacing: .3px;
    white-space: pre;
  }""",
    """  #goro-fps {
    position: fixed; left: 6px; top: 6px; z-index: 31; display: none;
    font: 12px/1.4 monospace; color: #ffe9a0; pointer-events: none;
    text-shadow: 0 1px 2px rgba(0, 0, 0, .85); letter-spacing: .3px;
    white-space: pre; background: rgba(0, 0, 0, .45);
    padding: 2px 6px; border-radius: 4px;
  }
  #goro-version {
    position: fixed; right: 10px; bottom: 64px; z-index: 31; display: none;
    font: 11px/1.4 monospace; color: #c8d0d8; pointer-events: none;
    text-shadow: 0 1px 2px rgba(0, 0, 0, .8); letter-spacing: .3px;
    background: rgba(0, 0, 0, .35); padding: 2px 6px; border-radius: 4px;
  }""",
)

# 2) markup
sub1(
    '<div id="goro-fps"></div>',
    '<div id="goro-fps"></div><div id="goro-version"></div>',
)

# 3) hook beside goroFpsSync
sub1(
    """    window.goroFpsSync = function (text) {
      fpsEl.textContent = text || '';
      fpsEl.style.display = text ? 'block' : 'none';
    };""",
    """    window.goroFpsSync = function (text) {
      fpsEl.textContent = text || '';
      fpsEl.style.display = text ? 'block' : 'none';
    };
    var verEl = document.getElementById('goro-version');
    window.goroVersionSync = function (label) {
      verEl.textContent = label || '';
      verEl.style.display = label ? 'block' : 'none';
    };""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
