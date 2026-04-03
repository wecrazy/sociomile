<!-- markdownlint-disable MD033 MD041 -->
<div align="center">
    <img src="docs/readme-mark.svg" alt="Sociomile" width="84" />
    <h1>Sociomile</h1>
    <p><strong>Proyek fullstack take-home untuk alur dukungan omnichannel multi-tenant.</strong></p>
    <p>
        <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-0f766e.svg" alt="License: Apache 2.0" /></a>
        <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white" alt="Go 1.26+" />
        <img src="https://img.shields.io/badge/Fiber-v3-00B2A9?logo=go&logoColor=white" alt="Go Fiber v3" />
        <img src="https://img.shields.io/badge/React-19-20232A?logo=react&logoColor=61DAFB" alt="React 19" />
        <img src="https://img.shields.io/badge/Vite-6-646CFF?logo=vite&logoColor=white" alt="Vite 6" />
    </p>
    <p>
        <img src="https://img.shields.io/badge/MySQL-8.4-4479A1?logo=mysql&logoColor=white" alt="MySQL 8.4" />
        <img src="https://img.shields.io/badge/Redis-7.4-DC382D?logo=redis&logoColor=white" alt="Redis 7.4" />
        <img src="https://img.shields.io/badge/RabbitMQ-3.13-FF6600?logo=rabbitmq&logoColor=white" alt="RabbitMQ 3.13" />
        <img src="https://img.shields.io/badge/Podman-Compose-892CA0?logo=podman&logoColor=white" alt="Podman Compose" />
    </p>
    <p>Bahasa Indonesia | <a href="README.en.md">English</a></p>
</div>
<!-- markdownlint-enable MD033 MD041 -->

Sociomile adalah implementasi fullstack take-home untuk alur dukungan omnichannel berbasis multi-tenant:

<!-- markdownlint-disable MD033 -->
<div align="center">

**`Channel webhook`** &nbsp;→&nbsp; **`Conversation`** &nbsp;→&nbsp; **`Assignment`** &nbsp;→&nbsp; **`Agent reply`** &nbsp;→&nbsp; **`Ticket escalation`**

</div>
<!-- markdownlint-enable MD033 -->

Repository ini berisi backend Go Fiber v3, worker async terpisah, operator UI berbasis React + Vite, MySQL, Redis, dan RabbitMQ, semuanya dibungkus dalam lingkungan lokal yang kompatibel dengan Podman.

## Stack dan Lisensi

- Lisensi project: Apache License 2.0.
- Stack inti aplikasi: Go 1.26, Go Fiber v3, GORM, React 19, Vite 6, dan TypeScript 5.
- Stack infrastruktur lokal: MySQL 8.4, Redis 7.4, RabbitMQ 3.13, dan Podman Compose.
- Lisensi upstream terpilih: Go Fiber `MIT`, GORM `MIT`, React `MIT`, dan Vite `MIT`. Service infrastruktur container mengikuti lisensi upstream masing-masing.

## Cakupan Implementasi

- Login JWT dengan kontrol akses berbasis role `admin` dan `agent`
- Webhook channel publik yang membuat atau menggunakan kembali percakapan berdasarkan tenant dan channel
- Daftar percakapan dengan pagination offset dan penyaringan di sisi server
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

- Payload webhook sengaja mewajibkan `channel_key` agar simulasi multi-channel menjadi lebih eksplisit. Pendekatan ini sedikit lebih ketat daripada payload minimal di brief, tetapi tetap sejalan dengan alur channel ke percakapan.

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

## Ringkasan Environment

Alur kerja lokal menggunakan dua file env di root:

- `.env` untuk nilai shared dan secret lokal
- `.env.compose` untuk wiring internal container di compose

Ringkasan variabel yang paling penting untuk reviewer:

| Variabel            | Default                                                  | Fungsi                                                                                                                  |
| ------------------- | -------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| `APP_ENV`           | `development`                                            | Menentukan mode runtime backend dan worker                                                                              |
| `BACKEND_PORT`      | `8080`                                                   | Port API pada host                                                                                                      |
| `FRONTEND_PORT`     | `5173`                                                   | Port UI pada host                                                                                                       |
| `MYSQL_DSN`         | `sociomile:sociomile@tcp(localhost:13306)/sociomile?...` | DSN host-side untuk migrate, seed, dan run backend                                                                      |
| `REDIS_ADDR`        | `localhost:16379`                                        | Redis host-side untuk cache dan rate limit                                                                              |
| `RABBITMQ_URL`      | `amqp://guest:guest@localhost:5672/`                     | URL broker untuk backend dan worker                                                                                     |
| `JWT_SECRET`        | `sociomile-local-dev-secret`                             | Secret signing JWT                                                                                                      |
| `ACCESS_TOKEN_TTL`  | `15m`                                                    | TTL access token                                                                                                        |
| `VITE_API_BASE_URL` | `http://localhost:8080/api/v1`                           | Base URL API pada browser                                                                                               |
| `VITE_APP_VERSION`  | auto-generated                                           | Menimpa versi build frontend; jika dikosongkan, Vite menggunakan versi dari `package.json` ditambah timestamp build     |
| `SWAGGER_FILE`      | `./docs/openapi.yaml`                                    | File OpenAPI statis yang diserve backend                                                                                |

Daftar lengkap environment variables ada di [docs/REFERENCE.md](docs/REFERENCE.md).

## Perilaku Update Frontend

- Aset JS dan CSS hasil build sudah menggunakan nama file hashed dari Vite.
- Build frontend juga menghasilkan `version.json` dan metadata build agar UI dapat mendeteksi adanya versi terbaru.
- Saat manifest tersebut menunjukkan versi yang lebih baru, operator akan melihat notifikasi (toast) yang dapat diklik untuk melakukan refresh biasa dan memuat bundle terbaru tanpa perlu hard refresh manual.
- Jika ingin mengunci versi rollout, atur `VITE_APP_VERSION` saat build frontend atau `COMPOSE_VITE_APP_VERSION` pada alur compose.
- Untuk deployment mirip production, cache `/assets/*` boleh bersifat immutable, tetapi `index.html` dan `version.json` sebaiknya menggunakan revalidation atau `no-store`.
- Pada mode development, pemantau pembaruan dinonaktifkan dan Vite HMR tetap menangani perubahan lokal.

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

- Monorepo dipilih agar backend, frontend, worker, dan infrastruktur mudah diulas dalam satu repository.
- Backend menggunakan row-based multi-tenancy pada shared schema, bukan database terpisah per tenant.
- Swagger disajikan dari file OpenAPI statis agar deliverable mudah diperiksa tanpa generator tambahan.
- Frontend sengaja menggunakan state lokal dan utility request yang ringan agar fokus take-home tetap pada alur produk.
- Stack compose tetap menggunakan Vite dev server untuk mempercepat tinjauan lokal, meskipun bukan setup frontend production-grade.

## Dokumentasi Lanjutan

- [Arsitektur dan flow](docs/ARCHITECTURE.md)
- [Referensi operasional](docs/REFERENCE.md)
- [Panduan pengujian](docs/TESTING.md)
- [Spesifikasi OpenAPI](backend/docs/openapi.yaml)

## Keterbatasan Saat Ini

- Target full coverage dari brief belum tercapai; detail snapshot coverage ada di [docs/TESTING.md](docs/TESTING.md)
- Hasil coverage bersih terbaru yang terverifikasi: backend `95.6%` statement coverage dan frontend `97.88%` statement coverage dengan `86.79%` branch coverage serta `85.07%` function coverage; detail lengkap ada di [docs/TESTING.md](docs/TESTING.md)
- Area backend dengan coverage terendah saat ini terutama berada di cabang error saat memuat seed, sebagian helper repository tenant-aware, serta beberapa jalur validasi service seperti kegagalan transaksi webhook dan eskalasi tiket
- Frontend di stack compose masih menggunakan Vite dev server, belum web server statis production-grade
