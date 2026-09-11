import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new, count=1):
    global src
    if src.count(old) < count:
        print("ANCHOR NOT FOUND:", old[:90]); sys.exit(1)
    src = src.replace(old, new, count)

# 1) CSS: prerequisite highlight + level badge (mirrors the canvas grid's
#    pink SkillRequired slot and bottom-right level badge)
sub1(
    "  #goro-skills .gitem.unlearned .gnm { color: #a49c92; }",
    """  #goro-skills .gitem.unlearned .gnm { color: #a49c92; }
  #goro-skills .gitem.reqhl .gslot, #goro-skills .srow.reqhl { background: #ffc0cb; }
  #goro-skills .gitem.reqhl .reqbadge, #goro-skills .srow .reqbadge {
    position: absolute; right: 0; bottom: 15px; min-width: 13px; height: 13px;
    padding: 0 2px; box-sizing: border-box; border-radius: 3px;
    background: #3d72b5; color: #fff; font: 9px/13px Sarabun,system-ui,sans-serif;
    text-align: center; pointer-events: none;
  }
  #goro-skills .srow { position: relative; }
  #goro-skills .srow .reqbadge { right: 4px; top: 50%; bottom: auto; transform: translateY(-50%); }""",
)

# 2) grid cells carry their prerequisite list as JSON
sub1(
    """          html += '<div class="gitem ' + (c.canStage ? 'canstage' : '') + (c.learned ? '' : ' unlearned') + '"' +
            ' style="grid-column:' + col + ';grid-row:' + row + ';" data-id="' + c.id + '">' +""",
    """          html += '<div class="gitem ' + (c.canStage ? 'canstage' : '') + (c.learned ? '' : ' unlearned') + '"' +
            ' style="grid-column:' + col + ';grid-row:' + row + ';" data-id="' + c.id + '" data-req="' + esc(JSON.stringify(c.req || [])) + '">' +""",
)

# 3) hover wiring: highlight prerequisite cells with level badges
sub1(
    """        node.addEventListener('mouseenter', function (ev) { showTip(data.tip, ev.clientX, ev.clientY); });""",
    """        node.addEventListener('mouseenter', function (ev) {
          showTip(data.tip, ev.clientX, ev.clientY);
          applyReq(node);
        });""",
)
sub1(
    """        node.addEventListener('mouseleave', hideTip);""",
    """        node.addEventListener('mouseleave', function () { hideTip(); clearReq(); });""",
)
sub1(
    """    window.goroSkillSync = function (s) {""",
    """    // Hover prerequisites: highlight every transitive prerequisite of the
    // hovered skill across the current view and badge the level it needs —
    // the DOM twin of the canvas grid's pink SkillRequired slots.
    function applyReq(node) {
      clearReq();
      var reqs;
      try { reqs = JSON.parse(node.getAttribute('data-req') || '[]'); } catch (e) { return; }
      var view = el.querySelector('.view');
      if (!view) return;
      reqs.forEach(function (r) {
        var target = view.querySelector('[data-id="' + r.id + '"]');
        if (!target) return;
        target.classList.add('reqhl');
        if (r.level > 0) {
          var b = document.createElement('span');
          b.className = 'reqbadge';
          b.textContent = r.level;
          target.appendChild(b);
        }
      });
    }
    function clearReq() {
      el.querySelectorAll('.reqhl').forEach(function (n) {
        n.classList.remove('reqhl');
        var b = n.querySelector('.reqbadge');
        if (b) b.remove();
      });
    }
    window.goroSkillSync = function (s) {""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
