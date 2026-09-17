import io, sys

# Chat console scrolling: the log (.clines) rendered overflow:hidden with
# only the latest lines, so old messages were unreachable once the panel
# expanded. The log now scrolls (wheel/touch), shows the full history the
# game already sends, sticks to the bottom on new lines while the reader
# is at the end, keeps the scroll position when the reader scrolled up,
# and shows a "new messages" pill until they return to the bottom.
# Idempotent.

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

if "goro-console .clines {-scrollable" in src or "#console-scroll" in src:
    print("console scroll already patched")
    sys.exit(0)

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# 1) CSS (end of the style block so it wins the cascade)
sub1(
    "</style>",
    """  /* #console-scroll: chat log history */
  #goro-console { flex-direction: column; }
  #goro-console .clines {
    flex: 1 1 auto; min-height: 0;
    overflow-y: auto; touch-action: pan-y; overscroll-behavior: contain;
    scrollbar-width: thin; scrollbar-color: rgba(180, 198, 218, .45) transparent;
  }
  #goro-console .clines::-webkit-scrollbar { width: 7px; }
  #goro-console .clines::-webkit-scrollbar-thumb { background: rgba(180, 198, 218, .4); border-radius: 4px; }
  #goro-console .clines::-webkit-scrollbar-track { background: transparent; }
  #goro-console .newmsg {
    position: absolute; right: 14px; bottom: 44px; display: none; z-index: 2;
    padding: 2px 10px; border-radius: 10px; cursor: pointer;
    background: rgba(74, 138, 202, .92); color: #fff;
    font-size: 11px; font-weight: 600; line-height: 1.6;
  }
  #goro-console .newmsg.show { display: block; }
</style>""",
)

# 2) markup: drop the inline overflow/flex-end (flex-end makes overflowing
#    content unreachable at the top of a scroll box) and add the badge
sub1(
    """el.innerHTML = '<div class="clines" style="display:flex;flex-direction:column;justify-content:flex-end;overflow:hidden"></div>' +
        '<div class="inrow"><input id="goro-chat-in" type="text" maxlength="80" autocomplete="off" placeholder="Type and press Enter..."></div>';""",
    """el.innerHTML = '<div class="clines" style="display:flex;flex-direction:column"></div>' +
        '<div class="inrow"><input id="goro-chat-in" type="text" maxlength="80" autocomplete="off" placeholder="Type and press Enter..."></div>' +
        '<div class="newmsg">New messages \\u2193</div>';
      var clines = el.querySelector('.clines');
      ['wheel', 'mousewheel', 'DOMMouseScroll'].forEach(function (k) {
        clines.addEventListener(k, function (ev) { ev.stopPropagation(); });
      });
      clines.addEventListener('scroll', function () {
        if (clines.scrollHeight - clines.scrollTop - clines.clientHeight < 12) {
          var b = el.querySelector('.newmsg'); if (b) b.classList.remove('show');
        }
      });
      var badge = el.querySelector('.newmsg');
      badge.addEventListener('pointerdown', function (ev) {
        ev.stopPropagation(); ev.preventDefault();
        badge.classList.remove('show');
        clines.scrollTop = clines.scrollHeight;
      });""",
)

# 3) renderLines: preserve the reader's scroll offset across the rebuild,
#    stick to the bottom when they were at the end, and flag new content
#    for the badge when they were reading history
sub1(
    """function renderLines(lines, count) {
      var frag = document.createDocumentFragment();
      for (var i = Math.max(0, lines.length - count); i < lines.length; i++) {
        var d = document.createElement('div');
        d.className = 'line';
        d.textContent = lines[i][0];
        d.style.color = lines[i][1];
        frag.appendChild(d);
      }
      var box = el.querySelector('.clines');
      box.textContent = '';
      box.appendChild(frag);
    }""",
    """var consoleLastSig = '';
    function renderLines(lines, count) {
      var frag = document.createDocumentFragment();
      for (var i = Math.max(0, lines.length - count); i < lines.length; i++) {
        var d = document.createElement('div');
        d.className = 'line';
        d.textContent = lines[i][0];
        d.style.color = lines[i][1];
        frag.appendChild(d);
      }
      var box = el.querySelector('.clines');
      var badge = el.querySelector('.newmsg');
      var active = el.classList.contains('active');
      var stick = !active || box.scrollHeight - box.scrollTop - box.clientHeight < 12;
      var prevTop = box.scrollTop;
      var sig = lines.length + ':' + (lines.length ? lines[lines.length - 1][0] : '');
      box.textContent = '';
      box.appendChild(frag);
      if (!active) { if (badge) badge.classList.remove('show'); consoleLastSig = sig; return; }
      if (stick) {
        box.scrollTop = box.scrollHeight;
        if (badge) badge.classList.remove('show');
      } else {
        box.scrollTop = prevTop;
        if (sig !== consoleLastSig && badge) badge.classList.add('show');
      }
      consoleLastSig = sig;
    }""",
)

# 4) active mode renders the whole history the game already sends
sub1(
    """        renderLines(lines, 12);""",
    """        renderLines(lines, lines.length);""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("console scroll patched, %d -> %d bytes" % (len(orig), len(src)))
