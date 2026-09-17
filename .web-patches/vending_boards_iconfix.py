import io, sys

# Vending board icon fix: shop.bmp uses magenta as the RO chroma-key for
# transparency, which the game's loader converts to alpha — but a plain
# <img> renders the raw BMP, so DOM boards showed a pink/purple icon.
# The page now decodes the bitmap once through a canvas, zeroes the
# magenta pixels (same rule as res.isROMagenta) and caches the data URL.
# Run after vending_boards_dom.py. Idempotent.

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

if "boardIconMagentaKey" in src:
    print("vending icon fix already applied")
    sys.exit(0)

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# 1) replace the direct-BMP URL with the chroma-keyed canvas pipeline
sub1(
    "    var iconURL = encodeURI('data/texture/\\uc720\\uc800\\uc778\\ud130\\ud398\\uc774\\uc2a4/basic_interface/shop.bmp');",
    """    // boardIconMagentaKey: shop.bmp keys transparency on magenta (the
    // same rule as the game's res.isROMagenta); decode once, cache.
    var iconURL = '';
    var iconQueue = [];
    var iconLoading = false;
    function boardIconMagentaKey() {
      if (iconURL || iconLoading) return;
      iconLoading = true;
      var img = new Image();
      img.onload = function () {
        var cv = document.createElement('canvas');
        cv.width = img.naturalWidth || 24;
        cv.height = img.naturalHeight || 24;
        var cx = cv.getContext('2d');
        cx.drawImage(img, 0, 0);
        var d;
        try { d = cx.getImageData(0, 0, cv.width, cv.height); }
        catch (e) { iconLoading = false; return; }
        var p = d.data;
        for (var i = 0; i < p.length; i += 4) {
          if (p[i] >= 248 && p[i + 1] <= 8 && p[i + 2] >= 248) p[i + 3] = 0;
        }
        cx.putImageData(d, 0, 0);
        iconURL = cv.toDataURL();
        iconQueue.forEach(function (im) { im.src = iconURL; });
        iconQueue = [];
      };
      img.onerror = function () { iconLoading = false; };
      img.src = encodeURI('data/texture/\\uc720\\uc800\\uc778\\ud130\\ud398\\uc774\\uc2a4/basic_interface/shop.bmp');
    }
    function boardIcon(el) {
      var im = el.querySelector('img');
      if (iconURL) { im.src = iconURL; return; }
      boardIconMagentaKey();
      iconQueue.push(im);
    }""",
)

# 2) create the <img> without a src and route it through the pipeline
sub1(
    "          el.innerHTML = '<img alt=\"\" src=\"' + iconURL + '\"><span></span>';",
    """          el.innerHTML = '<img alt=""><span></span>';
          boardIcon(el);""",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("vending icon fix applied, %d -> %d bytes" % (len(orig), len(src)))
