import io, sys

# DOM twin of the Create Party window (#goro-partycreate): party name
# field plus Item Pickup / Item Sharing radios, family styling. Actions
# flow back through goroPartyCreateAction (ok\x1fname\x1fpickup\x1fdivision
# or cancel). Reuses the .frm/.lb/.finput/.btns styles the chat room
# create form ships. Idempotent.

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

if "goroPartyCreateSync" in src:
    print("party create already patched")
    sys.exit(0)

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# 1) markup
sub1(
    '  <div id="goro-chatroom-create" class="goro-social" style="left:50%;top:40%;transform:translate(-50%,-50%)"></div>',
    '''  <div id="goro-chatroom-create" class="goro-social" style="left:50%;top:40%;transform:translate(-50%,-50%)"></div>
  <div id="goro-partycreate" class="goro-social" style="left:50%;top:40%;transform:translate(-50%,-50%)"></div>''',
)

# 2) JS
sub1(
    "  // ---- DOM chat room windows ----",
    """  // ---- DOM party create (goroPartyCreateSync) ----
  (function () {
    function esc(s) { return String(s == null ? '' : s).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }
    function stop(el) {
      ['pointerdown', 'pointerup', 'click', 'wheel'].forEach(function (k) {
        el.addEventListener(k, function (ev) { ev.stopPropagation(); });
      });
    }
    function wireDrag(panel, title) {
      var drag = null;
      title.addEventListener('pointerdown', function (ev) {
        if (ev.target.closest('.x')) return;
        var r = panel.getBoundingClientRect();
        if (panel.style.transform) { panel.style.transform = 'none'; panel.style.left = r.left + 'px'; panel.style.top = r.top + 'px'; }
        drag = { x: ev.clientX - r.left, y: ev.clientY - r.top };
        try { title.setPointerCapture(ev.pointerId); } catch (e) {}
        ev.preventDefault();
      });
      title.addEventListener('pointermove', function (ev) {
        if (!drag) return;
        panel.style.left = Math.max(0, Math.min(ev.clientX - drag.x, window.innerWidth - panel.offsetWidth)) + 'px';
        panel.style.top = Math.max(0, Math.min(ev.clientY - drag.y, window.innerHeight - panel.offsetHeight)) + 'px';
      });
      title.addEventListener('pointerup', function () { drag = null; });
      title.addEventListener('pointercancel', function () { drag = null; });
    }
    var US = '\\u001f';
    var pc = document.getElementById('goro-partycreate');
    var pcOpen = false;
    function pcAct(a) { if (window.goroPartyCreateAction) window.goroPartyCreateAction(a); }
    function pcBuild() {
      pc.innerHTML = '<div class="title">Create Party<div class="x">\\u2715</div></div>' +
        '<div class="frm"><span class="lb">Party Name</span><input class="finput" data-k="name" maxlength="23" placeholder="Party Name"></div>' +
        '<div class="frm"><span class="lb">Item Pickup</span>' +
        '<label style="margin:0"><input type="radio" name="pc-pickup" value="0" checked> Each Take</label>' +
        '<label style="margin:0"><input type="radio" name="pc-pickup" value="1"> Party Share</label></div>' +
        '<div class="frm"><span class="lb">Item Sharing</span>' +
        '<label style="margin:0"><input type="radio" name="pc-share" value="0" checked> Individual</label>' +
        '<label style="margin:0"><input type="radio" name="pc-share" value="1"> Shared</label></div>' +
        '<div class="btns"><button class="ok">OK</button><button class="no">Cancel</button></div>';
      stop(pc);
      wireDrag(pc, pc.querySelector('.title'));
      function submit() {
        var name = pc.querySelector('[data-k="name"]').value.trim();
        if (!name) { pc.querySelector('[data-k="name"]').focus(); return; }
        var pickup = pc.querySelector('input[name="pc-pickup"]:checked');
        var share = pc.querySelector('input[name="pc-share"]:checked');
        pcAct('ok' + US + name + US + (pickup ? pickup.value : '0') + US + (share ? share.value : '0'));
      }
      var nameInput = pc.querySelector('[data-k="name"]');
      ['keydown', 'keyup', 'keypress'].forEach(function (k) {
        nameInput.addEventListener(k, function (ev) {
          ev.stopPropagation();
          if (ev.key === 'Enter') { ev.preventDefault(); submit(); }
          if (ev.key === 'Escape') { ev.preventDefault(); pcAct('cancel'); }
        });
      });
      nameInput.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
      pc.querySelectorAll('input[type="radio"]').forEach(function (i) {
        i.addEventListener('pointerdown', function (ev) { ev.stopPropagation(); });
      });
      pc.querySelector('.ok').addEventListener('pointerdown', function () { submit(); });
      pc.querySelector('.no').addEventListener('pointerdown', function () { pcAct('cancel'); });
      pc.querySelector('.x').addEventListener('pointerdown', function () { pcAct('cancel'); });
      nameInput.focus();
    }
    window.goroPartyCreateSync = function (d) {
      var open = !!d.open;
      if (open && !pcOpen) { pcBuild(); pc.style.display = 'block'; }
      else if (!open && pcOpen) { pc.style.display = 'none'; pc.innerHTML = ''; }
      pcOpen = open;
    };
  })();
  // ---- DOM chat room windows ----""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("party create patched, %d -> %d bytes" % (len(orig), len(src)))
