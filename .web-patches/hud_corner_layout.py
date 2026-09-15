import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# Bottom-right corner layout: version badge at the very corner, FPS above
# it, and the show/hide-controls chevron above the FPS (it used to live in
# the middle-right camera cluster).

# 1) CSS: corner positions + standalone toggle button
sub1(
    "  #goro-fps {\n    position: fixed; right: 10px; bottom: 88px; z-index: 31; display: none;",
    "  #goro-fps {\n    position: fixed; right: 10px; bottom: 34px; z-index: 31; display: none;",
)
sub1(
    "  #goro-version {\n    position: fixed; right: 10px; bottom: 64px; z-index: 31; display: none;",
    "  #goro-version {\n    position: fixed; right: 10px; bottom: 10px; z-index: 31; display: none;",
)
sub1(
    "  #goro-cam #goro-touch-toggle.on { background: #10ef21; color: #143c00; }",
    """  #goro-cam #goro-touch-toggle.on { background: #10ef21; color: #143c00; }
  #goro-cam-toggle {
    position: fixed; right: 10px; bottom: 64px; z-index: 31; display: none;
    width: 30px; height: 30px; border: 1px solid rgba(120, 110, 90, .5);
    border-radius: 8px; background: rgba(232, 223, 200, .95); color: #2a2622;
    cursor: pointer; font: 13px Sarabun, system-ui, sans-serif; touch-action: manipulation;
  }
  #goro-cam-toggle:active { background: #d8cba8; }""",
)

# 2) markup: lift the toggle out of the camera cluster
sub1(
    """    <button id="goro-touch-toggle" title="Touch controls">&#128073;</button>
    <button class="toggle" title="Show/hide controls">&#187;</button>
  </div>""",
    """    <button id="goro-touch-toggle" title="Touch controls">&#128073;</button>
  </div>
  <button id="goro-cam-toggle" class="toggle" title="Show/hide controls">&#187;</button>""",
)

# 3) JS: the toggle now lives outside the cluster element
sub1(
    "      cam.querySelector('.toggle').textContent = camHidden ? '\u00ab' : '\u00bb';",
    "      document.getElementById('goro-cam-toggle').textContent = camHidden ? '\u00ab' : '\u00bb';",
)
sub1(
    "    cam.querySelector('.toggle').addEventListener('pointerdown', function (ev) {",
    "    document.getElementById('goro-cam-toggle').addEventListener('pointerdown', function (ev) {",
)

sub1(
    "    cam.style.display = 'flex';",
    "    cam.style.display = 'flex';
    document.getElementById('goro-cam-toggle').style.display = 'block';",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
