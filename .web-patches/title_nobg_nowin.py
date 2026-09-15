import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# Title layer debug-flag support: nobg=1 now pushes state without a
# background URL (plain black layer, login form still playable), and
# nowin=1 hides the DOM login form the way it hid the canvas window.

sub1(
    "  #goro-title {",
    """  #goro-title.nowin #goro-login, #goro-title.nowin #goro-service { display: none !important; }
  #goro-title {""",
)
sub1(
    """    window.goroTitleSync = function (s) {
      if (s.bg && img.getAttribute('src') !== s.bg) img.setAttribute('src', s.bg);
      var show = s.phase === 'account' && s.fade < 0.999;""",
    """    window.goroTitleSync = function (s) {
      if (s.bg && img.getAttribute('src') !== s.bg) img.setAttribute('src', s.bg);
      layer.classList.toggle('nowin', !!s.nowin);
      var show = s.phase === 'account' && s.fade < 0.999;""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
