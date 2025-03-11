# httpxsnap

httpxsnap is a powerful tool designed for bug bounty hunters, pen-testers, and developers to efficiently scan multiple URLs, log HTTP responses, and capture screenshots for easy analysis and reporting. It automates the process of testing URLs and generates a visually appealing HTML report.

---

## Features
- Concurrently processes multiple URLs for efficient testing.
- Captures HTTP response details, including status code, content type, and length.
- Takes full-page screenshots of URLs.
- Saves HTTP response bodies as text files for in-depth analysis.
- Generates a detailed HTML report with sortable tables and clickable links to screenshots and responses.

---

## Installation

To install **httpxsnap**, use the following command:

```bash
go install -v github.com/ir4gh4v/httpxsnap@latest
```

# Usage

Run the following command to see all available options:

```bash
httpxsnap -i urls.txt -o output_folder -t 10 -rl 300
```

## Options
- **`-i string`**: Input file containing URLs (default: `file.txt`).
- **`-o string`**: Output folder (default: `output`).
- **`-t int`**: Number of threads to use (default: `20`).
- **`-rl int`**: Rate limit in milliseconds (default: `500`).

