import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# Gameplay section gains a Snap Radius slider (0.5x-3x). It rides the shared
# range-input listeners, so change/input events already route to
# setAct('snapradius:<value>').
sub1(
    """      html += toggle('snap', 'Snap to targets', s.snapTargets);
      html += toggle('itemsnap', 'Snap to items', s.snapItems);
      html += '</div>';""",
    """      html += toggle('snap', 'Snap to targets', s.snapTargets);
      html += toggle('itemsnap', 'Snap to items', s.snapItems);
      html += '<div class="srow"><span class="lbl">Snap Radius</span>' +
        '<input type="range" min="0.5" max="3" step="0.25" value="' + (s.snapRadius || 1) + '" data-k="snapradius">' +
        '<span class="val" data-v="snapradius">' + Math.round((s.snapRadius || 1) * 100) + '%</span></div>';
      html += '</div>';""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok")
