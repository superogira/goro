import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# Multiple independent DOM item-info panels (upstream "Multiple window
# instances" semantics, DOM edition): descriptions and card artwork open as
# separate windows, each with its own snapshot, drag, close, and raise;
# clicking a slotted card opens that card's description; a card
# description's View button opens its artwork panel; Escape closes the
# topmost panel first.

# 1) CSS: generalize the single #goro-iteminfo panel to a .giw class and
#    add the View button + artwork panel styles.
sub1(
    """  #goro-iteminfo {
    position: fixed; z-index: 21; display: none; width: 230px;
    background: rgba(255, 255, 253, .96); border: 1px solid rgba(120, 110, 90, .55);
    border-radius: 8px; padding: 6px 10px 9px; user-select: none;
    font: 12px/1.35 Sarabun, 'Noto Sans Thai', system-ui, sans-serif; color: #2a2622;
  }""",
    """  .giw {
    position: fixed; z-index: 21; width: 230px; box-sizing: border-box;
    background: rgba(255, 255, 253, .96); border: 1px solid rgba(120, 110, 90, .55);
    border-radius: 8px; padding: 6px 10px 9px; user-select: none;
    font: 12px/1.35 Sarabun, 'Noto Sans Thai', system-ui, sans-serif; color: #2a2622;
  }
  .giw.wide { width: 316px; }
  .giw img.art { display: block; width: 294px; height: 392px; object-fit: contain; image-rendering: auto; }
  .giw .viewrow { display: flex; justify-content: center; margin-top: 7px; }
  .giw .viewbtn {
    padding: 2px 16px; border-radius: 5px; border: 1px solid rgba(120,110,90,.5);
    background: rgba(232, 223, 200, .8); color: #2a2622; cursor: pointer;
    font: 11px/1.5 Sarabun, 'Noto Sans Thai', system-ui, sans-serif;
  }
  .giw .cslot.filled { cursor: pointer; }""",
)
for sel in [
    "  #goro-iteminfo .desc { color: #4a4440; }\n  #goro-iteminfo .cards { color: #7a5c20; }\n  #goro-iteminfo .x { background: rgba(120, 110, 90, .2); color: #6a6055; }\n",
    "  #goro-iteminfo .title { position: relative;",
    "  #goro-iteminfo .head { display: flex;",
    "  #goro-iteminfo .head img { width: 75px;",
    "  #goro-iteminfo .head .meta { flex: 1;",
    "  #goro-iteminfo .x { position: absolute;",
]:
    if src.count(sel) != 1:
        print("CSS selector anchor problem:", sel[:60]); sys.exit(1)
src = src.replace("  #goro-iteminfo .", "  .giw .")

# 2) markup: the static single panel is replaced by dynamically created panels
sub1('  <div id="goro-iteminfo"></div>\n', '')

# 3) JS: replace the single-panel builder with the multi-panel manager
sub1(
    """  // ---- DOM item info panel ----
  (function () {
    var panel = document.getElementById('goro-iteminfo');
    window.goroItemInfoSync = function (d) {""",
    """  // ---- DOM item info panels (multiple independent windows) ----
  (function () {
    var panels = {};
    var zTop = 21;
    function act(a) { if (window.goroItemInfoAction) window.goroItemInfoAction(a); }
    function raise(id) {
      var p = panels[id];
      if (!p) return;
      p.z = ++zTop;
      p.el.style.zIndex = p.z;
    }
    function closePanel(id) {
      var p = panels[id];
      if (!p) return;
      p.el.remove();
      delete panels[id];
      act('close:' + id);
    }
    function buildPanel(d) {
      var el = document.createElement('div');
      el.className = 'giw' + (d.kind === 'art' ? ' wide' : '');
      var n = d.seq || 0;
      var x = Math.min(window.innerWidth - 250, Math.round(window.innerWidth / 2) + 40 + (n % 6) * 26);
      var y = Math.max(8, Math.min(window.innerHeight - 120, Math.round(window.innerHeight / 2) - 80 + (n % 6) * 26));
      el.style.left = x + 'px';
      el.style.top = y + 'px';
      document.body.appendChild(el);
      panels[d.id] = { el: el, z: ++zTop };
      el.style.zIndex = panels[d.id].z;
    }
    window.goroItemInfoEscape = function () {
      var topId = -1, topZ = -1;
      Object.keys(panels).forEach(function (id) {
        if (panels[id].z > topZ) { topZ = panels[id].z; topId = +id; }
      });
      if (topId >= 0) {
        panels[topId].el.remove();
        delete panels[topId];
        return topId;
      }
      return -1;
    };
    window.goroItemInfoWindowsSync = function (list) {
      var seen = {};
      for (var i = 0; i < (list ? list.length : 0); i++) {
        var d = list[i];
        seen[d.id] = true;
        if (!panels[d.id]) buildPanel(d);
        renderPanel(d);
      }
      Object.keys(panels).forEach(function (id) {
        if (!seen[id]) { panels[id].el.remove(); delete panels[id]; }
      });
    };
    function renderPanel(d) {
      var p = panels[d.id];
      if (!p) return;
      var html;
      if (d.kind === 'art') {
        html = '<div class="title">' + (d.title || 'Card') + '<div class="x">\u2715</div></div>' +
          (d.art ? '<img class="art" src="' + d.art + '" draggable="false">' : '');
      } else {
        html = '<div class="title">' + (d.title || 'Item') + '<div class="x">\u2715</div></div>' +
          '<div class="head">' + (d.icon ? '<img src="' + d.icon + '">' : '') +
          '<div class="meta"><div class="desc">' + (d.descHTML || d.desc || '') + '</div></div></div>';
        var cards = d.cards || [];
        if (cards.length) html += '<div class="cards">' + cards.map(function (c) { return '\u2022 ' + c; }).join('<br>') + '</div>';
        var slots = d.slots || [];
        if (slots.length) {
          html += '<div class="slotsrow">';
          for (var s = 0; s < slots.length; s++) {
            var sl = slots[s];
            if (sl.state === 'card') {
              html += '<div class="cslot filled" data-slot="' + s + '" title="' + (sl.name || '') + ' \u2014 click for details">' +
                (sl.icon ? '<img src="' + sl.icon + '">' : '') + '</div>';
            } else if (sl.state === 'empty') {
              html += '<div class="cslot empty"></div>';
            } else {
              html += '<div class="cslot none"></div>';
            }
          }
          html += '</div>';
        }
        if (d.viewable) html += '<div class="viewrow"><button class="viewbtn">View</button></div>';
      }
      p.el.innerHTML = html;
      p.el.querySelector('.x').addEventListener('pointerdown', function (ev) {
        ev.stopPropagation();
        closePanel(d.id);
      });
      p.el.addEventListener('pointerdown', function () { raise(d.id); });
      p.el.querySelectorAll('.cslot.filled').forEach(function (slotEl) {
        slotEl.addEventListener('pointerdown', function (ev) {
          ev.stopPropagation();
          act('card:' + d.id + ':' + slotEl.getAttribute('data-slot'));
        });
      });
      var vb = p.el.querySelector('.viewbtn');
      if (vb) vb.addEventListener('pointerdown', function (ev) {
        ev.stopPropagation();
        act('view:' + d.id);
      });
      var drag = null;
      var title = p.el.querySelector('.title');
      title.addEventListener('pointerdown', function (ev) {
        if (ev.target.closest('.x')) return;
        drag = { x: ev.clientX - p.el.offsetLeft, y: ev.clientY - p.el.offsetTop, id: ev.pointerId };
        try { title.setPointerCapture(ev.pointerId); } catch (e) {}
        ev.preventDefault();
      });
      title.addEventListener('pointermove', function (ev) {
        if (!drag) return;
        p.el.style.left = Math.max(0, Math.min(ev.clientX - drag.x, window.innerWidth - p.el.offsetWidth)) + 'px';
        p.el.style.top = Math.max(0, Math.min(ev.clientY - drag.y, window.innerHeight - p.el.offsetHeight)) + 'px';
      });
      title.addEventListener('pointerup', function () { drag = null; });
      title.addEventListener('pointercancel', function () { drag = null; });
    }
    window.goroItemInfoSync = function (d) {""",
)

# 4) the legacy single-panel sync hook now feeds one instance through the
#    multi-panel renderer so old call sites keep working
sub1(
    """    window.goroItemInfoSync = function (d) {
      var html = '<div class="title">' + (d.title || 'Item') + '<div class="x">\u2715</div></div>' +
        '<div class="head">' + (d.icon ? '<img src="' + d.icon + '">' : '') +
        '<div class="meta"><div class="desc">' + (d.descHTML || d.desc || '') + '</div></div></div>';
      var cards = d.cards || [];
      if (cards.length) {
        html += '<div class="cards">' + cards.map(function (c) { return '\u2022 ' + c; }).join('<br>') + '</div>';
      }
      var slots = d.slots || [];
      if (slots.length) {
        html += '<div class="slotsrow">';
        for (var s = 0; s < slots.length; s++) {
          var sl = slots[s];
          if (sl.state === 'card') {
            html += '<div class="cslot filled" title="' + (sl.name || '') + '">' +
              (sl.icon ? '<img src="' + sl.icon + '">' : '') + '</div>';
          } else if (sl.state === 'empty') {
            html += '<div class="cslot empty"></div>';
          } else {
            html += '<div class="cslot none"></div>';
          }
        }
        html += '</div>';
      }
      panel.innerHTML = html;
      panel.style.display = 'block';
      // position near center-right, clamped
      var x = Math.min(window.innerWidth - 250, Math.round(window.innerWidth / 2) + 40);
      var y = Math.max(8, Math.round(window.innerHeight / 2) - 80);
      panel.style.left = x + 'px';
      panel.style.top = y + 'px';
      panel.querySelector('.x').addEventListener('pointerdown', function (ev) {
        ev.stopPropagation();
        panel.style.display = 'none';
      });
      // Draggable by the title bar.
      var drag = null;
      var title = panel.querySelector('.title');
      title.addEventListener('pointerdown', function (ev) {
        if (ev.target.closest('.x')) return;
        drag = { x: ev.clientX - panel.offsetLeft, y: ev.clientY - panel.offsetTop, id: ev.pointerId };
        try { title.setPointerCapture(ev.pointerId); } catch (e) {}
        ev.preventDefault();
      });
      title.addEventListener('pointermove', function (ev) {
        if (!drag) return;
        panel.style.left = Math.max(0, Math.min(ev.clientX - drag.x, window.innerWidth - panel.offsetWidth)) + 'px';
        panel.style.top = Math.max(0, Math.min(ev.clientY - drag.y, window.innerHeight - panel.offsetHeight)) + 'px';
        panel.style.right = 'auto';
      });
      title.addEventListener('pointerup', function () { drag = null; });
    };""",
    """    window.goroItemInfoSync = function (d) {
      // Legacy single-panel entry: route through the instance renderer so
      // callers compiled before the multi-window manager still display.
      d.id = d.id || 999999;
      d.seq = d.seq || 0;
      d.kind = d.kind || 'desc';
      if (!panels[d.id]) buildPanel(d);
      renderPanel(d);
    };""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
