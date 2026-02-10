# Books Backend

Personal book library API: upload EPUB/PDF (EPUBs stored in S3, metadata from Open Library by ISBN in MongoDB), and simple email/password auth.

## Setup

1. Copy `.env.example` to `.env` and set:
   - `MONGODB_URI`, `MONGODB_DB`
   - `AWS_S3_BUCKET`, `AWS_REGION` (and AWS credentials for S3)
   - `AUTH_EMAIL`, `AUTH_PASSWORD` (predefined login)
   - `JWT_SECRET`

2. Run MongoDB locally or use a hosted instance.

3. Build and run:
   ```bash
   go mod tidy
   go run .
   ```

Server listens on `PORT` (default 8080).

## API

- **GET /** – Health/welcome
- **POST /api/auth/login** – Body: `{"email":"...","password":"..."}`. Returns `{"token":"...","email":"..."}`. Use the token in `Authorization: Bearer <token>` for protected routes.
- **POST /api/upload** – (Auth) Multipart form field `file`: EPUB or PDF. EPUBs are parsed for ISBN and metadata is fetched from Open Library and stored in MongoDB; PDFs are stored in S3 with minimal record. Files are stored in S3 under `{userId}/{uuid}.epub|.pdf`.
- **GET /api/books** – (Auth) List the current user’s books (metadata from MongoDB).

## Auth

Currently a single predefined user: set `AUTH_EMAIL` and `AUTH_PASSWORD` in `.env`. If no user exists in the DB, login with those credentials creates the user. Register and additional auth flows can be added later.
