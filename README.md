# alert-api

Backend ของระบบแจ้งเตือนเหตุฉุกเฉินภายในสถานที่ (devper-alert) — Go / Gin / MongoDB / Redis
Base path: `/api/alert/v1` (ผ่าน `https://api.devper.app/api/alert/**`)

อ้างอิงข้อกำหนด: [`../alert_requirements.md`](../alert_requirements.md)

## Run

```bash
cp .env.example .env   # เติมค่าให้ครบ
go mod download
go run main.go         # → :8089 (ตาม PORT)
go test ./...
gofmt -w <file>
```

## Environment

| Var | ความหมาย |
|---|---|
| `PORT`, `MONGO_HOST`, `REDIS_HOST` | โครงสร้างพื้นฐาน |
| `MONGO_DB_PREFIX` | prefix ของ DB ต่อ tenant (default `alert`) — clientId `000` ใช้ชื่อ prefix ตรง ๆ, อื่น ๆ เป็น `<prefix>_<clientId>` |
| `SECRET_KEY` | ต้องตรงกับ um-api (JWT HS256 + ใช้ hash OTP) |
| `SYSTEM` | ตรวจ claim ของ token พนักงาน |
| `CLIENT_ID` | (ไม่บังคับ) ถ้าตั้งค่า จะล็อก deployment ให้รับเฉพาะ tenant นั้น; เว้นว่าง = multi-tenant รับทุก clientId จาก JWT |
| `CHECKIN_BASE_URL` | URL หน้าเช็กอินที่ฝังใน QR |
| `SMS_API_URL`, `SMS_BALANCE_URL`, `SMS_API_KEY`, `SMS_API_SECRET`, `SMS_SENDER_ID` | **ค่า default/fallback** ของ Bulk SMS Gateway — ตั้งจริงต่อร้านผ่าน `PUT /admin/messaging-config` |
| `SMS_WEBHOOK_SECRET` | fallback secret ของ delivery report |
| `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBSCRIBER` | Web Push (ระดับแอป ใช้ร่วมทุก tenant) |
| `LINE_CHANNEL_TOKEN`, `LINE_CHANNEL_SECRET` | **fallback** ของ LINE OA (LON) — ตั้งจริงต่อร้านผ่าน messaging config |

**ช่องทางแจ้งเตือน:** Web Push เปิดเสมอ (default) — SMS และ LINE เป็น opt-in ต่อร้านผ่าน
`smsEnabled`/`lineEnabled` (default ปิด เพราะมีค่าใช้จ่ายต่อข้อความ); โหมดทดสอบต้องเปิด SMS ก่อน

**การลงทะเบียนลูกค้า:** ระบุสาขาได้สองทาง — (1) QR token ที่ admin สร้าง (revocable) หรือ
(2) **ลิงก์ตรงด้วย `clientId`+`branchId`** ไม่ต้อง gen token (`GET /public/branch`,
`POST /public/check-ins` รับ clientId/branchId แทน qrToken). **ข้ามขั้นตอน OTP** ได้ต่อสาขา
ผ่าน branch setting `skipOtp` (default ปิด) — เมื่อเปิด ลูกค้ากรอกข้อมูล+ยอมรับ Privacy Notice
แล้ว active ทันทีโดยไม่ต้องยืนยัน OTP (ไม่ส่ง SMS OTP, ไม่มีค่าใช้จ่าย)

**SMS/LINE config ต่อ clientId:** แต่ละร้านตั้ง gateway + Sender ID + LINE OA ของตัวเองใน
collection `messaging_configs` (tenant DB) ผ่าน `GET|PUT /admin/messaging-config`
(GET ตอบแบบ mask, PUT เว้น secret ว่าง = คงค่าเดิม) — ค่าใดไม่ตั้งจะ fallback เป็น env
Webhook ต่อร้านชี้มาที่ `/webhook/sms?clientId=<id>` และ `/webhook/line?clientId=<id>`
เพื่อใช้ secret ของ tenant นั้นตรวจ signature

Provider ใดไม่ตั้งค่า → โหมด dev จะ log ข้อความแทนการส่งจริง (simulated success)

## Multi-tenant (DB ต่อ clientId)

ตาม pattern ของ pharmacy-api: `db.Manager` ถือ `*mongo.Client` เดียว + `sync.Map` cache
ต่อ clientId (`ForClient(clientId)`), สร้าง TTL/unique indexes + seed template เริ่มต้น
ครั้งแรกที่เปิด tenant (`sync.Once`, best-effort) — พนักงานถูก route ด้วย clientId จาก JWT claim;
ฝั่งลูกค้า (ไม่มี JWT) identifiers สาธารณะทุกตัวเป็น tenant ref รูปแบบ `<clientId>.<value>`:
QR token, `checkInId` ที่ตอบจาก API และ `X-Session-Token` — ทำให้ backend เปิด tenant DB
ถูกตัวโดยไม่ต้อง query ข้าม tenant; LINE `X-Line-Delivery-Tag` ก็ฝัง clientId เพื่อให้
delivery webhook จับคู่ log ได้; SMS delivery webhook รองรับ `?clientId=` (ไม่ส่งมาจะไล่ตาม
tenant ที่ active ในแคช)

## Layout

Workspace-standard Go layering: handlers ใน `app/featues/<domain>/`, DTOs ใน
`app/domain/request/`, entities ใน `app/data/entities/`, repositories ใน
`app/data/repositories/`, DI ใน `app/domain/init.go`

- `app/core/alerting` — pure logic: เลือกผู้รับ (invariant 3.7), cooldown, mask เบอร์,
  เลือกภาษา, validate ห้ามลิงก์, นับ SMS segment, OTP/RefCode/EventNo (test coverage ~91%)
- `app/core/messaging` — `MessageProvider` interface + SMS (batch + retry) / Web Push (VAPID,
  จัดการ 410 Gone) / LINE Official Notification (PNP push ตามเบอร์โทรที่ยืนยัน OTP แล้ว —
  ลูกค้าไม่ต้อง add เพื่อน OA, delivery status กลับมาทาง `/webhook/line` ผ่าน
  X-Line-Delivery-Tag) + dispatcher fan-out ขนานทุกช่องทาง
- Middleware chain พนักงาน: `RequireAuthenticated → RequireSession → RequireBranch →
  RequireAuthorization(roles...)`; ลูกค้าใช้ session token (`X-Session-Token`) ที่ออกหลังยืนยัน OTP
- ทุก endpoint ตอบ envelope `{success, data, error, meta}`
- TTL Index: `check_ins.expiresAt`, `otp_requests.expiresAt`, `delivery_logs.expiresAt`
  (PDPA auto-delete ≤ 24 ชม.)

## API surface

ดู [`docs/openapi.yaml`](docs/openapi.yaml) — สรุป:

- `GET /public/qr/:token`, `POST /public/check-ins`, `POST /public/check-ins/:id/verify`,
  `POST /public/check-ins/:id/resend-otp`, `GET /public/vapid`
- `GET|POST /public/me[...]` — สถานะ, เช็กเอาต์, push, ถอนความยินยอม (X-Session-Token)
- `GET /emergency/preview`, `POST /emergency/trigger`, `GET /emergency/active` (พนักงาน)
- `GET /dashboard/summary|check-ins|events[...]`, `POST /dashboard/check-ins/:id/checkout`
- `GET|POST|PUT /admin/templates|settings|permissions|qr[...]`, `GET /admin/sms-credit` (ADMIN)
- `POST /webhook/sms` — delivery report (HMAC signature), `POST /webhook/line` — LON delivery completion (X-Line-Signature)
- `GET /health`
