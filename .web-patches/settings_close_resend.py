import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# Settings window close (X button) re-sends the picked resolution before
# closing. Some touch browsers deliver a <select>'s final value late or drop
# the change/input events entirely; without the resend the picker keeps
# showing the picked percent while the canvas keeps rendering at the old
# scale. Re-sending on close lands the value the user sees selected.
sub1(
    """      set.querySelector('.x').addEventListener('pointerdown', function (ev) {
        ev.stopPropagation();
        setAct('close');
      });""",
    """      set.querySelector('.x').addEventListener('pointerdown', function (ev) {
        ev.stopPropagation();
        // Re-send the picked scale before closing: if the select's
        // change/input events were lost on a touch browser, closing the
        // window still lands the value the user sees selected.
        var sel = set.querySelector('select[data-k="scale"]');
        if (sel) setAct('scale:' + sel.value);
        setAct('close');
      });""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(src), len(src)))
