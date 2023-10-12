# Go Upload File CSV

This is an example Go project that demonstrates uploading and processing CSV files containing account information. It includes a web server to handle file uploads and a backend service for processing the uploaded files.

## Features

- Upload CSV files containing account information.
- Process and create accounts from the uploaded files.
- Monitor the status of file processing.
- Retrieve information about processed files and their accounts.

## Prerequisites

- Go 1.11 or higher
- Git

## Getting Started

1. Clone the repository:

```bash
git clone https://github.com/kirian-dev/go-import-file-csv.git
cd go-upload-file-csv
```
2. Run server

```bash
go run ./cmd/main.go
```

### The server will be available at ```http://localhost:8080.```

## API Endpoints
- GET /api/files: Get a list of processed files.
- GET /api/files/{fileId}: Get details of a specific processed file.
- POST /api/files: Upload a CSV file for processing.

## Project Structure
- cmd/: Main application entry point.
- internal/: Internal application code.
- handlers/: HTTP request handlers.
- models/: Data models.
- services/: Business logic and file processing services.
- utils/: Helpers for working with a file.
- scripts/: Scripts for generating test CSV files.

## License
This project is licensed under the MIT License. See the LICENSE file for details.
