# goro บน Anbernic RG35XX Plus (StockOS mod) — fbdev build

สายพันธุ์ spike: เรนเดอร์ด้วย **software renderer ลง /dev/fb0 ตรง ๆ** (ไม่ใช้ GPU)
อ่านจอยจาก evdev ตรง ๆ ไม่มี X11/Wayland — ทำงานบน StockOS ของ
cbepx-me/Anbernic-H700-RG-xx-StockOS-Modification แบบที่แอป Z3apps ทำ

## วิธี build

จากเครื่อง dev (Windows หรือ Linux ก็ได้):

```sh
./build-rg35xx.sh
```

ได้แพ็กเกจใน `dist/rg35xx/` ประกอบด้วย:

```
GorORG35/
  goro        (linux/arm64)
  goro.ini
GorORG35.sh   (launcher)
```

## วิธีติดตั้งลงเครื่อง

1. คัดลอกโฟลเดอร์ `GorORG35/` และไฟล์ `GorORG35.sh` ไปไว้ที่ `Roms/APPS/`
   ของ SD card (แบบเดียวกับการติดตั้ง Z3apps ด้วยมือ)
2. ใส่ไฟล์ข้อมูลเกม: คัดลอกโฟลเดอร์ `data/` จาก deployment ของเว็บ
   (ชุดเดียวกับที่ใช้ stream บนเว็บ — GRF ที่แตกแล้ว) ไปไว้
   `Roms/APPS/GorORG35/data/`
3. แก้ที่อยู่เซิร์ฟเวอร์ใน `Roms/APPS/GorORG35/data/clientinfo.xml`:
   `<address>` ให้ชี้มาที่ IP ของเครื่องที่รัน rAthena ในวง LAN
   (ของเว็บเป็น 127.0.0.1 เพราะ webbridge อยู่บนเครื่องเดียวกัน —
   เครื่องเกมต่อ TCP ตรง จึงต้องใช้ IP จริง)
4. รีสตาร์ทเครื่อง (หรือรีสแกน APPS) แล้วเปิดจากเมนู APPS

log การรันอยู่ที่ `Roms/APPS/GorORG35-logfile.txt`

## ปุ่มจอย

| ปุ่ม          | ทำหน้าที่                      |
|---------------|--------------------------------|
| สติ๊กซ้าย/ขวา  | เลื่อนเคอร์เซอร์เมาส์          |
| A             | คลิกซ้าย                       |
| B             | คลิกขวา                        |
| X             | คลิกกลาง                       |
| Y             | Space                          |
| Start         | Enter                          |
| Select        | Esc                            |
| L1/R1/L2/R2   | F1/F2/F3/F4                    |
| L3 (กดสติ๊กซ้าย) | Tab                        |
| R3 (กดสติ๊กขวา)  | Backspace                  |
| D-pad         | ลูกศร                          |

คีย์บอร์ด USB ที่เสียบเพิ่มก็ใช้ได้ (มี keymap ในตัว)

## ข้อจำกัดของ spike นี้

- **ประสิทธิภาพ**: ทั้งหมดเรนเดอร์ด้วย CPU (4×A53) คาดหวัง fps ต่ำ
  หน่วย — เป้าหมายของ spike คือพิสูจน์ว่า pipeline ทำงานได้
  ขั้นถัดไปคือ GBM/KMS + EGL ใช้ GPU Mali-G31 จริง
- **UI เป็นชุด canvas** ของ native build: DOM ทั้งหมดที่ทำไว้สำหรับเว็บ
  (inventory, หน้าเลือก/สร้างตัวละคร, สมุดทะเบียน AEOPD ฯลฯ) ไม่มีในเวอร์ชันนี้
- เสียงปิดไว้ก่อนใน `goro.ini` (`noaudio = true`) — เปิดทีหลังเมื่อ
  ส่วนอื่นเสถียร

## ทดสอบบนเครื่อง dev โดยไม่ต้องมีเครื่องเกม

```sh
# สร้าง fb จำลองเป็นไฟล์เปล่าแล้วรัน
truncate -s $((640*480*4)) fb.bin
GOGPU_PLATFORM=fbdev GOGPU_FB=$PWD/fb.bin GOGPU_FB_WIDTH=640 \
  GOGPU_FB_HEIGHT=480 GOGPU_INPUT_NONE=1 \
  ./goro -data-dir /path/to/data

# ดูว่าเรนเดอร์อะไรออกมา
go build -o fbdump ./cmd/fbdump && ./fbdump -in fb.bin -out frame.png
```

`cmd/fbdump` แปลง raw framebuffer เป็น PNG (32bpp ปกติ / 16bpp RGB565)
ใช้บนเครื่องจริงกับ `/dev/fb0` ได้เหมือนกัน
