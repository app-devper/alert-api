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
| `PORT`, `MONGO_HOST`, `MONGO_ALERT_DB_NAME`, `REDIS_HOST` | โครงสร้างพื้นฐาน |
| `SECRET_KEY` | ต้องตรงกับ um-api (JWT HS256 + ใช้ hash OTP) |
| `CLIENT_ID`, `SYSTEM` | ตรวจ claim ของ token พนักงาน |
| `CHECKIN_BASE_URL` | URL หน้าเช็กอินที่ฝังใน QR |
| `SMS_API_URL`, `SMS_BALANCE_URL`, `SMS_API_KEY`, `SMS_API_SECRET`, `SMS_SENDER_ID` | Bulk SMS Gateway (Sender ID ต้องจดทะเบียน) |
| `SMS_WEBHOOK_SECRET` | ตรวจ HMAC signature ของ delivery report |
| `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBSCRIBER` | Web Push |
| `LINE_CHANNEL_TOKEN` | LINE Official Notification (PNP) — OA ต้องเปิดใช้ LON กับ LINE ก่อน |
| `LINE_CHANNEL_SECRET` | ตรวจ X-Line-Signature ของ delivery webhook |

Provider ใดไม่ตั้งค่า → โหมด dev จะ log ข้อความแทนการส่งจริง (simulated success)

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
