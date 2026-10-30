# rAthena Setup

Goro targets the classic pre-renewal `2008-09-10aSakexe` flow by default.
Current upstream rAthena already contains the old-client protocol fixes Goro used
to need as local patches, so the setup is now mostly a normal rAthena checkout
plus the remaining 2008 Sakray profile/configuration defaults.

## Recommended Source

Use the Goro-compatible rAthena branch:

```sh
git clone https://github.com/kivutar/rathena.git
cd rathena
git checkout goro
git pull --ff-only
```

The `goro` branch is rebased on current upstream rAthena and keeps only the
remaining local compatibility/configuration deltas:

- `PACKETVER=20080910`, `PACKETVER_SAK_NUM=20080910`, and `PRERE` defaults in
  `src/custom/defines_pre.hpp`
- the 2008 Sakexe packet DB gate that prevents early Renewal packet
  shuffles from overriding the Sakexe table
- local development config, NPC script selection, rates, and map caches

Upstream rAthena now already has the homunculus property packet support,
homunculus `0x022E` refresh fallback, EXP-notify packet gate, old skill cast ACK
fallback, old packetver cloth-color build fix, and legacy `0x006d` character
record layout.

Starting from upstream `rathena/rathena` is possible, but for the exact
`2008-09-10aSakexe` environment use the `goro` branch until the remaining
profile-specific packet/configuration deltas are upstream.

## Build

From the rAthena checkout:

```sh
./configure
make clean
make server
```

The `goro` branch already defines the 2008 Sakray packet profile in
`src/custom/defines_pre.hpp`, so `--enable-packetver=20080910` and
`--enable-prere` are not required there. If you maintain a separate checkout
with the remaining `goro` branch deltas, make sure these are defined before
building:

```cpp
#ifndef PACKETVER
	#define PACKETVER 20080910
#endif
#ifndef PACKETVER_SAK_NUM
	#define PACKETVER_SAK_NUM 20080910
#endif
#ifndef PRERE
	#define PRERE
#endif
```

## Required Config

The `goro` branch already carries these local defaults. If you use another
rAthena checkout, mirror them in `conf/char_athena.conf`:

```conf
char_del_delay: 0
char_del_option: 1
pincode_enabled: no
char_move_enabled: no
allowed_job_flag: 1
```

`char_del_delay: 0` is important: delayed character deletion is for newer
clients, while the 2008 client deletes directly after the email check.

For local-only testing, keep all server IPs on localhost:

```conf
// conf/char_athena.conf
login_ip: 127.0.0.1
char_ip: 127.0.0.1

// conf/map_athena.conf
char_ip: 127.0.0.1
map_ip: 127.0.0.1
```

For LAN testing, use the host IP that clients can reach:

```conf
// example: rAthena host is 192.168.1.169
char_ip: 192.168.1.169
map_ip: 192.168.1.169
```

Some useful convenience settings for testing:

```conf
// conf/login_athena.conf
new_account: yes

// conf/char_athena.conf
start_zeny: 100000
```

Starter zeny, automatic account creation, test NPCs, and GM accounts are
optional. They are not client compatibility requirements.

## MariaDB Import

For a fresh local install, create the main and log databases:

```sql
CREATE DATABASE rathena;
CREATE DATABASE log;
```

Import the base schemas:

```sh
mysql rathena < sql-files/main.sql
mysql rathena < sql-files/web.sql
mysql rathena < sql-files/roulette_default_data.sql
mysql log < sql-files/logs.sql
```

For pre-renewal SQL item/mob tables, also import:

```sh
mysql rathena < sql-files/item_db.sql
mysql rathena < sql-files/item_db_equip.sql
mysql rathena < sql-files/item_db_etc.sql
mysql rathena < sql-files/item_db_usable.sql
mysql rathena < sql-files/item_db2.sql
mysql rathena < sql-files/mob_db.sql
mysql rathena < sql-files/mob_db2.sql
mysql rathena < sql-files/mob_skill_db.sql
mysql rathena < sql-files/mob_skill_db2.sql
```

Refresh the ignored YAML import overrides from the current templates after a
fresh checkout or after rebasing the rAthena branch:

```sh
mkdir -p db/import
cp db/import-tmpl/item_group_db.yml db/import/item_group_db.yml
cp db/import-tmpl/item_packages.yml db/import/item_packages.yml
cp db/import-tmpl/job_stats.yml db/import/job_stats.yml
cp db/import-tmpl/elemental_db.yml db/import/elemental_db.yml
```

`db/import` is intentionally ignored by rAthena, so existing local files are not
updated by `git pull` or `git rebase`. Stale template copies can trigger startup
warnings such as outdated `ITEM_GROUP_DB`, `ITEM_PACKAGE_DB`, `JOB_STATS`, or
`ELEMENTAL_DB` database versions. If you keep custom rows in these files, upgrade
or merge them instead of overwriting them.

Use the same database names in `conf/inter_athena.conf`:

```conf
map_server_db: rathena
log_db_db: log
```

## Client Data

Point rAthena at the same 2008-era data used by Goro:

```conf
// conf/grf-files.txt
grf: /home/kivutar/Téléchargements/OldRO/data.grf
data_dir: /home/kivutar/Téléchargements/OldRO/
```

The `goro` branch already has this local path. Change it if your OldRO data is
stored elsewhere.

If your map cache does not match the GRF, rebuild it:

```sh
make tools
./mapcache \
  -grf conf/grf-files.txt \
  -list db/map_index.txt \
  -cache db/pre-re/map_cache.dat \
  -rebuild
```

## Legacy Mail Compatibility

Goro implements legacy mail for `PACKETVER=20080910`. Open the mailbox through
an NPC's `openmail` command, or `@mail` if the server permits it.

The local rAthena checkout at `7f30a25e62411b1e1c17c4f21eb151e96eeaf37a`
was checked on 2026-10-22:

- `clif_parse_Mail_setattach` still validates the inventory index before handling
  index zero, which represents Zeny. This rejects legacy Zeny attachments.
- `clif_parse_Mail_winopen` and `clif_parse_Mail_refreshinbox` now pass the wire
  index (internal index + 2) to `mail_removeitem`. Older versions passed the
  internal index and could fail to release an attachment reservation.
- `mail_removeitem`'s delivery branch does not clear its reservation entry.
  Repeated sends and subsequent cleanup still need live validation.

Live acceptance remains pending for sending, reading, replying, taking item and
Zeny attachments, cancelling drafts, returning/deleting mail, full inventories,
invalid recipients, and reconnecting with a request pending. Check both
characters' inventory and Zeny after relogging. These server-side issues must be
resolved without client-side refunds or switching the client to RODEX packets.

## Running

Start the three rAthena servers from separate terminals:

```sh
./login-server
./char-server
./map-server
```

To make a test account a GM:

```sql
UPDATE login SET group_id = 99 WHERE userid = 'Kivutar';
```
