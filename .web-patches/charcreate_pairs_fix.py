import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# Stat pair labels for the make-character table: the Go order is
# STR, AGI, VIT, INT, DEX, LUK and the real pairs are STR-INT, AGI-LUK,
# VIT-DEX. The DEX and LUK entries were swapped, so pressing + on DEX said
# it draws from AGI (it actually takes VIT) and LUK said VIT (takes AGI).
# Only the labels were wrong; bumpCreateStat always paired correctly.

sub1(
    "    var STAT_PAIRS = ['INT', 'LUK', 'DEX', 'STR', 'AGI', 'VIT'];",
    "    var STAT_PAIRS = ['INT', 'LUK', 'DEX', 'STR', 'VIT', 'AGI'];",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
