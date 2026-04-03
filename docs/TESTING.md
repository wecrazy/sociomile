# Panduan Pengujian Sociomile

Bahasa Indonesia | [English](TESTING.en.md) | [README](../README.md)

Dokumen ini merangkum alur pengujian otomatis, laporan coverage, dan validasi manual end-to-end.

## Pengujian Otomatis

Jalankan dari root repository:

```bash
make backend-test
make frontend-test
make lint
make coverage
cd frontend && npm run build
```

Saat dokumen ini ditinjau ulang, perintah berikut berhasil dijalankan:

- `make backend-test`
- `make frontend-test`
- `make lint`
- `make coverage`
- `cd frontend && npm run build`

## Snapshot Coverage

- Backend: `95.6%` statement coverage dari clean run `go test -count=1 -covermode=atomic -coverpkg=./... ./...`
- Frontend: `97.88%` statement coverage, `86.79%` branch coverage, dan `85.07%` function coverage dari `make frontend-coverage`
- Report HTML: `coverage/backend.html` dan `coverage/frontend/index.html`

## Cakupan Test Saat Ini

### Backend

- Login sukses dan gagal
- Lifecycle conversation, lifecycle ticket, dan isolasi tenant di service layer
- Integrasi Fiber router untuk health, login, auth/me, users/agents, webhook intake, assignment, reply, close, escalation, list atau detail ticket, dan update status
- Pemuatan konfigurasi, runner migrasi, cabang CLI `migrate` atau `seed`, titik integrasi startup API atau worker, helper logger dan Redis, helper outbox repository, retry worker untuk cancelation atau eventual success, dan muat data seed
- Helper cache Redis untuk hit atau miss JSON, invalidasi version, rate limit, serta seam setup, `Publish`, atau `Close` publisher dan `OpenDatabase`

### Frontend

- Login sukses dan gagal
- Routing aplikasi dan redirect protected route
- Loading metrik dashboard
- Perilaku app layout untuk pergantian locale dan logout
- Persistensi auth saat login, pembersihan state saat logout, dan fallback untuk state tersimpan yang invalid
- Pergantian locale, fallback bundle English, missing-key behavior, dan persistensi theme
- Hook `useTenantAgents` untuk success, no-token, dan failure path
- Update query filter conversation dan pagination callback atau reset offset
- Update query filter ticket dan pagination callback atau reset offset
- Assignment conversation oleh admin
- Reply conversation oleh agent, eskalasi ticket, dan tampilan linked ticket
- Update status ticket oleh admin dan pembatasan tampilan untuk agent
- Persistensi theme dan locale di settings
- State loading atau empty serta callback pagination di `DataTable`

## Pengujian Manual End-to-End

1. Jalankan stack lokal, migration, dan seed.

```bash
make dev
make migrate
make seed
```

1. Buka service lokal.

- Frontend: `http://localhost:5173`
- Swagger UI: `http://localhost:8080/swagger`

1. Login sebagai admin.

- Email: `alice.admin@acme.local`
- Password: `Password123!`

1. Buat conversation baru lewat webhook simulasi.

```bash
curl -X POST http://localhost:8080/api/v1/channel/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "11111111-1111-1111-1111-111111111111",
    "channel_key": "whatsapp",
    "customer_external_id": "cust-manual-001",
    "customer_name": "Manual QA",
    "message": "Halo, saya butuh bantuan"
  }'
```

1. Verifikasi sebagai admin di UI.

- Conversation baru muncul di daftar conversation
- Admin dapat assign conversation ke `Aaron Agent`

1. Login sebagai agent.

- Email: `aaron.agent@acme.local`
- Password: `Password123!`

1. Verifikasi sebagai agent di UI.

- Agent dapat membuka conversation yang di-assign
- Agent dapat mengirim reply
- Agent dapat melakukan eskalasi menjadi ticket

1. Login kembali sebagai admin dan verifikasi ticket.

- Ticket baru muncul di daftar ticket
- Admin dapat mengubah status ticket menjadi `in_progress`, `resolved`, atau `closed`

## Gap Pengujian yang Masih Ada

- Target full coverage dari brief belum tercapai
- Area backend dengan coverage paling rendah saat ini terutama berada di cabang error saat memuat seed, sebagian helper repository tenant-aware, dan beberapa jalur validasi service seperti kegagalan transaksi webhook serta eskalasi atau pembaruan tiket
- Area frontend dengan coverage paling rendah saat ini terutama berada di cabang edge halaman detail percakapan atau tiket, sebagian callback atau cabang pada halaman list, dan sejumlah kecil cabang UI dashboard atau layout
