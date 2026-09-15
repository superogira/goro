import io, sys

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

def sub1(old, new):
    global src
    if src.count(old) != 1:
        print("ANCHOR count != 1:", old[:90]); sys.exit(1)
    src = src.replace(old, new, 1)

# The empty-slot message area gets a white background (matching the value
# cells) instead of showing the info panel's navy.

sub1(
    "  #goro-charselect .info .empty { text-align: center; padding: 18px 8px; color: #8a7f6a; }",
    "  #goro-charselect .info .empty { text-align: center; padding: 18px 8px; color: #8a7f6a; background: #fff; }",
)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("patched ok, %d -> %d bytes" % (len(orig), len(src)))
