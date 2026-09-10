import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new, count=1):
    global src
    if src.count(old) < count:
        print("ANCHOR NOT FOUND:", old[:90]); sys.exit(1)
    src = src.replace(old, new, count)

# 1) chat lines wrap instead of ellipsizing — long messages stay fully
#    readable; the container clips the oldest lines at the top
sub1(
    "  #goro-console div.line { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }",
    "  #goro-console div.line { white-space: normal; word-break: break-word; overflow-wrap: anywhere; }",
)

# 2) typing console sits above every gameplay window (stats 21, menu 16,
#    item info 30) so the lifted panel is never covered while typing
sub1(
    "  #goro-console.active { bottom: 32px; max-height: 236px; }",
    "  #goro-console.active { bottom: 32px; max-height: 236px; z-index: 45; }",
)

# 3) drop the legacy bottom-shift pan — the shared transform lift owns
#    keyboard avoidance now, and running both moved the panel twice
sub1(
    """      // Keep the panel above the tablet keyboard while typing.
      var panFor = function () {
        var kb = keyboardHeight();
        el.style.bottom = (kb > 0 ? Math.min(kb + 8, window.innerHeight * 0.6) : 32) + 'px';
      };
      input.addEventListener('focus', panFor);
      input.addEventListener('blur', function () { setTimeout(panFor, 150); });
      if (window.visualViewport) {
        window.visualViewport.addEventListener('resize', function () {
          if (document.activeElement === input) panFor();
        });
      }
    }""",
    "    }",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
