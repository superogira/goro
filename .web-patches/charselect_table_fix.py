import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub(old, new, count=1):
    global src
    if src.count(old) != count:
        print("ANCHOR count %d != %d:" % (src.count(old), count), old[:90]); sys.exit(1)
    src = src.replace(old, new, count)

# Stats table adjustments: value cells become white with black text, both
# label columns share one width and both value columns share one width
# (fixed table layout splits the remaining space evenly).

# 1) CSS: fixed layout, equal label widths, white/black value cells.
sub(
    "  #goro-charselect .info table { width: 100%; border-collapse: collapse; font-size: 12px; }",
    "  #goro-charselect .info table { width: 100%; border-collapse: collapse; font-size: 12px; table-layout: fixed; }",
)
sub(
    "  #goro-charselect .info td.h { background: #deedfc; color: #3a689c; width: 44px; }\n  #goro-charselect .info td.s { background: #deedfc; color: #3a689c; width: 32px; }",
    """  #goro-charselect .info td.h { background: #deedfc; color: #3a689c; width: 54px; }
  #goro-charselect .info td.s { background: #deedfc; color: #3a689c; width: 54px; }""",
)
sub(
    "  #goro-charselect .info td.v { text-align: right; color: #fff; }",
    "  #goro-charselect .info td.v { text-align: right; background: #fff; color: #1a1a1a; }",
)

# 2) JS: the left value column (Name/Job/Level/Exp/HP/SP values) gets the
# same class as the right one so both value columns style identically.
sub("</td><td>' + esc(sel.", "</td><td class=\"v\">' + esc(sel.", 6)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
