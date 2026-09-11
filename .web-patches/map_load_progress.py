import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new, count=1):
    global src
    if src.count(old) < count:
        print("ANCHOR NOT FOUND:", old[:90]); sys.exit(1)
    src = src.replace(old, new, count)

# 1) loading label gains a progress bar (shown only while a map pack
#    actually downloads during a map change)
sub1(
    '<div id="goro-loading">Now Loading...</div>',
    '<div id="goro-loading"><span>Now Loading...</span>' +
    '<div class="bar"><div class="track"><div class="fill"></div></div><div class="plabel"></div></div></div>',
)
sub1(
    """  #goro-loading {
    position: fixed; inset: 0; z-index: 30; display: none;
    align-items: center; justify-content: center; pointer-events: none;
    font: 20px Sarabun, 'Noto Sans Thai', system-ui, sans-serif; color: #fff;
    text-shadow: 0 1px 3px rgba(0, 0, 0, .8); letter-spacing: .5px;
  }""",
    """  #goro-loading {
    position: fixed; inset: 0; z-index: 30; display: none;
    flex-direction: column; align-items: center; justify-content: center; pointer-events: none;
    font: 20px Sarabun, 'Noto Sans Thai', system-ui, sans-serif; color: #fff;
    text-shadow: 0 1px 3px rgba(0, 0, 0, .8); letter-spacing: .5px;
  }
  #goro-loading .bar { display: none; flex-direction: column; align-items: center; margin-top: 14px; }
  #goro-loading .bar.show { display: flex; }
  #goro-loading .track { width: 260px; max-width: 70vw; height: 8px; border-radius: 4px; background: rgba(0, 0, 0, .45); overflow: hidden; }
  #goro-loading .fill { height: 100%; width: 0%; border-radius: 4px; background: #7db2e8; transition: width .15s ease-out; }
  #goro-loading .fill.indet {
    width: 38% !important;
    animation: goroindet 1.1s ease-in-out infinite alternate;
  }
  @keyframes goroindet { from { margin-left: 0; } to { margin-left: 62%; } }
  #goro-loading .plabel { font-size: 12px; margin-top: 7px; color: #cfe0f2; letter-spacing: .3px; }""",
)

# 2) route map pack progress to the loading label's bar (before the
#    boot/defer branches — otherwise map packs lit up the sound-pack bar)
sub1(
    """    window.goroPackProgress = function (file, loaded, total) {
      var bootBar = document.querySelector('#packbar');""",
    """    window.goroPackProgress = function (file, loaded, total) {
      if (file.indexOf('map_pack/') === 0) {
        var lb = document.getElementById('goro-loading');
        var bar = lb && lb.querySelector('.bar');
        if (!bar) return;
        bar.classList.add('show');
        var plbl = bar.querySelector('.plabel');
        if (plbl) plbl.textContent = 'map pack  ' + fmtMB(loaded) + (total > 0 ? ' / ' + fmtMB(total) : '');
        var fill = bar.querySelector('.fill');
        if (fill) {
          fill.classList.toggle('indet', !(total > 0));
          if (total > 0) fill.style.width = Math.min(100, loaded / total * 100).toFixed(1) + '%';
        }
        return;
      }
      var bootBar = document.querySelector('#packbar');""",
)
sub1(
    """    window.goroPackDone = function (file, ok) {
      var isBoot = file === 'data_web1.grf' || file === 'data_web.grf';""",
    """    window.goroPackDone = function (file, ok) {
      if (file.indexOf('map_pack/') === 0) {
        var lb = document.getElementById('goro-loading');
        var bar = lb && lb.querySelector('.bar');
        if (bar) {
          bar.classList.remove('show');
          var fill = bar.querySelector('.fill');
          if (fill) { fill.classList.remove('indet'); fill.style.width = '0%'; }
        }
        return;
      }
      var isBoot = file === 'data_web1.grf' || file === 'data_web.grf';""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
