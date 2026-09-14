import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# Character select panel styling to match the canvas RO window family:
# - The panel title becomes the canvas title bar (vertical gradient
#   WindowTitleTop -> WindowTitle with the WindowBorder hairline below).
# - The stats table gets the classic RO look: label cells in the
#   selected-slot highlight color (#deedfc) with the theme's label blue,
#   value cells navy with white text (as in the canvas/original client).

sub1(
    "  #goro-charselect .title { text-align: center; font-weight: 600; margin-bottom: 8px; }",
    """  #goro-charselect .title {
    text-align: center; font-weight: 600;
    margin: -8px -14px 10px; padding: 6px 10px;
    background: linear-gradient(180deg, #d6e8fa, #b8d6f2);
    border-bottom: 1px solid #76a0ce;
    border-radius: 9px 9px 0 0;
    color: #163658;
  }""",
)
sub1(
    "  #goro-charselect .info {\n    margin: 0 auto; width: 318px; max-width: 100%; box-sizing: border-box;\n    border: 1px solid rgba(120, 110, 90, .45); border-radius: 6px; overflow: hidden;\n    background: rgba(255,255,255,.35);\n  }",
    """  #goro-charselect .info {
    margin: 0 auto; width: 318px; max-width: 100%; box-sizing: border-box;
    border: 1px solid #76a0ce; border-radius: 6px; overflow: hidden;
    background: #31506e;
  }""",
)
sub1(
    "  #goro-charselect .info td.h { color: #6a6055; width: 44px; }\n  #goro-charselect .info td.s { color: #6a6055; width: 32px; }",
    """  #goro-charselect .info td.h { background: #deedfc; color: #3a689c; width: 44px; }
  #goro-charselect .info td.s { background: #deedfc; color: #3a689c; width: 32px; }""",
)
sub1(
    "  #goro-charselect .info td.v { text-align: right; }",
    "  #goro-charselect .info td.v { text-align: right; color: #fff; }",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
