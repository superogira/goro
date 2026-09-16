import io, sys

# Unify every DOM panel title bar on the light family style used by the
# Friends window: gradient #d6e8fa -> #b8d6f2, dark blue text, #76a0ce
# bottom border and parchment close chips. Idempotent: safe to re-run.

OLD_GRAD = "linear-gradient(180deg, #9cc9f0 0%, #5b95d4 52%, #3d72b5 100%)"
NEW_GRAD = "linear-gradient(180deg, #d6e8fa, #b8d6f2)"

REPLACEMENTS = [
    # title gradients
    (OLD_GRAD, NEW_GRAD),
    # white-on-blue title text (single- and two-line spellings)
    ("color: #fff; text-shadow: 0 1px 1px rgba(15, 35, 65, .55);", "color: #163658;"),
    ("color: #fff;\n    text-shadow: 0 1px 1px rgba(15, 35, 65, .55);", "color: #163658;"),
    ("color: #fff;\n    text-shadow: 0 1px 1px rgba(15, 35, 65, .55);\n    font-weight: 600;", "color: #163658;\n    font-weight: 600;"),
    # NPC dialog header text
    ("#goro-npc .bar .t { color: #fff; }", "#goro-npc .bar .t { color: #163658; }"),
    # NPC dialog header border
    ("border-bottom: 1px solid rgba(46, 84, 130, .8);", "border-bottom: 1px solid #76a0ce;"),
    # glass close chips on the old blue bar -> parchment chips
    ("background: rgba(255, 255, 255, .28);\n    color: #fff; text-shadow: 0 1px 1px rgba(15, 35, 65, .55);",
     "background: rgba(232, 223, 200, .7);\n    color: #6a6055;"),
    ("background: rgba(255, 255, 255, .28); color: #fff; cursor: pointer;",
     "background: rgba(232, 223, 200, .7); color: #6a6055; cursor: pointer;"),
    ("color: #6a6055; cursor: pointer; text-align: center; line-height: 15px;
    text-shadow: 0 1px 1px rgba(15, 35, 65, .55);",
     "color: #6a6055; cursor: pointer; text-align: center; line-height: 15px;"),
    (".x:active { background: rgba(255, 255, 255, .5); }", ".x:active { background: #d8cba8; }"),
    ("#goro-hud .x:active, #goro-stats .x:active, #goro-inventory .x:active,\n  #goro-skills .x:active, #goro-equip .x:active, #goro-iteminfo .x:active { background: rgba(255, 255, 255, .5); }",
     "#goro-hud .x:active, #goro-stats .x:active, #goro-inventory .x:active,\n  #goro-skills .x:active, #goro-equip .x:active, #goro-iteminfo .x:active { background: #d8cba8; }"),
]

UNIFY_BLOCK = """
  /* Family title bars: one look for every DOM window (title_unify). */
  #goro-hud .title, #goro-stats .title, #goro-inventory .title,
  #goro-skills .title, #goro-equip .title, #goro-iteminfo .title, .giw .title,
  #goro-option .title, #goro-settings .title, #goro-login .title,
  #goro-service .title, #goro-alert .title, #goro-npc .bar,
  #goro-invdrop > div:first-child {
    border-bottom: 1px solid #76a0ce;
  }
"""

path = r"C:\laragon\www\webro\index.html"
src = io.open(path, encoding="utf-8").read()
orig = src

for old, new in REPLACEMENTS:
    if old in src:
        src = src.replace(old, new)

if "Family title bars: one look for every DOM window" not in src:
    anchor = "  /* DOM social windows */"
    if anchor not in src:
        print("anchor for unify block not found"); sys.exit(1)
    src = src.replace(anchor, UNIFY_BLOCK + anchor, 1)

io.open(path, "w", encoding="utf-8", newline="").write(src)
print("title unify applied, %d -> %d bytes (old gradient left: %d)" % (
    len(orig), len(src), src.count(OLD_GRAD)))
