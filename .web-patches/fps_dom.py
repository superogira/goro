import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new, count=1):
    global src
    if src.count(old) < count:
        print("ANCHOR NOT FOUND:", old[:90]); sys.exit(1)
    src = src.replace(old, new, count)

# 1) CSS: small crisp meter at the top-left corner
sub1(
    "  #goro-loading {",
    """  #goro-fps {
    position: fixed; left: 6px; top: 6px; z-index: 31; display: none;
    font: 12px/1.4 monospace; color: #ffe9a0; pointer-events: none;
    text-shadow: 0 1px 2px rgba(0, 0, 0, .85); letter-spacing: .3px;
    white-space: pre;
  }
  #goro-loading {""",
)

# 2) markup
sub1(
    '<div id="goro-loading"><span>Now Loading...</span>',
    '<div id="goro-fps"></div><div id="goro-loading"><span>Now Loading...</span>',
)

# 3) hook next to the loading label hook
sub1(
    """    window.goroLoadingSet = function (v) {
      loading.classList.toggle('show', !!v);
    };""",
    """    window.goroLoadingSet = function (v) {
      loading.classList.toggle('show', !!v);
    };
    var fpsEl = document.getElementById('goro-fps');
    window.goroFpsSync = function (text) {
      fpsEl.textContent = text || '';
      fpsEl.style.display = text ? 'block' : 'none';
    };""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
