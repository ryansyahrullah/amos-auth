# AMK API Go

REST API for AMK built with Go (Golang) using Modular Monolith architecture.

## Features

- **Authentication**: Login (NRP/Email), JWT Token, Forgot Password (OTP).
- **User Management**: CRUD, Role-based Access Control (RBAC).
- **Dynamic Permissions**: Granular permission management and assignment to roles.
- **HCGS Integration**: Automatic creation of Employee data (`hc_pegawai`) upon User creation.
- **File Uploads**: Automatic image resizing for `hca_berkas` uploads.
- **Roles**: Super Admin, Direktur, Admin HCGS, Admin FAT, Pegawai.

## Setup

1.  **Clone the repository**
2.  **Install dependencies**:
    ```bash
    go mod tidy
    ```
3.  **Configure Environment**:
    Copy `.env.example` to `.env` and update the values.
    ```bash
    cp .env.example .env
    ```
    Make sure to set `DB_USER`, `DB_PASSWORD`, `DB_NAME`, and `EMAIL_PASSWORD`.
4.  **Run the Application**:

    ```bash
    go run cmd/api/main.go
    ```

5.  **Seeded Credentials**:
    - **Admin**: `syahrullahryan@gmail.com` / `@Ryan0852`
    - **Operator**: `operator@amk.com` / `password123` (NRP: OPR-001)

- `POST /auth/forgot-password`: Request OTP for password reset.
  ```json
  {
    "email": "syahrullahryan@gmail.com"
  }
  ```
- `POST /auth/reset-password`: Reset password using OTP.
  ```json
  {
    "email": "syahrullahryan@gmail.com",
    "otp": "123456",
    "new_password": "NewPassword123"
  }
  ```

### Users

- `POST /users/`: Create a new user (Super Admin or Admin HCGS).
  ```json
  {
    "email": "pegawai@amk.com",
    "password": "password123",
    "role": "pegawai",
    "nrp": "AM121002",
    "full_name": "Budi Santoso"
  }
  ```
- `GET /users/`: Get all users (Super Admin, Direktur, Admin HCGS, Admin FAT).
  - Headers: `Authorization: Bearer <token>`

### HCGS Module

#### Get Current User Employee Data

- **URL**: `/hcgs/me`
- **Method**: `GET`
- **Headers**: `Authorization: Bearer <token>`
- **Response**: Returns full employee data including attributes and history.

#### Common Response for DELETE Operations

All DELETE endpoints return a JSON object with a success message upon successful deletion:

```json
{
  "message": "Resource deleted successfully"
}
```

#### Master Data (Admin Only)

- **Departemen**:
  - `GET /hcgs/master/departemen`
  - `POST /hcgs/master/departemen`
    ```json
    {
      "nama_departemen": "Human Capital",
      "kode_departemen": "HC001"
    }
    ```
  - `DELETE /hcgs/master/departemen/:id`
- **Jabatan**:
  - `GET /hcgs/master/jabatan`
  - `POST /hcgs/master/jabatan`
    ```json
    {
      "id_departemen": 1,
      "nama_jabatan": "Manager",
      "level_jabatan": "Managerial"
    }
    ```
  - `DELETE /hcgs/master/jabatan/:id`
- **Job Site**:
  - `GET /hcgs/master/job-site`
  - `POST /hcgs/master/job-site`
    ```json
    {
      "nama_site": "Head Office Jakarta",
      "alamat_site": "Jl. Sudirman No. 1",
      "latitude": -6.2088,
      "longitude": 106.8456
    }
    ```
  - `DELETE /hcgs/master/job-site/:id`
- **Kontrak**:
  - `GET /hcgs/master/kontrak`
  - `POST /hcgs/master/kontrak`
    ```json
    {
      "jenis_kontrak": "PKWT",
      "keterangan": "Perjanjian Kerja Waktu Tertentu"
    }
    ```
  - `DELETE /hcgs/master/kontrak/:id`

#### Attributes (User Data)

- **Detail Personal**: `/hcgs/detail-personal` (GET, POST)
  ```json
  {
    "alamat_ktp": "Jl. Merdeka No. 10",
    "alamat_domisili": "Jl. Sudirman No. 5",
    "tempat_lahir": "Jakarta",
    "tanggal_lahir": "1990-01-01T00:00:00Z",
    "tinggi_badan": 175.5,
    "berat_badan": 70.0,
    "golongan_darah": "O",
    "agama": "Islam",
    "status_pernikahan": "Menikah",
    "tanggal_pernikahan": "2015-05-20T00:00:00Z",
    "telepon_utama": "081234567890",
    "telepon_alternatif": "081987654321",
    "pendidikan_terakhir": "S1 Teknik Informatika",
    "pekerjaan_terakhir": "Software Engineer",
    "ukuran_sepatu_safety": "42",
    "ukuran_baju": "L",
    "ukuran_celana": "32"
  }
  ```
- **Identitas**: `/hcgs/identitas` (GET, POST)
  ```json
  {
    "no_ktp": "3171234567890001",
    "no_kk": "3171234567890002",
    "no_npwp": "12.345.678.9-012.000",
    "no_bpjs_kesehatan": "00012345678",
    "no_bpjs_ketenagakerjaan": "12345678901",
    "nama_bank": "BCA",
    "no_rekening": "1234567890",
    "nama_pemilik_rekening": "Budi Santoso"
  }
  ```
- **Keluarga**:
  - `GET /hcgs/keluarga`
  - `POST /hcgs/keluarga`
    ```json
    {
      "nama_anggota": "Siti Aminah",
      "hubungan": "Istri",
      "anak_ke_berapa": 0,
      "jumlah_saudara": 0,
      "apakah_tanggungan": true
    }
    ```
  - `DELETE /hcgs/keluarga/:id`
- **Kontak Darurat**:
  - `GET /hcgs/kontak-darurat`
  - `POST /hcgs/kontak-darurat`
    ```json
    {
      "nama_kontak": "Bambang",
      "hubungan": "Ayah",
      "alamat_domisili": "Jl. Gatot Subroto No. 10",
      "no_telepon": "081122334455"
    }
    ```
  - `DELETE /hcgs/kontak-darurat/:id`
- **Ahli Waris**:
  - `GET /hcgs/ahli-waris`
  - `POST /hcgs/ahli-waris`
    ```json
    {
      "nama_ahli_waris": "Rizky Santoso",
      "tempat_lahir": "Bandung",
      "tanggal_lahir": "2016-06-15T00:00:00Z",
      "hubungan": "Anak",
      "alamat_domisili": "Jl. Sudirman No. 5",
      "no_telepon": "081234567890"
    }
    ```
  - `DELETE /hcgs/ahli-waris/:id`
- **Berkas**:
  - `GET /hcgs/berkas`
  - `POST /hcgs/berkas` (Multipart/Form-Data)
    - **Fields**:
      - `file`: The image file (jpg, png). Automatically resized to max width 1024px.
      - `jenis_berkas`: Type of document (e.g., "KTP", "Foto Profil").
    - **Response**:
      ```json
      {
        "ID": 1,
        "IDPegawai": 5,
        "JenisBerkas": "KTP",
        "PathFile": "uploads/berkas/1698400000_berkas.jpg",
        "TanggalUpload": "2023-10-27T10:00:00Z"
      }
      ```
  - `DELETE /hcgs/berkas/:id`

#### History

- **Riwayat Kontrak**:
  - `GET /hcgs/riwayat-kontrak`
  - `POST /hcgs/riwayat-kontrak`
    ```json
    {
      "no_kontrak": "KONTRAK/2023/001",
      "id_kontrak": 1,
      "tanggal_awal": "2023-01-01T00:00:00Z",
      "tanggal_akhir": "2024-01-01T00:00:00Z",
      "status_aktif": true
    }
    ```
  - `DELETE /hcgs/riwayat-kontrak/:id`

## Roles & Permissions

The system uses a dynamic permission model where permissions can be assigned to roles.

        "permission_id": 2

## Project Structure

- `cmd/api`: Entry point.
- `internal/auth`: Authentication logic.
- `internal/user`: User management.
- `internal/hcgs`: HCGS (Employee) data.
- `internal/platform`: Shared utilities (DB, JWT, Email).

## API Summary & Functions

This API serves as the backend for the AMK system, providing the following core functionalities:

### 1. Authentication Module (`/auth`)

- **Login**: Authenticates users via Email or NRP and returns a JWT token.
- **Forgot Password**: Initiates password reset flow by sending an OTP to the user's email.
- **Reset Password**: Verifies OTP and allows users to set a new password.

### 2. User Management Module (`/users`)

- **User CRUD**: Create, Read, Update, and Delete users.
- **Role Management**: Assign roles to users (Super Admin, Direktur, Admin HCGS, Admin FAT, Pegawai).
- **Permission Management**:
  - **Create Permission**: Define new permissions dynamically.
  - **List Permissions**: View all available permissions.
  - **Assign Permission**: Grant specific permissions to roles.

### 3. HCGS (Human Capital) Module (`/hcgs`)

- **Employee Data**: Automatically manages `hc_pegawai` records linked to users.
- **Personal Attributes**: Manages detailed employee information:
  - **Personal Details**: Address, birth date, physical attributes.
  - **Identity**: KTP, KK, NPWP, BPJS numbers.
  - **Family**: Spouse and children data.
  - **Emergency Contacts**: Contact info for emergencies.
  - **Heirs**: Beneficiary information.
- **Documents (`Berkas`)**: Handles file uploads (KTP, Photos, etc.) with **automatic image resizing** to optimize storage.
- **History**: Tracks contract history (`Riwayat Kontrak`).

#### Slip Gaji (Salary Slips)

- **Employee**:
  - `GET /hcgs/slip-gaji/me`: Get latest slip.
  - `GET /hcgs/slip-gaji/me/history`: Get slip history.
  - `GET /hcgs/slip-gaji/me/html`: View slip as HTML.
  - `GET /hcgs/slip-gaji/me/pdf`: Download slip as PDF.
  - `GET /hcgs/dashboard/stats`: Get dashboard statistics (e.g., total slips, announcements).
- **Admin**:
  - `POST /hcgs/slip-gaji/upload`: Upload Excel file for bulk slip creation.
  - `GET /hcgs/slip-gaji/draft/pending`: List pending upload batches.
  - `GET /hcgs/slip-gaji/draft/:batchID`: View draft slips in a batch.
  - `GET /hcgs/slip-gaji/draft/:batchID/validate`: Validate a batch before submission.
  - `POST /hcgs/slip-gaji/draft/:batchID/submit`: Submit a batch to finalize slips.
  - `DELETE /hcgs/slip-gaji/draft/:batchID`: Delete a draft batch.
  - `GET /hcgs/slip-gaji`: List all finalized slips.
  - `GET /hcgs/slip-gaji/template`: Download Excel template for uploads.

#### Pengumuman (Announcements)

- `GET /hcgs/pengumuman`: List all announcements.
- `POST /hcgs/pengumuman`: Create a new announcement (supports file attachment).
- `PUT /hcgs/pengumuman/:id`: Update an announcement.
- `DELETE /hcgs/pengumuman/:id`: Delete an announcement.
- `GET /hcgs/pengumuman/:id/file`: Download attached file.

- **Master Data**: Manages reference data for Departments, Job Titles, Job Sites, and Contract Types.

### 4. Platform & Security

- **JWT Authentication**: Secures all protected endpoints.
- **RBAC & Dynamic Permissions**: Enforces access control based on assigned roles and granular permissions.
- **Middleware**: Custom middleware for logging, error handling, and permission checks.
