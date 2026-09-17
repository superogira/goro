import io, sys

# Button depth: every DOM button gets a subtle drop shadow toward the
# bottom-right, and the pressed state flattens + nudges it for a tactile
# feel. Idempotent.

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

if "#button-shadow" in src:
    print("button shadow already patched")
    sys.exit(0)

ANCHOR = "</style>"
if src.count(ANCHOR) < 1:
    print("style anchor missing"); sys.exit(1)

BLOCK = """  /* #button-shadow: pressed depth for DOM buttons */
  .goro-social .btns button, .goro-social .rowbtn, .goro-social .chatrow button,
  .goro-social .foot button, #goro-bmenu button, #goro-option .btns button, #goro-login .foot button,
  #goro-service .foot button, #goro-alert .foot button, #goro-invdrop button,
  .giw .viewbtn, #goro-charselect button, #goro-charcreate button,
  #goro-saraban button, #goro-settings button {
    box-shadow: 1px 2px 3px rgba(46, 38, 20, .38);
    transition: box-shadow .06s ease, transform .06s ease;
  }
  .goro-social .btns button:active, .goro-social .rowbtn:active,
  .goro-social .chatrow button:active, .goro-social .foot button:active,
  #goro-bmenu button:active,
  #goro-option .btns button:active, #goro-login .foot button:active,
  #goro-service .foot button:active, #goro-alert .foot button:active,
  #goro-invdrop button:active, .giw .viewbtn:active,
  #goro-charselect button:active, #goro-charcreate button:active,
  #goro-saraban button:active, #goro-settings button:active {
    box-shadow: 0 0 1px rgba(46, 38, 20, .3);
    transform: translate(1px, 1px);
  }
</style>"""

src = src.replace(ANCHOR, BLOCK, 1)
io.open(path, "w", encoding="utf-8", newline="").write(src)
print("button shadow patched, %d -> %d bytes" % (len(orig), len(src)))
