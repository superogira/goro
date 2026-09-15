import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# OS-cursor mode for ?nocursor=1: the game calls goroCursorOSMode(true)
# while the debug no-cursor mode is active. It drops the pointer-seen class
# (restoring the OS arrow through the boot-state CSS) and stops the sprite
# from reappearing; goroCursorOSMode(false) from normal cursor frames
# re-enables the sprite everywhere.

sub1(
    """    var curTrack = { seen: false, onCanvas: false, x: 0, y: 0, dx: 0, dy: 0 };
    window.addEventListener('mousemove', function (ev) {
      curTrack.x = ev.clientX; curTrack.y = ev.clientY;
      curTrack.onCanvas = !!(ev.target && ev.target.tagName === 'CANVAS');
      if (!curTrack.seen) {
        curTrack.seen = true;
        document.body.classList.add('goro-pointer-seen');
      }
    }, true);""",
    """    var curTrack = { seen: false, onCanvas: false, osMode: false, x: 0, y: 0, dx: 0, dy: 0 };
    window.goroCursorOSMode = function (on) {
      on = !!on;
      if (on === curTrack.osMode) return;
      curTrack.osMode = on;
      if (on) {
        curTrack.seen = false;
        document.body.classList.remove('goro-pointer-seen');
        curEl.style.display = 'none';
      }
    };
    window.addEventListener('mousemove', function (ev) {
      curTrack.x = ev.clientX; curTrack.y = ev.clientY;
      curTrack.onCanvas = !!(ev.target && ev.target.tagName === 'CANVAS');
      if (!curTrack.seen && !curTrack.osMode) {
        curTrack.seen = true;
        document.body.classList.add('goro-pointer-seen');
      }
    }, true);""",
)
sub1(
    """    window.goroCursorSync = function (url, x, y) {
      if (!url || !curTrack.seen) { curEl.style.display = 'none'; return; }""",
    """    window.goroCursorSync = function (url, x, y) {
      if (!url || !curTrack.seen || curTrack.osMode) { curEl.style.display = 'none'; return; }""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
