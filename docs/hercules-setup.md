# Hercules Setup

Goro targets the classic pre-renewal `2008-09-10aSakexe` flow by default.
Use the Goro-compatible Hercules fork for this profile: upstream still needs
packet fixes for equipment, NPC dialogs, skills, and other 2008 client features.
This guide describes a local Linux development server.

## Recommended Source

Use the `goro` branch:

```sh
git clone --branch goro https://github.com/kivutar/hercules.git
cd hercules
git pull --ff-only
```

The branch is based on upstream `39f61e195` (v2026.09 plus subsequent fixes)
and contains:

- corrected packet lengths for the `20080910` Sakray profile
- packet shuffle guards that keep early Renewal layouts out of Sakray
- old-client skill updates, homunculus property packets, and packet safety fixes
- an Izlude collision cache matching the 268×268 map in OldRO

Compatibility testing is ongoing. The fork contains source and map-cache
changes; configure your database credentials and local paths separately.

## Build

Install Git, a C++20 compiler, CMake 3.25 or newer, Ninja, Python with venv/pip,
and MariaDB. This Hercules version uses CMake and Conan instead of the old
`./configure` / `make server` workflow.

From the Hercules checkout:

```sh
python3 -m venv .venv
.venv/bin/python -m pip install conan==2.33.0

cmake -S . -B build -G Ninja \
  -DCMAKE_BUILD_TYPE=Debug \
  -DCMAKE_PROJECT_TOP_LEVEL_INCLUDES=cmake_modules/conan_provider.cmake \
  -DCONAN_COMMAND="$PWD/.venv/bin/conan" \
  '-DCONAN_INSTALL_ARGS=--build=missing;-c;tools.build:jobs=4' \
  -DPACKETVER=20080910 \
  -DENABLE_PACKETVER_SAK=ON \
  -DENABLE_RENEWAL=OFF

cmake --build build --parallel 4
cmake --install build
```

Conan downloads and builds the dependencies during configuration; the first
build can take a while. `Debug` keeps packet validation enabled for testing.
Use a fresh build directory if switching from another packet profile.

The default install location is `bin/` inside the checkout. Keep its `lib/`
and `herc-plugins/` directories alongside the executables. Run the servers
from the checkout root so they can find `conf/`, `db/`, `maps/`, and `npc/`.

Check the compiled profile:

```sh
./bin/map-server --version
```

It should report `20080910 sak`. Both the date and `ENABLE_PACKETVER_SAK`
matter; a Renewal build with the same date uses different packets.

## Required Config

Use the local override files under `conf/import/`. On a fresh checkout, copy
the templates if that directory has not been created yet:

```sh
test -d conf/import || cp -r conf/import-tmpl conf/import
```

Merge the following settings into their corresponding files. Replace
`CHANGE_ME_INTER_PASSWORD` with the same inter-server password in both files
and in the SQL account configured below.

```conf
// conf/import/login-server.conf
login_configuration: {
    inter: {
        bind_ip: "127.0.0.1"
        login_port: 6900
    }
    account: {
        new_account: true
    }
}
```

```conf
// conf/import/char-server.conf
char_configuration: {
    server_name: "Hercules"
    inter: {
        userid: "goro_inter"
        passwd: "CHANGE_ME_INTER_PASSWORD"
        login_ip: "127.0.0.1"
        bind_ip: "127.0.0.1"
        char_ip: "127.0.0.1"
        login_port: 6900
        char_port: 6121
    }
    player: {
        new: {
            start_point_pre: {
                map: "prontera"
                x: 156
                y: 191
            }
        }
        deletion: {
            delay: 0
        }
    }
    pincode: {
        enabled: false
    }
}
```

```conf
// conf/import/map-server.conf
map_configuration: {
    inter: {
        userid: "goro_inter"
        passwd: "CHANGE_ME_INTER_PASSWORD"
        char_ip: "127.0.0.1"
        bind_ip: "127.0.0.1"
        map_ip: "127.0.0.1"
        char_port: 6121
        map_port: 5121
    }
}
```

Starting new characters in Prontera avoids requiring the newer training-ground
map selected by upstream. Starter zeny and test NPCs are optional; the 2008
client uses immediate character deletion with an email check, without a PIN.

These examples bind all game servers to localhost. For LAN testing, bind them
to a reachable interface, set the advertised `char_ip` and `map_ip` to the
host's LAN address, and add that subnet to `conf/network.conf`. The login
server address in the client's `clientinfo.xml` must match. If rAthena is
already using ports 6900, 6121, and 5121, stop it or choose different ports
consistently in both server and client settings.

## MariaDB Import

For a fresh local install, start MariaDB and run the following SQL as a database
administrator. Replace `CHANGE_ME_DB_PASSWORD` with your own database password:

```sql
CREATE DATABASE hercules;
CREATE USER 'hercules'@'127.0.0.1' IDENTIFIED BY 'CHANGE_ME_DB_PASSWORD';
GRANT ALL PRIVILEGES ON hercules.* TO 'hercules'@'127.0.0.1';
```

Import both schemas into that database:

```sh
mariadb -h 127.0.0.1 -P 3306 -u hercules -p hercules < sql-files/main.sql
mariadb -h 127.0.0.1 -P 3306 -u hercules -p hercules < sql-files/logs.sql
```

The default configuration reads item and monster definitions from `db/pre-re/`;
the optional SQL item/mob tables are not needed for this setup. The main and
log tables share one database here, matching Hercules's global SQL defaults.

Edit the connection settings in `conf/global/sql_connection.conf`:

```conf
sql_connection: {
    db_hostname: "127.0.0.1"
    db_port: 3306
    db_username: "hercules"
    db_password: "CHANGE_ME_DB_PASSWORD"
    db_database: "hercules"
}
```

Use your actual MariaDB port throughout; a separate development instance can
use 3307. Keep the real password out of commits: this global config file is
tracked, unlike `conf/import/`.

In the `hercules` database, replace the schema's default inter-server account
credentials with the values used in the char/map configuration:

```sql
UPDATE login
SET userid = 'goro_inter', user_pass = 'CHANGE_ME_INTER_PASSWORD'
WHERE account_id = 1 AND sex = 'S';
```

For an existing database, back it up and apply missing files from
`sql-files/upgrades/` in order instead of importing the fresh schemas. In
particular, `2024-11-03--07-00.sql` adds `storage.storage_id`. Check whether its
timestamp `1730631632` is already present in `sql_updates` before applying it.
A fresh `main.sql` import already includes that migration.

## Client Data

Use the same 2008-era data as Goro; see [Client setup](client-setup.md).
Point Hercules at your OldRO directory:

```conf
// conf/grf-files.txt
grf: /path/to/OldRO/data.grf
data_dir: /path/to/OldRO/
```

`data_dir` is the root directory, not its `data/` subdirectory. These paths are
local configuration. Hercules does not read Goro's `data.ini`; if you override
maps in other GRFs, list those archives in `conf/grf-files.txt` as well, with
higher-priority archives last.

Hercules normally uses cached collision maps in `maps/pre-re/`. Merely changing
the GRF path does not update those caches. With the map server stopped, rebuild
an affected map using the installed mapcache plugin:

```sh
./bin/map-server --load-plugin mapcache --map izlude
```

The fork already includes the OldRO Izlude cache. Upstream's cache is 268×300,
while OldRO's map is 268×268; using the wrong one leaves the character unable
to move at the entrance from `prt_fild08`. Repeat the command for any other map
whose collision data differs from your client, then restart the map server.

## Running

Start the three servers from separate terminals, each at the checkout root:

```sh
./bin/login-server
./bin/char-server
./bin/map-server
```

Wait for the map server to report `Map Server is now online.` The API server
is not needed for this 2008 client. Point Goro's `data/clientinfo.xml` at
`127.0.0.1:6900`, following [Client setup](client-setup.md), and launch Goro
with your OldRO data directory.

With `new_account: true`, log in once with a username ending in `_M` or `_F`
to create an account, then use the username without the suffix. The default
email for these accounts is `a@a.com`; that is the character-deletion email
unless you change it.

To make a test account a GM, run this in the `hercules` database and reconnect:

```sql
UPDATE login SET group_id = 99 WHERE userid = 'YourTestAccount';
```
