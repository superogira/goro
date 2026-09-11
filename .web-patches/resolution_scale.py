import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new, count=1):
    global src
    if src.count(old) < count:
        print("ANCHOR NOT FOUND:", old[:90]); sys.exit(1)
    src = src.replace(old, new, count)

# 1) select styling for the resolution row
sub1(
    "  #goro-settings .srow .val { width: 34px; text-align: right; color: #6a6055; font-size: 11px; flex: none; }",
    """  #goro-settings .srow .val { width: 34px; text-align: right; color: #6a6055; font-size: 11px; flex: none; }
  #goro-settings .srow select {
    flex: 1; height: 24px; border-radius: 5px; border: 1px solid rgba(120, 110, 90, .5);
    background: #fff; color: #2a2622; font: 12px Sarabun,'Noto Sans Thai',system-ui,sans-serif;
  }""",
)

# 2) the Resolution row after the FPS meter toggle
sub1(
    "      html += toggle('fps', 'FPS meter', s.fps);",
    """      html += toggle('fps', 'FPS meter', s.fps);
      html += '<div class="srow"><span class="lbl">Resolution</span>' +
        '<select data-k="scale">' + [100, 90, 80, 70, 60, 50].map(function (p) {
          return '<option value="' + (p / 100).toFixed(2) + '"' +
            (Math.abs((s.scale || 1) - p / 100) < 0.005 ? ' selected' : '') + '>' + p + '%</option>';
        }).join('') + '</select></div>';""",
)

# 3) wire the select: pointerdown must not drag the window, change applies live
sub1(
    """      set.querySelectorAll('input[type=range]').forEach(function (el) {
        el.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
        el.addEventListener('input', function () {
          set.querySelector('[data-v="' + el.getAttribute('data-k') + '"]').textContent = Math.round(el.value * 100) + '%';
          setAct(el.getAttribute('data-k') + ':' + (+el.value).toFixed(2));
        });
      });""",
    """      set.querySelectorAll('input[type=range]').forEach(function (el) {
        el.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
        el.addEventListener('input', function () {
          set.querySelector('[data-v="' + el.getAttribute('data-k') + '"]').textContent = Math.round(el.value * 100) + '%';
          setAct(el.getAttribute('data-k') + ':' + (+el.value).toFixed(2));
        });
      });
      set.querySelectorAll('select').forEach(function (el) {
        el.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
        el.addEventListener('change', function () {
          setAct(el.getAttribute('data-k') + ':' + el.value);
        });
      });""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
