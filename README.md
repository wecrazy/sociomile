# Sociomile

Bahasa Indonesia | [English](README.en.md)

Sociomile adalah implementasi take-home fullstack untuk alur omnichannel support berbasis multi-tenant:

`Channel webhook -> Conversation -> Assignment -> Agent reply -> Ticket escalation`

Repository ini berisi backend Go Fiber v3, worker async terpisah, operator UI berbasis React + Vite, MySQL, Redis, dan RabbitMQ, semuanya dibungkus dalam workflow lokal yang kompatibel dengan Podman.

## Cakupan Implementasi

- Login JWT dengan kontrol akses berbasis role `admin` dan `agent`
- Webhook channel publik yang membuat atau memakai ulang conversation berdasarkan tenant dan channel
- Antrean conversation dengan pagination offset dan filtering di sisi server
- Alur assignment oleh admin dan reply oleh agent
- Eskalasi ticket dengan aturan satu conversation maksimal satu ticket
- Update status ticket khusus admin
- Locale berbasis YAML dengan default English dan opsi Indonesian
- Persistensi preferensi dark mode dan light mode
- Activity log plus outbox event yang dipublish worker melalui RabbitMQ
- Swagger UI dari `backend/docs/openapi.yaml`

## Diagram Workflow

```mermaid
flowchart LR
    A[Channel webhook] --> B{Conversation aktif ada?}
    B -- Tidak --> C[Buat conversation baru]
    B -- Ya --> D[Pakai conversation existing]
    C --> E[Simpan message customer]
    D --> E
    E --> F[Admin assign agent]
    F --> G[Agent reply]
    G --> H[Agent escalate ke ticket]
    H --> I[Buat ticket]
    I --> J[Admin update status ticket]
```

## Diagram Arsitektur

```mermaid
flowchart TB
    EXT[Simulated channel webhook] --> API[Go Fiber API]
    FE[React + Vite Frontend] --> API
    API --> MYSQL[(MySQL)]
    API --> REDIS[(Redis)]
    API --> OUTBOX[Activity log + outbox]
    OUTBOX --> MYSQL
    WORKER[Go Worker] --> MYSQL
    WORKER --> RMQ[(RabbitMQ)]
```

## Audit Kesesuaian Assignment

| Area                           | Status    | Catatan                                                                                                           |
| ------------------------------ | --------- | ----------------------------------------------------------------------------------------------------------------- |
| Authentication & authorization | `Sesuai`  | Login JWT, role `admin` dan `agent`, proteksi endpoint, serta validasi otorisasi di service layer                 |
| Channel -> conversation flow   | `Sesuai`  | Webhook membuat atau memakai conversation aktif, lalu menyimpan message customer                                  |
| Conversation management        | `Sesuai`  | List, detail, reply, assign, close, filter, dan pagination tersedia                                               |
| Escalation to ticket           | `Sesuai`  | Hanya agent yang bisa escalate, satu conversation maksimal satu ticket, dan status ticket hanya bisa diubah admin |
| Multi-tenancy                  | `Sesuai`  | Isolasi tenant diterapkan di JWT claim, service, repository, dan seed data                                        |
| Database & migration           | `Sesuai`  | SQL migration memakai foreign key dan index yang relevan                                                          |
| Async events & worker          | `Sesuai`  | `activity_logs` dan `outbox_events` diproses worker RabbitMQ                                                      |
| Redis usage                    | `Sesuai`  | Redis dipakai untuk rate limit webhook dan cache list conversation atau ticket                                    |
| Docker compose stack           | `Sesuai`  | Stack mencakup backend, worker, MySQL, Redis, RabbitMQ, dan frontend                                              |
| Testing                        | `Parsial` | Test inti sudah ada, tetapi target brief untuk full coverage belum tercapai                                       |
| OpenAPI                        | `Sesuai`  | Swagger sekarang mendeskripsikan seluruh route aplikasi yang diimplementasikan                                    |

Catatan tambahan:

- Payload webhook sengaja mewajibkan `channel_key` agar simulasi multi-channel eksplisit. Ini sedikit lebih ketat daripada payload minimal di brief, tetapi tetap sejalan dengan flow channel ke conversation.

## Prasyarat

- Go 1.26+
- Node.js 22+
- Podman 4.9+ dengan `podman compose` atau `podman-compose`

## Mulai Cepat

```bash
git clone https://github.com/wecrazy/sociomile.git
cd sociomile
make env
make setup
make dev
```

Perintah opsional setelah stack hidup:

- `make dev-logs` untuk mengikuti log
- `make dev-down` untuk menghentikan stack
- `make migrate` dan `make seed` jika ingin menjalankan ulang workflow database dari host secara manual

URL lokal utama:

- Frontend: `http://localhost:5173`
- Backend health: `http://localhost:8080/health`
- Swagger UI: `http://localhost:8080/swagger`
- RabbitMQ management: `http://localhost:15672`

> Catatan: file nyata `.env`, `.env.compose`, `backend/.env`, dan `frontend/.env` dipakai untuk pengujian lokal dan sudah diabaikan oleh git.
>
> `make dev` menyalakan stack penuh di background. Backend juga otomatis menerapkan migration saat startup, dan pada `APP_ENV=development` demo tenant, user, channel, serta sample data akan di-ensure tanpa workflow seed terpisah. Setelah reset volume MySQL lokal, akun demo akan muncul lagi tanpa perlu `make seed` rutin.

## Ringkasan Environment

Workflow lokal memakai dua file env di root:

- `.env` untuk nilai shared dan secret lokal
- `.env.compose` untuk wiring internal container di compose

Ringkasan variabel yang paling penting untuk reviewer:

| Variabel            | Default                                                  | Fungsi                                                                                            |
| ------------------- | -------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `APP_ENV`           | `development`                                            | Menentukan mode runtime backend dan worker                                                        |
| `BACKEND_PORT`      | `8080`                                                   | Port API pada host                                                                                |
| `FRONTEND_PORT`     | `5173`                                                   | Port UI pada host                                                                                 |
| `MYSQL_DSN`         | `sociomile:sociomile@tcp(localhost:13306)/sociomile?...` | DSN host-side untuk migrate, seed, dan run backend                                                |
| `REDIS_ADDR`        | `localhost:16379`                                        | Redis host-side untuk cache dan rate limit                                                        |
| `RABBITMQ_URL`      | `amqp://guest:guest@localhost:5672/`                     | URL broker untuk backend dan worker                                                               |
| `JWT_SECRET`        | `sociomile-local-dev-secret`                             | Secret signing JWT                                                                                |
| `ACCESS_TOKEN_TTL`  | `15m`                                                    | TTL access token                                                                                  |
| `VITE_API_BASE_URL` | `http://localhost:8080/api/v1`                           | Base URL API pada browser                                                                         |
| `VITE_APP_VERSION`  | auto-generated                                           | Override versi build frontend; jika kosong Vite memakai versi `package.json` plus timestamp build |
| `SWAGGER_FILE`      | `./docs/openapi.yaml`                                    | File OpenAPI statis yang diserve backend                                                          |

Daftar lengkap environment variables ada di [docs/REFERENCE.md](docs/REFERENCE.md).

## Perilaku Update Frontend

- Asset JS dan CSS hasil build sudah memakai nama file hashed dari Vite.
- Build frontend juga menerbitkan `version.json` dan metadata build agar UI dapat mendeteksi adanya build baru.
- Saat manifest tersebut menunjukkan versi yang lebih baru, operator akan melihat toast untuk refresh normal dan mengambil bundle terbaru tanpa perlu hard refresh manual.
- Jika ingin mengunci versi rollout, set `VITE_APP_VERSION` saat build frontend atau `COMPOSE_VITE_APP_VERSION` pada workflow compose.
- Untuk deploy production-like, cache `/assets/*` boleh immutable, tetapi `index.html` dan `version.json` sebaiknya memakai revalidation atau `no-store`.
- Pada mode development, monitor update dimatikan dan Vite HMR tetap menangani perubahan lokal.

## Akun Demo

Halaman login sekarang menampilkan shortcut cepat untuk role `admin` dan `agent` tenant Acme. Semua user demo memakai password `Password123!`.

| Role    | Tenant         | Nama          | Email                      |
| ------- | -------------- | ------------- | -------------------------- |
| `admin` | `Acme Support` | `Alice Admin` | `alice.admin@acme.local`   |
| `agent` | `Acme Support` | `Aaron Agent` | `aaron.agent@acme.local`   |
| `admin` | `Globex Care`  | `Grace Admin` | `grace.admin@globex.local` |
| `agent` | `Globex Care`  | `Gina Agent`  | `gina.agent@globex.local`  |

## Ringkasan API

### Endpoint Publik

| Method | Path                      | Fungsi                            |
| ------ | ------------------------- | --------------------------------- |
| `GET`  | `/health`                 | Health probe backend              |
| `GET`  | `/swagger`                | Swagger UI                        |
| `POST` | `/api/v1/auth/login`      | Login email dan password          |
| `POST` | `/api/v1/channel/webhook` | Simulasi pesan masuk dari channel |

### Endpoint Terproteksi

| Method  | Path                                 | Role             | Fungsi                                         |
| ------- | ------------------------------------ | ---------------- | ---------------------------------------------- |
| `GET`   | `/api/v1/auth/me`                    | `admin`, `agent` | Payload user saat ini                          |
| `GET`   | `/api/v1/users/agents`               | `admin`, `agent` | Daftar agent aktif per tenant                  |
| `GET`   | `/api/v1/conversations`              | `admin`, `agent` | List conversation dengan filter dan pagination |
| `GET`   | `/api/v1/conversations/:id`          | `admin`, `agent` | Detail conversation dan thread message         |
| `POST`  | `/api/v1/conversations/:id/messages` | `agent`          | Reply dari agent                               |
| `PATCH` | `/api/v1/conversations/:id/assign`   | `admin`          | Assign conversation ke agent                   |
| `PATCH` | `/api/v1/conversations/:id/close`    | `admin`, `agent` | Tutup conversation                             |
| `POST`  | `/api/v1/conversations/:id/escalate` | `agent`          | Eskalasi conversation menjadi ticket           |
| `GET`   | `/api/v1/tickets`                    | `admin`, `agent` | List ticket dengan filter dan pagination       |
| `GET`   | `/api/v1/tickets/:id`                | `admin`, `agent` | Detail ticket                                  |
| `PATCH` | `/api/v1/tickets/:id/status`         | `admin`          | Update status ticket                           |

## Penjelasan Multi-Tenancy

- Semua entity yang dimiliki tenant membawa `tenant_id` pada level database.
- Endpoint terproteksi mengambil konteks tenant dari claim JWT, bukan dari payload client.
- Repository layer selalu memfilter query berdasarkan `tenant_id` untuk mencegah cross-tenant read atau write.
- Endpoint webhook adalah pengecualian yang memang menerima `tenant_id` karena mensimulasikan callback dari channel eksternal.
- Seed data berisi dua tenant agar isolasi tenant bisa diuji cepat dari UI, API, dan test otomatis.

## Asumsi dan Trade-Off

- Monorepo dipilih agar backend, frontend, worker, dan infrastruktur mudah direview dalam satu repository.
- Backend memakai row-based multi-tenancy pada shared schema, bukan database terpisah per tenant.
- Swagger disajikan dari file OpenAPI statis agar deliverable mudah diinspeksi tanpa generator tambahan.
- Frontend sengaja memakai local state dan utility request yang ringan supaya fokus take-home tetap pada flow produk.
- Stack compose tetap memakai Vite dev server untuk mempercepat review lokal, walaupun itu bukan setup frontend production-grade.

## Dokumentasi Lanjutan

- [Arsitektur dan flow](docs/ARCHITECTURE.md)
- [Referensi operasional](docs/REFERENCE.md)
- [Panduan pengujian](docs/TESTING.md)
- [Spesifikasi OpenAPI](backend/docs/openapi.yaml)

## Keterbatasan Saat Ini

- Target full coverage dari brief belum tercapai; detail snapshot coverage ada di [docs/TESTING.md](docs/TESTING.md)
- Snapshot clean terbaru yang tervalidasi sekarang adalah backend `95.6%` statement coverage dan frontend `97.88%` statement coverage dengan `86.79%` branch coverage serta `85.07%` function coverage; detail lengkap ada di [docs/TESTING.md](docs/TESTING.md)
- Area coverage backend yang masih tersisa sekarang terutama ada di cabang error seed loading, sebagian helper repository tenant-aware, serta beberapa path validasi service seperti transaksi webhook dan eskalasi ticket
- Frontend di stack compose masih memakai Vite dev server, belum web server statis production-grade
