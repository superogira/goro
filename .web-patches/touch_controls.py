import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:80]); sys.exit(1)
    src = src.replace(old, new, 1)

# 1) cam-cluster toggle + touch layer + stick visuals
sub1(
    """    <button data-fs title="Fullscreen">&#x26F6;</button>
    <button class="toggle" title="Show/hide controls">&#187;</button>
  </div>""",
    """    <button data-fs title="Fullscreen">&#x26F6;</button>
    <button id="goro-touch-toggle" title="Touch controls">&#127918;</button>
    <button class="toggle" title="Show/hide controls">&#187;</button>
  </div>
  <div id="goro-touchpad">
    <button id="goro-touch-action" title="Pick nearest item / attack nearest monster (drag to move)"></button>
  </div>
  <div id="goro-stick"><div id="goro-stick-base"></div><div id="goro-stick-thumb"></div></div>""",
)

# 2) CSS after the cam rules
sub1(
    "  #goro-cam .toggle { font-size: 13px; }",
    """  #goro-cam .toggle { font-size: 13px; }
  #goro-cam #goro-touch-toggle.on { background: #10ef21; color: #143c00; }
  #goro-touchpad {
    position: fixed; left: 0; top: 0; width: 0; height: 0; z-index: 17; display: none;
  }
  #goro-touchpad.on { display: block; }
  #goro-touch-action {
    position: fixed; right: 60px; bottom: 96px;
    width: 74px; height: 74px; border-radius: 50%;
    border: 2px solid rgba(80, 70, 50, .55); background: rgba(232, 223, 200, .92);
    color: #2a2622; font: 30px/1 Sarabun, system-ui, sans-serif;
    box-shadow: 0 3px 10px rgba(0, 0, 0, .35); touch-action: manipulation;
    -webkit-user-select: none; user-select: none;
  }
  #goro-touch-action:active { background: #d8cba8; transform: scale(.94); }
  #goro-touch-action::after {
    content: ''; display: block; width: 14px; height: 14px; margin: 0 auto;
    border-radius: 50%; background: rgba(80, 70, 50, .35);
  }
  #goro-stick { position: fixed; left: 0; top: 0; z-index: 18; pointer-events: none; display: none; }
  #goro-stick-base {
    position: absolute; width: 108px; height: 108px; border-radius: 50%;
    border: 2px solid rgba(255, 255, 255, .5); background: rgba(20, 20, 25, .28);
    transform: translate(-50%, -50%);
  }
  #goro-stick-thumb {
    position: absolute; width: 48px; height: 48px; border-radius: 50%;
    background: rgba(232, 223, 200, .85); border: 2px solid rgba(80, 70, 50, .5);
    transform: translate(-50%, -50%);
  }""",
)

# 3) the touch-control script after the cam toggle wiring
CAM_ANCHOR = """    cam.querySelector('.toggle').addEventListener('pointerdown', function (ev) {
      ev.preventDefault();
      camHidden = !camHidden;
      applyCamVisibility();
    });"""

JS = CAM_ANCHOR + """

    // ---- Touch controls: floating stick + action button (toggle \\U0001F3AE) ----
    // When enabled, touch pointers on the game canvas are intercepted at
    // window capture phase (before the wasm input bridge's canvas
    // listeners): a short tap is re-dispatched as a synthetic click so
    // direct touch on monsters/items/NPCs keeps working, while a drag past
    // the threshold becomes a floating virtual stick that steers the
    // character. The action button targets the nearest item/monster around
    // the player; its reach follows the in-game Snap Radius setting.
    (function () {
      var toggleBtn = document.getElementById('goro-touch-toggle');
      var pad = document.getElementById('goro-touchpad');
      var actionBtn = document.getElementById('goro-touch-action');
      var stick = document.getElementById('goro-stick');
      var stickBase = document.getElementById('goro-stick-base');
      var stickThumb = document.getElementById('goro-stick-thumb');
      var enabled = false;
      var drag = null;          // active stick touch: {id, ox, oy}
      var DRAG_START_PX = 22;   // beyond this a touch becomes a stick drag
      var STICK_MAX_PX = 54;    // thumb travel clamp
      var sendHold = null;      // interval refreshing the stick vector

      function sendStick(dx, dy) {
        if (window.goroTouchAction) window.goroTouchAction('stick:' + dx.toFixed(2) + ',' + dy.toFixed(2));
      }
      function showStick(x, y) {
        stick.style.display = 'block';
        stickBase.style.left = x + 'px'; stickBase.style.top = y + 'px';
        stickThumb.style.left = x + 'px'; stickThumb.style.top = y + 'px';
      }
      function moveThumb(x, y) {
        stickThumb.style.left = x + 'px'; stickThumb.style.top = y + 'px';
      }
      function endDrag() {
        if (!drag) return;
        drag = null;
        stick.style.display = 'none';
        if (sendHold) { clearInterval(sendHold); sendHold = null; }
        sendStick(0, 0);
      }
      function applyToggle(on) {
        enabled = on;
        try { localStorage.setItem('goroTouchPad', on ? '1' : '0'); } catch (e) {}
        toggleBtn.classList.toggle('on', on);
        pad.classList.toggle('on', on);
        if (!on) endDrag();
      }
      toggleBtn.addEventListener('pointerdown', function (ev) {
        ev.preventDefault(); ev.stopPropagation();
        applyToggle(!enabled);
      });
      try { if (localStorage.getItem('goroTouchPad') === '1') applyToggle(true); } catch (e) {}

      // The action button doubles as its own move handle: a press that
      // moves past ACTION_DRAG_PX relocates the button (position saved),
      // a press released in place fires the action.
      var ACTION_DRAG_PX = 12;
      var btnDrag = null;
      function clampBtn(x, y) {
        var r = actionBtn.getBoundingClientRect();
        x = Math.min(Math.max(8, x), window.innerWidth - r.width - 8);
        y = Math.min(Math.max(8, y), window.innerHeight - r.height - 8);
        return [x, y];
      }
      function saveBtnPos(x, y) {
        try { localStorage.setItem('goroTouchPadPos', Math.round(x) + ',' + Math.round(y)); } catch (e) {}
      }
      (function restoreBtnPos() {
        try {
          var saved = localStorage.getItem('goroTouchPadPos');
          if (!saved) return;
          var parts = saved.split(',');
          var x = +parts[0], y = +parts[1];
          if (!isFinite(x) || !isFinite(y)) return;
          var clamped = clampBtn(x, y);
          actionBtn.style.left = clamped[0] + 'px';
          actionBtn.style.top = clamped[1] + 'px';
          actionBtn.style.right = 'auto';
          actionBtn.style.bottom = 'auto';
        } catch (e) {}
      })();
      actionBtn.addEventListener('pointerdown', function (ev) {
        ev.preventDefault(); ev.stopPropagation();
        try { actionBtn.setPointerCapture(ev.pointerId); } catch (e) {}
        var r = actionBtn.getBoundingClientRect();
        btnDrag = { id: ev.pointerId, ox: ev.clientX, oy: ev.clientY,
                    left: r.left, top: r.top, moved: false };
      });
      actionBtn.addEventListener('pointermove', function (ev) {
        if (!btnDrag || ev.pointerId !== btnDrag.id) return;
        ev.preventDefault(); ev.stopPropagation();
        var dx = ev.clientX - btnDrag.ox, dy = ev.clientY - btnDrag.oy;
        if (!btnDrag.moved && Math.hypot(dx, dy) < ACTION_DRAG_PX) return;
        btnDrag.moved = true;
        var clamped = clampBtn(btnDrag.left + dx, btnDrag.top + dy);
        actionBtn.style.left = clamped[0] + 'px';
        actionBtn.style.top = clamped[1] + 'px';
        actionBtn.style.right = 'auto';
        actionBtn.style.bottom = 'auto';
      });
      function onBtnUp(ev) {
        if (!btnDrag || ev.pointerId !== btnDrag.id) return;
        ev.preventDefault(); ev.stopPropagation();
        var wasMoved = btnDrag.moved;
        btnDrag = null;
        if (wasMoved) {
          var r = actionBtn.getBoundingClientRect();
          saveBtnPos(r.left, r.top);
          return;
        }
        if (window.goroTouchAction) window.goroTouchAction('action');
      }
      actionBtn.addEventListener('pointerup', onBtnUp);
      actionBtn.addEventListener('pointercancel', onBtnUp);
      actionBtn.addEventListener('contextmenu', function (ev) { ev.preventDefault(); });

      function synthesizeTap(x, y, type) {
        // Re-dispatch the tap straight at the canvas so the wasm input
        // bridge handles it like any other click. _goroTap lets our own
        // capture interceptor pass it through.
        var canvas = document.querySelector('canvas');
        if (!canvas) return;
        var rect = canvas.getBoundingClientRect();
        var cx = x - rect.left, cy = y - rect.top;
        var opts = {
          bubbles: false, cancelable: true, pointerId: 999001, pointerType: 'touch',
          isPrimary: true, clientX: x, clientY: y, button: 0, buttons: type === 'up' ? 0 : 1
        };
        var evDown = new PointerEvent('pointerdown', opts);
        evDown._goroTap = true;
        canvas.dispatchEvent(evDown);
        if (type === 'up') {
          var optsUp = Object.assign({}, opts, { buttons: 0 });
          var evUp = new PointerEvent('pointerup', optsUp);
          evUp._goroTap = true;
          canvas.dispatchEvent(evUp);
        }
      }

      window.addEventListener('pointerdown', function (ev) {
        if (!enabled || ev._goroTap) return;
        if (ev.pointerType !== 'touch') return;
        // Only touches that land on the game canvas become stick/tap
        // gestures; DOM UI elements keep their normal handling.
        if (!ev.target || ev.target.tagName !== 'CANVAS') return;
        ev.preventDefault(); ev.stopPropagation();
        drag = { id: ev.pointerId, ox: ev.clientX, oy: ev.clientY, sx: ev.clientX, sy: ev.clientY, moved: false };
      }, { capture: true });

      window.addEventListener('pointermove', function (ev) {
        if (!enabled || ev._goroTap) return;
        if (!drag || ev.pointerId !== drag.id) return;
        ev.preventDefault(); ev.stopPropagation();
        var dx = ev.clientX - drag.ox, dy = ev.clientY - drag.oy;
        if (!drag.moved) {
          if (Math.hypot(dx, dy) < DRAG_START_PX) return;
          drag.moved = true;
          showStick(drag.ox, drag.oy);
          // Keep the vector fresh for the wasm-side staleness timer.
          sendHold = setInterval(function () {
            if (drag && drag.moved) sendStick(drag.vx || 0, drag.vy || 0);
          }, 300);
        }
        var len = Math.hypot(dx, dy) || 1;
        var clamped = Math.min(len, STICK_MAX_PX);
        var nx = dx / len, ny = dy / len;
        drag.vx = nx * (clamped / STICK_MAX_PX);
        drag.vy = ny * (clamped / STICK_MAX_PX);
        moveThumb(drag.ox + nx * clamped, drag.oy + ny * clamped);
        sendStick(drag.vx, drag.vy);
      }, { capture: true });

      function onTouchEnd(ev) {
        if (!enabled || ev._goroTap) return;
        if (!drag || ev.pointerId !== drag.id) return;
        ev.preventDefault(); ev.stopPropagation();
        if (!drag.moved) {
          // Short tap on the canvas: replay it as a normal click so direct
          // touch attack/pick/NPC/walk behavior is unchanged.
          synthesizeTap(ev.clientX, ev.clientY, 'up');
        }
        endDrag();
      }
      window.addEventListener('pointerup', onTouchEnd, { capture: true });
      window.addEventListener('pointercancel', onTouchEnd, { capture: true });
      window.addEventListener('contextmenu', function (ev) {
        if (enabled && drag) ev.preventDefault();
      }, { capture: true });
    })();"""

sub1(CAM_ANCHOR, JS)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok")
