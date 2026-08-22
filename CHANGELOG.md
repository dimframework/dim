# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

---

## [v0.11.0] - 2026-08-22

### Added
- **`GetAllMigrations()`**: Mengembalikan gabungan migrasi framework dan migrasi aplikasi yang terdaftar via `Register()`, terurut berdasarkan `Version`. Dimaksudkan sebagai sumber tunggal urutan migrasi, sehingga `migrate`, `migrate:list`, dan `migrate:rollback` melihat urutan yang sama.
- **Flag `-allow-missing` pada `migrate:rollback`**: Meneruskan rollback dengan melewati migrasi yang tercatat di database tetapi kodenya tidak ada di registry. Tanpa flag ini perintahnya menolak berjalan — flag-nya menjadi jalan keluar bagi yang memang punya migrasi yatim dan tanpa itu akan terkunci dari rollback sama sekali.

### Fixed
- **`GetRegisteredMigrations()` tidak mengurutkan berdasarkan `Version`**: Doc comment-nya menjanjikan pengurutan, tetapi fungsinya hanya menyalin slice registry. Urutan jalan migrasi menjadi urutan `dim.Register()` dipanggil — yakni urutan `init()` package — bukan urutan `Version`. Closes [#20](https://github.com/dimframework/dim/issues/20).
  - Aplikasi yang menaruh seluruh migrasinya di **satu** package selamat tanpa menyadarinya: Go menjalankan `init()` sebuah package menurut urutan nama berkas, dan konvensi penamaan `20260802100100_create_partners_table.go` membuat urutan nama = urutan versi. Benar karena kebetulan, bukan karena dijamin.
  - Menggigit begitu migrasi tersebar ke lebih dari satu package — aplikasi modular yang tiap modulnya membawa migrasinya sendiri lalu di-blank-import dari `cmd/app`. Migrasi bernomor lebih besar dapat berjalan lebih dulu, dan `CREATE TABLE` ber-`REFERENCES` ke tabel yang belum lahir gagal.
  - Sulit ketahuan karena `RunMigrations` melewati versi yang sudah tercatat: **database yang sudah terisi tidak pernah menunjukkannya.** Yang gagal hanya database segar — deploy pertama di lingkungan baru, dan test yang membangun skema dari nol.
  - Pengurutan bersifat stabil, sehingga migrasi dengan `Version` kembar tetap mengikuti urutan pendaftarannya alih-alih berubah-ubah.
  - Ditambahkan `GetAllMigrations()` yang menggabungkan migrasi framework dan migrasi aplikasi lalu mengurutkan **slice gabungannya**. `migrate`, `migrate:list`, dan `migrate:rollback` kini memakainya sebagai sumber tunggal urutan. Sebelumnya ketiganya menyambung dengan `append(GetFrameworkMigrations(), GetRegisteredMigrations()...)` yang urut hanya karena versi framework kebetulan selalu lebih kecil.
  - `migrate:rollback` tidak pernah terdampak dan tetap berjalan menurun: urutannya berasal dari `ORDER BY version DESC` pada query, bukan dari registry, yang hanya dipakai untuk mencari fungsi `Down` berdasarkan versi.
- **`migrate:rollback` membongkar migrasi yang lebih tua saat ada migrasi yang kodenya hilang**: Baris yang tercatat di tabel `migrations` tetapi tidak ada di registry hanya diberi peringatan lalu dilewati, sementara rollback lanjut ke migrasi di bawahnya. Skema milik migrasi yang dilewati tetap hidup sementara dependensinya ikut dibongkar — kebalikan dari masalah urutan di atas.
  - Perintah kini **menolak berjalan dan tidak menyentuh apa pun** bila ada migrasi target yang fungsi `Down`-nya tidak tersedia, dengan menyebut versi mana yang bermasalah.
  - Flag baru **`-allow-missing`** menyediakan jalan keluar bagi yang memang punya migrasi yatim dan ingin melewatinya; tanpa itu, mereka akan terkunci dari rollback sama sekali.
  - Pesan "Rolling back N migration(s)" kini memakai jumlah sebenarnya, bukan nilai `-step` mentah — sebelumnya `-step 3` yang hanya mengerjakan 2 tetap melaporkan 3.
- **`migrate:list` salah menghitung `Pending`**: Rumusnya `len(migrations) - len(appliedMap)`, sehingga versi yang tercatat di database tetapi tidak ada di registry ikut mengurangi hitungan — angkanya bisa keliru, bahkan negatif. Dengan registry `{100, 200, 400}` dan database `{100, 200, 300}`, keluarannya berbunyi `Applied: 3 | Pending: 0` padahal migrasi `400` belum pernah dijalankan.
  - Hitungan kini diambil dengan menelusuri registry, bukan dari `len()`.
  - Versi yang hanya ada di database ditampilkan sebagai baris berstatus **`Orphan`** beserta ringkasannya sendiri. Sengaja dimunculkan, bukan disembunyikan: inilah yang menjelaskan mengapa `migrate:rollback` menolak berjalan.

---

## [v0.10.0] - 2026-08-08

### Changed
- **`BcryptCost` kini `var`, bukan `const`**: Sebagai `const` ia tidak dapat ditimpa dari luar paket — tidak lewat `TestMain`, tidak lewat build tag, tidak lewat apa pun selain fork atau `replace`. Bawaannya tetap `12`, sehingga tidak ada perubahan perilaku bagi yang tidak menyentuhnya. Closes [#18](https://github.com/dimframework/dim/issues/18).
  - ⚠️ **Catatan kompatibilitas**: `var` tidak dapat dipakai dalam ekspresi konstanta. Kode seperti `const myCost = dim.BcryptCost` atau `[dim.BcryptCost]byte{}` berhenti terkompilasi dan perlu diganti dengan variabel biasa. Pemakaian sebagai nilai — yang praktis mencakup semua pemakaian nyata — tidak terdampak.
  - **Alasannya biaya di bawah `-race`**: bcrypt Go murni adalah gelung akses memori yang rapat, dan race detector menginstrumentasi setiap akses. Terukur di mesin pengembang, **per hash**: cost 12 → 216 ms tanpa `-race` tetapi **1,91 dtk** dengan `-race`; cost 6 → 2,9 ms dan 29,6 ms. Test yang menembak endpoint login sampai rate limit menutup, atau yang menguji kesetaraan waktu-tanggap antara email tak dikenal dan kata sandi salah, memanggil bcrypt puluhan kali — cukup untuk menembus batas 10 menit bawaan `go test` dan mati sebagai `panic: test timed out` alih-alih melaporkan hasil.
  - Pemakai menurunkannya sekali di `TestMain`: `dim.BcryptCost = 6`. Sebaiknya berhenti di 6 atau 8 — pada cost 4 test kesetaraan waktu-tanggap masih lulus tetapi selisih yang diukurnya menyusut mendekati derau.
  - **Sengaja `var`, bukan dibaca dari environment.** Cost yang dapat disetel lewat env membuat satu salah ketik di produksi melemahkan setiap kata sandi — tanpa galat, tanpa gejala, dan baru ketahuan setelah basis data bocor. Sebagai `var`, ia hanya berubah oleh kode yang memang ditulis untuk mengubahnya.
  - Nilainya dibaca setiap kali `HashPassword` dipanggil. Setel sekali sebelum melayani request; mengubahnya saat request berjalan adalah data race.
- **`HashPassword` menolak `BcryptCost` di luar rentang `4..31`**: Konsekuensi langsung dari membuat nilainya dapat ditulis. bcrypt menukar cost di bawah `MinCost` dengan `DefaultCost`-nya sendiri (`10`) tanpa mengembalikan galat, sehingga `dim.BcryptCost = cfg.BcryptCost` dengan field yang tidak terisi — bernilai `0` — akan menghasilkan cost `10`, bukan `12`, tanpa galat dan tanpa gejala. Persis jenis pelemahan senyap yang alasan `var` ini justru ingin dihindari, jadi salah setel kini berbunyi. Cost di atas `31` memang sudah ditolak bcrypt sendiri; kini keduanya berperilaku sama.

### Testing
- **Test untuk `BcryptCost`** (baru): menjaga bawaan tetap `12` lewat konstanta `defaultBcryptCost` — diperiksa terhadap konstanta, bukan terhadap nilai saat test berjalan, agar tetap bermakna bila suatu saat paket ini sendiri menurunkan cost lewat `TestMain`. Ditambah satu test yang memastikan nilainya benar-benar sampai ke hash (dibaca ulang lewat `bcrypt.Cost`), dan satu yang menjaga penolakan cost di luar rentang.

---

## [v0.9.0] - 2026-08-08

### Added
- **`Router.RedirectTrailingSlash(enabled bool)`** (baru): Mengarahkan path yang tidak cocok ke padanannya yang hanya berbeda pada slash di akhir — `/users/7/` → `/users/7`, dan sebaliknya bila pola memang didaftarkan dengan slash. **Nonaktif secara default**, sehingga tidak ada perubahan perilaku bagi yang tidak memakainya.
  - Redirect hanya dilakukan bila padanannya benar-benar punya handler untuk metode yang sama. `POST /users/7/` pada route yang hanya melayani `GET` tetap `404` alih-alih diarahkan ke path yang juga akan gagal.
  - `301` untuk GET dan HEAD agar mesin pencari mengambil satu URL kanonik; `308` untuk metode lain karena `301` membuat sebagian klien mengubah `POST` menjadi `GET`. Query string ikut terbawa, dan `/` tidak pernah diredirect ke path kosong.
  - `405` didahulukan: path yang memang terdaftar dengan metode lain dijawab `405`, bukan diredirect. Setelah itu barulah redirect diperiksa, dan itu pun masih sebelum fallback `Static()` dan `SPA()` — tanpa urutan itu, wildcard `GET /{path...}` milik SPA akan menelan setiap path sebelum sempat diredirect.
  - Seperti route pada umumnya, registrasi `GET` tidak otomatis melayani `HEAD`; daftarkan `router.Head(...)` bila klien memvalidasi URL lewat HEAD.

### Fixed
- **Route berparameter melayani path yang lebih dalam**: `matchInternal` mengembalikan endpoint node saat ini tanpa memeriksa apakah path sudah habis dikonsumsi, sehingga `/m/{slug}` juga menjawab `/m/abc/x/y/z` dan `/campaign/{slug}/donate` menjawab `/campaign/x/donate/lagi`. URL yang tidak terdaftar merender halaman yang tampak sah alih-alih `404`, dan perayap dapat mengindeks URL tak terhingga di bawah satu route. Closes [#15](https://github.com/dimframework/dim/issues/15).
  - Endpoint node kini hanya dipakai bila `path == ""`. Route tanpa parameter tidak terdampak karena dilayani peta statis, bukan pohon radix — itu pula sebabnya test yang ada tidak menangkap bug ini.
  - ⚠️ **Perubahan perilaku**: `/m/abc/` (trailing slash) pada `/m/{slug}` kini `404`, konsisten dengan route statis yang memang sudah begitu. Untuk melayani seluruh sub-path, daftarkan catch-all secara eksplisit: `/m/{slug}/{rest...}`.
  - Catatan: path yang tidak cocok tetap diteruskan ke `Static()` dan `SPA()`. Pada aplikasi yang memakai `SPA()`, path tersebut menerima `index.html` lewat wildcard `GET /{path...}`, bukan `404` — sebagaimana mestinya untuk route sisi klien.
- **`405` dari cabang yang cocok sebagian menutupi cabang lain yang benar-benar cocok**: Penelusuran berhenti begitu sebuah cabang mengembalikan daftar metode, sehingga dengan `POST /user/new/{id}` dan `GET /user/{name}/{id}` terdaftar, `GET /user/new/5` dijawab `405` padahal ada handler yang cocok. Daftar metode kini dikumpulkan dan baru dilaporkan bila tidak ada satu pun cabang yang menghasilkan handler.
  - Parameter dari cabang yang gagal juga ikut dibersihkan; sebelumnya nilai yang tertinggal bisa terbawa ke handler yang akhirnya cocok.
- **Header `Allow` hanya memuat metode dari cabang pertama**: Pada contoh di atas, `DELETE /user/new/5` menjawab `405` dengan `Allow: POST` dan menyembunyikan `GET`. Metode kini digabungkan dari seluruh cabang yang path-nya cocok, lalu diurutkan dan dide-duplikasi, sesuai RFC 7231 §7.4.1.
- **`405` dari peta statis menutupi route pohon**: `serveTree` mengembalikan `405` begitu path ditemukan di peta statis dengan metode lain, tanpa pernah menengok pohon radix. Dengan `POST /user/new` dan `GET /user/{name}` terdaftar, `GET /user/new` dijawab `405` padahal handler `{name}` cocok — bug yang sama persis dengan kasus di `matchInternal`, hanya satu tingkat lebih atas. Metode dari peta statis kini menjadi kandidat `405` yang digabung dengan hasil pohon, dan `405` baru dikembalikan bila keduanya tidak menghasilkan handler. Header `Allow` ikut menggabungkan metode dari kedua jalur.
- **Anak parameter kedua pada level yang sama tak pernah terjangkau**: Pencocokan hanya mencoba anak parameter pertama, sedangkan pola yang hanya berbeda nama parameternya menghasilkan node terpisah. Dengan `/a/{id}` dan `/a/{slug}/edit` terdaftar, `/a/5/edit` tidak pernah sampai ke handler-nya. Seluruh anak parameter (dan catch-all) kini dicoba dengan backtracking, seperti anak statis.

### Testing
- **`router_tree_test.go`** (baru): Menguji pohon radix secara terpisah dari peta statis — kedalaman path, trailing slash, catch-all, `405` beserta header `Allow`, backtracking antar cabang, dan kebocoran parameter. Jalur pencocokan keduanya berbeda, sehingga test yang hanya memakai route statis tidak akan pernah menyentuh pohon.
- **`router_redirect_test.go`** (baru): Menguji `RedirectTrailingSlash` — default nonaktif, kedua arah redirect, query string, `308`, syarat metode, prioritas `405`, root, urutan terhadap fallback SPA, dan toggle konkuren di bawah `-race`.

---

## [v0.8.2] - 2026-08-03

### Fixed
- **`LoggerMiddleware` melumpuhkan SSE, streaming, dan WebSocket**: Pembungkus `http.ResponseWriter` yang dipakai untuk menangkap status code tidak menyediakan `Unwrap()` dan tidak meneruskan antarmuka opsional. Karena middleware ini lazim dipasang global lewat `router.Use`, seluruh handler di aplikasi kehilangan `http.Flusher`, `http.Hijacker`, dan `io.ReaderFrom`. Closes [#16](https://github.com/dimframework/dim/issues/16).
  - **SSE dan streaming**: tanpa `Flush`, respons tertahan di buffer sampai handler selesai. `http.NewResponseController(w).Flush()` pun mengembalikan `feature not supported` karena `Unwrap()` tidak ada — sehingga tidak ada jalan keluar dari sisi pemakai.
  - **WebSocket**: `w.(http.Hijacker)` gagal, sehingga `gorilla/websocket` dan `coder/websocket` menolak koneksi. Keduanya memakai type assertion langsung, bukan `ResponseController`.
  - **File statis**: `io.ReaderFrom` yang hilang membuat `Router.Static` kehilangan jalur cepat `sendfile` dan jatuh ke salin-buffer.
  - Pembungkus kini menyediakan `Unwrap()` dan memilih varian sesuai kemampuan writer aslinya, sehingga antarmuka yang dimiliki diteruskan apa adanya — dan yang tidak dimiliki tidak dipalsukan. Di HTTP/2 yang tanpa `Hijacker`, `w.(http.Hijacker)` tetap gagal sebagaimana mestinya alih-alih berhasil lalu error saat dipanggil.
  - `http.Pusher` tidak diteruskan (HTTP/2 server push tidak lagi didukung peramban arus utama); tetap terjangkau lewat `Unwrap()`.

### Changed
- **Log request yang koneksinya diambil alih ditandai `hijacked=true`**: Setelah `Hijack`, koneksi keluar dari kendali `net/http` dan status yang tercatat tidak lagi mewakili apa pun yang dikirim ke klien.

---

## [v0.8.1] - 2026-08-02

### Fixed
- **`DatabaseRateLimitStore.Allow` rusak di kedua driver**: Query UPSERT mengirim 5 argumen untuk teks query yang hanya punya 4 placeholder berbeda. Akibatnya rate limiting berbasis database tidak berefek sama sekali sejak diperkenalkan — paling berdampak pada proteksi brute force di endpoint login dan forgot-password.
  - **PostgreSQL**: pgx menolak setiap panggilan dengan `expected 4 arguments, got 5`. Karena `RateLimit` fail open, error ini membuat seluruh request diteruskan tanpa pembatasan.
  - **SQLite**: tidak ada error, tetapi `Rebind` mengganti setiap `$N` dengan `?` secara posisional sehingga placeholder yang dipakai ulang menggeser pemetaan argumen. `expires_at` ditimpa dengan waktu yang sudah lewat, sehingga counter ter-reset di setiap request berikutnya dan limit tidak pernah tercapai.
  - Setiap placeholder kini muncul tepat sekali dan argumen dikirim mengikuti urutan kemunculannya. Nilai yang sama dikirim dua kali sebagai placeholder terpisah.

### Changed
- **`RateLimit` kini mencatat kegagalan store**: Blok fail open sebelumnya kosong — hanya berisi komentar — sehingga rate limit yang mati tidak meninggalkan jejak apa pun. Kegagalan store kini dicatat via `slog.Default()` pada level Error, untuk cakupan IP maupun user. Perilaku fail open tetap dipertahankan agar API tidak ikut tumbang saat DB bermasalah.

### Testing
- **Test Postgres untuk `DatabaseRateLimitStore`** (baru): Ketiadaan test inilah yang membuat bug di atas lolos ke rilis.
- **Test SQLite lintas batas detik** (baru): Test SQLite yang lama menjalankan seluruh panggilannya dalam satu detik yang sama, sehingga salah-peta argumen kebetulan tidak berdampak dan tidak tertangkap.
- **CI kini menjalankan service PostgreSQL**: Sebelumnya seluruh test integrasi Postgres skip diam-diam karena `TEST_PG_HOST` tidak pernah di-set. Ditambahkan pula gate `gofmt` dan `go vet`.

---

## [v0.8.0] - 2026-08-02

### Security
- **`GetClientIP` sekarang aman secara default**: Fungsi tidak lagi membaca header `X-Forwarded-For`, `X-Real-IP`, atau `X-Forwarded`. Header-header ini dapat diisi sembarang oleh klien dan tidak dapat dipercaya tanpa verifikasi proxy. Sekarang `GetClientIP` hanya menggunakan `r.RemoteAddr`. Closes [#12](https://github.com/dimframework/dim/issues/12).
  - ⚠️ **Breaking Change**: Kode yang mengandalkan `GetClientIP` untuk membaca IP dari header proxy harus memasang middleware `ClientIP` dan menyetel `TRUSTED_PROXY_COUNT`.
  - `Ctx.ClientIP()` juga ikut berubah — tanpa middleware `ClientIP` sekarang mengembalikan `RemoteAddr`.
- **`GetClientIPWithTrustedProxies(r, trustedProxyCount int) string`** (baru): Fungsi pengganti untuk aplikasi yang berjalan di belakang proxy tepercaya. Membaca `X-Forwarded-For` dari kanan ke kiri sebanyak `trustedProxyCount` hop, sehingga entri yang ditambahkan klien di sebelah kiri tidak ikut dipercaya. Seluruh baris `X-Forwarded-For` digabungkan lebih dulu, karena sebagian proxy menambahkan baris header baru alih-alih menyambung ke baris yang ada.
  - **Contoh**: `dim.GetClientIPWithTrustedProxies(r, 1)` untuk aplikasi di belakang satu load balancer (Cloud Run, nginx, dll).
  - Jatuh kembali ke `RemoteAddr` bila `trustedProxyCount <= 0`, header tidak ada, jumlah entri lebih sedikit daripada `trustedProxyCount`, atau nilai pada posisi tersebut bukan IP yang sah.
  - ⚠️ Hanya aman bila aplikasi tidak dapat dihubungi langsung — seluruh trafik wajib melewati proxy tepercaya.
- **`ClientIPConfig` + middleware `ClientIP(config)`** (baru): Konfigurasi resolusi IP klien tingkat aplikasi, bukan per middleware. `ClientIP` meresolusi IP satu kali di awal request dan menyimpannya ke request context, sehingga rate limiting, logging, audit trail, `GetClientIP`, dan `Ctx.ClientIP()` semuanya membaca nilai yang sama.
  - Env: `TRUSTED_PROXY_COUNT` (default `0`). Nilai negatif ditolak saat `LoadConfig()`.
  - **Sebelum** (rentan): `PerIP` rate limit dapat dilewati dengan mengirim header `X-Forwarded-For` sembarang.
  - **Sekarang** (aman): Tanpa middleware `ClientIP`, seluruh komponen memakai `RemoteAddr`. Di belakang proxy, pasang `router.Use(dim.ClientIP(cfg.ClientIP))` sedini mungkin dalam chain.
- **`SetClientIP(r, ip)`** (baru): Menyimpan IP klien yang sudah diresolusi ke request context, mengikuti pola `SetRequestID`.
- **`Ctx.Request()`** dan **`Ctx.ResponseWriter()`** (baru): Akses ke `*http.Request` / `http.ResponseWriter` yang dibungkus, untuk kebutuhan yang belum punya helper di `Ctx`.

---

## [v0.7.3] - 2026-07-19

### Added
- **`Gone(w, message)`** / **`Ctx.Gone(message)`**: Mengirim response 410 Gone. Berguna untuk resource yang sudah tidak berlaku secara permanen seperti one-time link/token yang sudah expired. Closes [#10](https://github.com/dimframework/dim/issues/10).
- **`UnprocessableEntity(w, message, errors)`** / **`Ctx.UnprocessableEntity(message, errors)`**: Mengirim response 422 Unprocessable Entity. Berguna untuk request yang valid secara sintaktik tetapi melanggar aturan domain/bisnis (semantically invalid) — berbeda dari 400 (malformed) dan 409 (state conflict). Closes [#10](https://github.com/dimframework/dim/issues/10).
- **`MethodNotAllowed(w, message)`** / **`Ctx.MethodNotAllowed(message)`**: Mengirim response 405 Method Not Allowed. Berguna ketika endpoint ada tetapi HTTP method yang digunakan tidak didukung.
- **`ServiceUnavailable(w, message)`** / **`Ctx.ServiceUnavailable(message)`**: Mengirim response 503 Service Unavailable. Berguna ketika server tidak dapat menangani request sementara waktu, misalnya karena database tidak tersedia atau mode maintenance.

---

## [v0.7.2] - 2026-06-20

### Added
- **`PostgresDatabase.WritePool()` dan `ReadPools()` accessors**: Mengekspos `*pgxpool.Pool` yang mendasari `PostgresDatabase` untuk integrasi lanjutan — memungkinkan penggunaan pustaka yang membutuhkan akses `pgx` langsung seperti job queue (`riverqueue/river`) dan `LISTEN/NOTIFY` broker. Konsisten dengan pola escape-hatch `PostgresTx.PgxTx()` yang sudah ada. Interface `Database` tidak berubah; accessor hanya tersedia di tipe konkret `*PostgresDatabase` dan diakses via type assertion. Closes [#8](https://github.com/dimframework/dim/issues/8).

---

## [v0.7.1] - 2026-06-11

### Changed
- **`Validator.ErrorMap()` return type**: Changed from `map[string]string` to `FieldErrors` (type alias for `map[string]any`). This allows seamless integration with `BadRequest()` and `JsonError()` — no adapter function needed. All signatures updated; `FieldErrorsFrom()` is now redundant and can be removed in application code.
  - **Before**: `dim.BadRequest(w, msg, dim.FieldErrorsFrom(v.ErrorMap()))`
  - **After**: `dim.BadRequest(w, msg, v.ErrorMap())`
  - ⚠️ **Breaking Change**: Code that type-asserts `v.ErrorMap()` as `map[string]string` will panic. Type-assert as `map[string]any` instead, or use the typed accessor methods `v.Error(field string)` and `v.Errors(field string) []string`.

### Added
- **`Validator.WithFullErrors()` method**: Enables accumulating all errors per field instead of first-error-wins. Can be chained at start, middle, or end of the validation chain. After `WithFullErrors()` is called, subsequent rules collect all errors in `map[string][]string` internally and merge them into `FieldErrors` on `ErrorMap()`.
  - **Example**: `v := dim.NewValidator().WithFullErrors().Required(...).Email(...)`
  - Errors are returned as `FieldErrors{"field": []string{"error1", "error2"}}`

### Updated
- **Docs**: Updated `13-validation.md`, `14-error-handling.md`, and `07-response-helpers.md` to reflect the new `ErrorMap()` behavior, `WithFullErrors()` patterns, and removed `FieldErrorsFrom()` adapter usage.

---

## [v0.7.0] - 2026-06-08

### Added
- **`Ctx` helper (`dim.Of`)**: Added opt-in ergonomic wrapper that bundles `http.ResponseWriter` and `*http.Request` into a single `*Ctx` object. Reduces boilerplate in handlers that call many helpers — use `c := dim.Of(w, r)` and replace `dim.GetParam(r, "id")` with `c.Param("id")`, `dim.OK(w, data)` with `c.OK(data)`, etc. No breaking changes; existing handlers continue to work unchanged. Closes [#6](https://github.com/dimframework/dim/issues/6).
  - Request helpers: `Param`, `Query`, `Queries`, `Header`, `Cookie`, `AuthToken`, `User`, `Claims`, `RequestID`, `ClientIP`
  - `Bind(&v)` — decodes JSON request body into a struct
  - `Validate()` — shorthand for `dim.NewValidator()`
  - Response helpers: `JSON`, `OK`, `Created`, `NoContent`, `BadRequest`, `Unauthorized`, `Forbidden`, `NotFound`, `Conflict`, `InternalServerError`, `TooManyRequests`, `AppError`
- **`Ctx` docs**: Added "Ctx Helper — Ergonomic Syntax" section to `docs/07-response-helpers.md` with method tables, side-by-side comparison, and a complete `CreateUser` handler example.

---

## [v0.6.2] - 2026-06-04

### Changed
- **Pure Go SQLite driver**: Replaced `github.com/mattn/go-sqlite3` (CGO) with `modernc.org/sqlite` (pure Go). No CGO toolchain required — builds work out of the box on all platforms without a C compiler. API and behavior are unchanged.

---

## [v0.6.1] - 2026-05-30

### Added
- **`WithClaimsProvider` tests**: Added comprehensive tests for `WithClaimsProvider` to verify custom claims are correctly embedded in access tokens.
- **Authentication & Token API docs**: Added full API reference section for `AuthService`, `JWTManager`, `BrancaManager`, `ClaimsProvider`, and `Authenticatable` in `docs/23-api-reference.md`.
- **`WithClaimsProvider` docs**: Added usage guide for `WithClaimsProvider` in `docs/12-authentication.md`.

### Fixed
- **`BRANCA_KEY` not validated at startup**: `Validate()` now decodes `BRANCA_KEY` at startup and returns a descriptive error if the key is invalid (wrong length or format). Previously, an invalid key was only caught at runtime when `NewBrancaManager` was called.
- **`JWT_SECRET` always required even when using Branca**: `Validate()` no longer requires `JWT_SECRET` when `BRANCA_KEY` is set. If `BRANCA_KEY` is present, Branca is treated as the active token provider and JWT validation is skipped entirely.

---

## [v0.6.0] - 2026-05-07

### Added
- **Branca Token Provider**: Added `BrancaManager` as an alternative to `JWTManager`. Branca tokens encrypt the payload using XChaCha20-Poly1305 — claims are unreadable by the client, suitable for sensitive payloads or internal services.
- **`TokenManager` Interface**: Introduced `TokenManager` interface and `TokenClaims` type alias to abstract token operations. Both `JWTManager` and `BrancaManager` implement this interface, enabling provider switching without changing application code.
- **`NewAuthServiceWithManager`**: New constructor for `AuthService` that accepts any `TokenManager` implementation, enabling Branca (or future providers) to be used with `AuthService`.
- **`BrancaConfig`**: New config struct with `BRANCA_KEY`, `BRANCA_ACCESS_TOKEN_EXPIRY`, and `BRANCA_REFRESH_TOKEN_EXPIRY` environment variables.
- **Base64 PEM support for JWT keys**: `JWT_PRIVATE_KEY` and `JWT_PUBLIC_KEYS` now accept base64-encoded PEM content in addition to file paths and raw PEM strings — recommended for Docker/Kubernetes environments where newlines in env vars are problematic.
- **Hybrid static map + radix tree router**: Replaced `http.ServeMux` routing backend with a two-tier dispatch. Static routes (no URL parameters) are stored in an O(1) map; dynamic routes (`{param}`, `{path...}`) use a chi-style radix tree (O(k) per path segment). `http.ServeMux` is retained as a fallback for `Static()` and `SPA()` file serving only.
- **Migration database connection**: Added `NewMigrationDatabase` constructor and `DB_MIGRATION_HOST/PORT/USERNAME/PASSWORD` env vars for a dedicated migration database connection. Falls back to the Write connection when any field is unset. `WithMigrationDB` adds it to the CLI `Console`; `CommandContext.MigrationDB` exposes it inside migration commands.

### Changed
- **`RequireAuth` and `OptionalAuth`**: Parameter type changed from `*JWTManager` to `TokenManager` interface. Fully backward compatible — existing code passing `*JWTManager` continues to work unchanged.
- **`JWTManager.VerifyToken`**: Return type changed from `jwt.MapClaims` to `TokenClaims` (a type alias for `map[string]interface{}`). Fully backward compatible for map access patterns.

### Fixed
- **Branca base62 leading zeros**: `brancaBase62Encode`/`brancaBase62Decode` now preserve leading zero bytes in the token binary, preventing silent payload corruption when the header starts with `0x00`.
- **`decodeBrancaKey` ambiguous format detection**: Length guards ensure a 32-character raw key is never misinterpreted as base64, and base64 variants are only attempted at their canonical lengths (44 for std, 43 for raw-URL).
- **Branca reserved claim protection**: `GenerateAccessToken` now returns an error if `extraClaims` contains a reserved key (`sub`, `sid`, `jti`, `email`, `iat`, `exp`, `nbf`, `typ`), preventing silent overwrite of internal claims.

---

## [v0.5.0] - 2026-02-10

### Added
- **Auth Middleware Flexibility**: Added Functional Options pattern to `RequireAuth` middleware (`WithBearerToken`, `WithCookieToken`) allowing token extraction from Headers or Cookies.
- **CORS Support for Exposed Headers**: Added `ExposedHeaders` to `CORSConfig` and support for `CORS_EXPOSED_HEADERS` environment variable.
- **CSRF Token Expiration**: Added `CookieMaxAge` to `CSRFConfig` and `CSRF_COOKIE_MAX_AGE` environment variable (default: 12 hours) to allow CSRF cookies to expire.
- **CORS Vary Header**: Added `Vary: Origin` header to CORS responses to prevent cache poisoning.

### Changed
- **CSRF Error Code**: Changed CSRF validation failure status code from `403 Forbidden` to `419 Authentication Timeout` (Custom Status) to better distinguish CSRF issues from permission issues.
- **CORS Preflight Status**: Changed CORS preflight response status from `200 OK` to `204 No Content`.
- **CORS Logic**: Updated CORS middleware to pass through non-CORS `OPTIONS` requests (requests without `Origin` header) instead of swallowing them.
- **CORS Max-Age**: Fixed bug in `Access-Control-Max-Age` header where integer value was incorrectly converted to string.
- **Documentation**: Updated `middleware.md`, `configuration.md`, and `security.md` with new configuration options and best practices.
