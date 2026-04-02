# Referensi Operasional Sociomile

Bahasa Indonesia | [English](REFERENCE.en.md) | [README](../README.md)

Dokumen ini merangkum layout repository, command utama, environment variables, dan ringkasan API.

## Struktur Repository

- `backend/` service backend, worker, migration, seed, dan file OpenAPI
- `frontend/` operator UI React
- `docker-compose.yml` definisi stack lokal yang kompatibel dengan Podman compose
- `Makefile` entry point tunggal untuk setup, run, build, test, migrate, seed, dan coverage
- `.env.example` template env shared untuk workflow lokal
- `.env.compose.example` template env khusus compose

## Ringkasan Cepat untuk Reviewer

### Alur Review Cepat

1. `git clone https://github.com/wecrazy/sociomile.git`
1. `cd sociomile`
1. `make env`
1. `make setup`
1. `make dev`
1. `make migrate && make seed`
1. Buka `http://localhost:5173`
1. Login dengan `alice.admin@acme.local` / `Password123!`

### URL Penting

| Layanan | URL |
| --- | --- |
| Frontend | `http://localhost:5173` |
| Backend health | `http://localhost:8080/health` |
| Swagger UI | `http://localhost:8080/swagger` |
| RabbitMQ management | `http://localhost:15672` |

## Default Lokal

- Frontend: `5173`
- Backend: `8080`
- MySQL: `13306`
- Redis: `16379`
- RabbitMQ AMQP: `5672`
- RabbitMQ management: `15672`

## Akun Demo

Perintah seed akan membuat user berikut dengan password `Password123!`:

- `alice.admin@acme.local`
- `aaron.agent@acme.local`
- `grace.admin@globex.local`
- `gina.agent@globex.local`

## Perintah Makefile

| Perintah | Fungsi |
| --- | --- |
| `make help` | Tampilkan daftar target yang tersedia |
| `make env` | Buat file env lokal tanpa menimpa yang sudah ada |
| `make setup` | Instal dependency backend dan frontend |
| `make dev` | Build lalu jalankan seluruh stack lokal |
| `make dev-down` | Hentikan dan hapus stack lokal |
| `make dev-logs` | Ikuti log compose secara realtime |
| `make config` | Cetak konfigurasi compose hasil resolusi dari `.env` dan `.env.compose` |
| `make migrate` | Terapkan migration SQL backend |
| `make seed` | Muat demo tenant, user, dan channel |
| `make fmt` | Format source Go backend dan file source atau config frontend |
| `make backend-fmt` | Format file Go backend dengan `gofmt` |
| `make frontend-fmt` | Format file source dan config frontend dengan `Prettier` |
| `make backend-lint` | Jalankan cek `gofmt`, `go vet`, dan `revive` untuk backend |
| `make backend-test` | Jalankan test backend |
| `make frontend-test` | Jalankan test frontend |
| `make backend-coverage` | Jalankan coverage backend dan hasilkan report `coverage/backend.html` |
| `make frontend-coverage` | Jalankan coverage frontend dan hasilkan report di `coverage/frontend/` |
| `make coverage` | Jalankan seluruh workflow coverage backend dan frontend |
| `make lint` | Jalankan lint Go backend, cek format frontend, dan lint TypeScript frontend |
| `make build` | Build image container |
| `make swagger` | Tampilkan petunjuk Swagger UI |

## Variabel Environment

Workflow lokal memakai dua file env di root:

- `.env` untuk nilai shared dan secret lokal
- `.env.compose` untuk wiring internal container compose

File `backend/.env` dan `frontend/.env` tetap dipakai untuk menjalankan service langsung dari host.

### Variabel Root Shared

| Variabel | Default | Fungsi |
| --- | --- | --- |
| `COMPOSE_PROJECT_NAME` | `sociomile` | Namespace project compose |
| `MYSQL_DATABASE` | `sociomile` | Nama database aplikasi |
| `MYSQL_USER` | `sociomile` | User MySQL aplikasi |
| `MYSQL_PASSWORD` | `sociomile` | Password MySQL aplikasi |
| `MYSQL_ROOT_PASSWORD` | `root` | Password root MySQL |
| `MYSQL_PORT` | `13306` | Host port untuk MySQL |
| `REDIS_PORT` | `16379` | Host port untuk Redis |
| `RABBITMQ_PORT` | `5672` | Host port untuk AMQP RabbitMQ |
| `RABBITMQ_MANAGEMENT_PORT` | `15672` | Host port untuk UI RabbitMQ |
| `RABBITMQ_DEFAULT_USER` | `guest` | User login RabbitMQ |
| `RABBITMQ_DEFAULT_PASS` | `guest` | Password RabbitMQ |
| `BACKEND_PORT` | `8080` | Host port untuk backend |
| `FRONTEND_PORT` | `5173` | Host port untuk frontend |
| `JWT_SECRET` | `sociomile-local-dev-secret` | Secret JWT untuk runtime compose |
| `ACCESS_TOKEN_TTL` | `15m` | Durasi kedaluwarsa JWT |
| `APP_ENV` | `development` | Penanda environment runtime |
| `LOG_LEVEL` | `debug` | Level log backend dan worker |

### Variabel Root Compose

| Variabel | Default | Fungsi |
| --- | --- | --- |
| `COMPOSE_MYSQL_DSN` | `sociomile:sociomile@tcp(mysql:3306)/sociomile?...` | DSN internal yang dipakai backend dan worker di network compose |
| `COMPOSE_REDIS_ADDR` | `redis:6379` | Alamat Redis internal di network compose |
| `COMPOSE_RABBITMQ_URL` | `amqp://guest:guest@rabbitmq:5672/` | URL RabbitMQ internal untuk backend dan worker |
| `COMPOSE_VITE_API_BASE_URL` | `http://localhost:8080/api/v1` | Base URL API yang bisa diakses browser saat frontend dijalankan lewat stack compose |
| `COMPOSE_VITE_APP_NAME` | `Sociomile` | Nama aplikasi frontend di compose |
| `COMPOSE_SWAGGER_FILE` | `./docs/openapi.yaml` | Lokasi file OpenAPI yang dipakai container backend |

Jika salah satu variabel yang dipakai compose hilang, `make dev`, `make build`, dan `podman-compose config` akan gagal lebih awal dengan pesan error yang menyebut nama variabel yang belum diisi.

Catatan untuk MySQL lokal: container MySQL hanya membuat database dan user dari env saat volume data masih kosong. Jika `MYSQL_DATABASE`, `MYSQL_USER`, atau password diubah setelah volume `sociomile_mysql_data` sudah terbuat, hapus volume tersebut lalu start ulang stack agar provisioning dijalankan ulang.

### Menjalankan Compose Secara Langsung

Jika tidak ingin lewat Makefile, source dulu `.env` lalu berikan `.env.compose` ke compose command:

```bash
set -a
. ./.env
set +a
podman compose --env-file ./.env.compose config
podman compose --env-file ./.env.compose up --build
```

Jika environment Anda memakai `podman-compose` secara langsung, flag `--env-file` yang sama tetap berlaku:

```bash
set -a
. ./.env
set +a
podman-compose --env-file ./.env.compose config
```

### Variabel Runtime Backend

| Variabel | Default | Fungsi |
| --- | --- | --- |
| `APP_ENV` | `development` | Environment runtime |
| `BACKEND_PORT` | `8080` | Port listen API |
| `MYSQL_DSN` | `sociomile:sociomile@tcp(localhost:13306)/sociomile?...` | DSN host-side untuk migrate, seed, dan run lokal |
| `REDIS_ADDR` | `localhost:16379` | Alamat Redis host-side |
| `REDIS_PASSWORD` | kosong | Password Redis |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | URL koneksi RabbitMQ host-side |
| `JWT_SECRET` | `sociomile-local-dev-secret` | Secret JWT |
| `ACCESS_TOKEN_TTL` | `15m` | Durasi JWT |
| `LOG_LEVEL` | `debug` | Tingkat verbosity log |
| `SWAGGER_FILE` | `./docs/openapi.yaml` | Sumber OpenAPI statis untuk Swagger UI |

### Variabel Runtime Frontend

| Variabel | Default | Fungsi |
| --- | --- | --- |
| `VITE_API_BASE_URL` | `http://localhost:8080/api/v1` | Base URL API untuk browser |
| `VITE_APP_NAME` | `Sociomile` | Nama aplikasi UI |

## Ringkasan API

### Endpoint Publik

| Method | Path | Fungsi |
| --- | --- | --- |
| `GET` | `/health` | Health probe |
| `GET` | `/swagger` | Swagger UI |
| `POST` | `/api/v1/auth/login` | Login email dan password |
| `POST` | `/api/v1/channel/webhook` | Simulasi pesan masuk dari channel |

### Endpoint Terproteksi

| Method | Path | Role | Fungsi |
| --- | --- | --- | --- |
| `GET` | `/api/v1/auth/me` | admin, agent | Payload user saat ini |
| `GET` | `/api/v1/users/agents` | admin, agent | Daftar agent aktif per tenant |
| `GET` | `/api/v1/conversations` | admin, agent | List conversation dengan filter di sisi server |
| `GET` | `/api/v1/conversations/:id` | admin, agent | Detail conversation dan thread message |
| `POST` | `/api/v1/conversations/:id/messages` | agent | Reply dari agent |
| `PATCH` | `/api/v1/conversations/:id/assign` | admin | Assign conversation ke agent |
| `PATCH` | `/api/v1/conversations/:id/close` | admin, agent | Tutup conversation |
| `POST` | `/api/v1/conversations/:id/escalate` | agent | Eskalasi conversation menjadi ticket |
| `GET` | `/api/v1/tickets` | admin, agent | List ticket dengan filter di sisi server |
| `GET` | `/api/v1/tickets/:id` | admin, agent | Detail ticket |
| `PATCH` | `/api/v1/tickets/:id/status` | admin | Update status ticket |
