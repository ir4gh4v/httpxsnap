package main

import (
    "bytes"
    "context"
    "flag"
    "fmt"
    "html/template"
    "io/ioutil"
    "net/http"
    "os"
    "path/filepath"
    "sync"
    "time"

    "github.com/chromedp/chromedp"
)

type Result struct {
    Serial        int
    URL           string
    StatusCode    int
    ContentType   string
    ContentLength int64
    Screenshot    string
    ResponseBody  string
}

func main() {
    inputFile := flag.String("i", "file.txt", "Input file containing URLs")
    outputFolder := flag.String("o", "output", "Output folder")
    threads := flag.Int("t", 20, "Number of threads")
    rateLimit := flag.Int("rl", 500, "Rate limit in milliseconds")
    flag.Parse()

    screenshotDir := filepath.Join(*outputFolder, "screenshots")
    responseDir := filepath.Join(*outputFolder, "responses")
    if err := os.MkdirAll(screenshotDir, os.ModePerm); err != nil {
        fmt.Println("Error creating screenshot directory:", err)
        return
    }
    if err := os.MkdirAll(responseDir, os.ModePerm); err != nil {
        fmt.Println("Error creating response directory:", err)
        return
    }

    urls, err := readURLs(*inputFile)
    if err != nil {
        fmt.Println("Error reading input file:", err)
        return
    }

    results := make([]Result, len(urls))
    var wg sync.WaitGroup
    sem := make(chan struct{}, *threads)

    for i, url := range urls {
        wg.Add(1)
        go func(i int, url string) {
            defer wg.Done()
            sem <- struct{}{}

            results[i] = fetchDetails(i+1, url, screenshotDir, responseDir)

            time.Sleep(time.Duration(*rateLimit) * time.Millisecond)
            <-sem
        }(i, url)
    }

    wg.Wait()

    reportPath := filepath.Join(*outputFolder, "report.html")
    err = generateReport(results, reportPath)
    if err != nil {
        fmt.Println("Error generating report:", err)
        return
    }

    fmt.Println("Report generated:", reportPath)
}

func readURLs(file string) ([]string, error) {
    data, err := ioutil.ReadFile(file)
    if err != nil {
        return nil, err
    }
    lines := bytes.Split(data, []byte("\n"))
    urls := make([]string, 0, len(lines))
    for _, line := range lines {
        url := string(bytes.TrimSpace(line))
        if url != "" {
            urls = append(urls, url)
        }
    }
    return urls, nil
}

func fetchDetails(serial int, url, screenshotDir, responseDir string) Result {
    resp, err := http.Get(url)
    if err != nil {
        return Result{Serial: serial, URL: url}
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)

    responsePath := filepath.Join(responseDir, fmt.Sprintf("response-%d.txt", serial))
    err = ioutil.WriteFile(responsePath, body, 0644)
    if err != nil {
        fmt.Println("Error saving response body:", err)
    }

    contentType := resp.Header.Get("Content-Type")
    contentLength := int64(len(body))

    screenshotPath := filepath.Join(screenshotDir, fmt.Sprintf("screenshot-%d.png", serial))
    screenshot := captureScreenshot(url, screenshotPath)

    screenshot = filepath.Join("screenshots", filepath.Base(screenshot))
    responsePath = filepath.Join("responses", filepath.Base(responsePath))

    return Result{
        Serial:        serial,
        URL:           url,
        StatusCode:    resp.StatusCode,
        ContentType:   contentType,
        ContentLength: contentLength,
        Screenshot:    screenshot,
        ResponseBody:  responsePath,
    }
}

func captureScreenshot(url, filepath string) string {
    ctx, cancel := chromedp.NewContext(context.Background())
    defer cancel()

    var buf []byte
    err := chromedp.Run(ctx,
        chromedp.Navigate(url),
        chromedp.WaitReady("body"),
        chromedp.CaptureScreenshot(&buf),
    )
    if err != nil {
        fmt.Println("Error capturing screenshot:", err)
        return ""
    }

    err = ioutil.WriteFile(filepath, buf, 0644)
    if err != nil {
        fmt.Println("Error saving screenshot:", err)
        return ""
    }

    return filepath
}

func generateReport(results []Result, outputFile string) error {
    const tmpl = `
<!DOCTYPE html>
<html>
<head>
    <title>Bug Bounty Report</title>
    <style>
        body {
            background-color: black;
            color: green;
            font-family: "Courier New", Courier, monospace;
        }
        h1 {
            text-align: center;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
            border: 1px solid green;
        }
        th, td {
            border: 1px solid green;
            padding: 10px;
            text-align: left;
        }
        th {
            cursor: pointer;
            background-color: rgba(0, 100, 0, 0.8);
            color: lime;
        }
        a {
            color: lime;
            text-decoration: none;
        }
        a:hover {
            text-decoration: underline;
        }
        .modal {
            display: none;
            position: fixed;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            background-color: black;
            border: 2px solid lime;
            padding: 20px;
            z-index: 1000;
        }
        .overlay {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0, 0, 0, 0.8);
            z-index: 500;
        }
        input.url-checkbox {
            width: 20px;
            height: 20px;
        }
    </style>
    <script>
        function showModal(contentPath, isResponse) {
            const modal = document.getElementById('modal');
            const modalContent = document.getElementById('modalContent');
            const overlay = document.getElementById('overlay');

            if (isResponse) {
                fetch(contentPath)
                    .then(response => response.text())
                    .then(data => {
                        modalContent.innerHTML = '<div style="max-height: 80vh; max-width: 100%; overflow: auto;"><pre>' + data + '</pre></div>';
                    });
            } else {
                modalContent.innerHTML = '<img src="' + contentPath + '" alt="Screenshot" style="max-width: 100%;">';
            }

            modal.style.display = 'block';
            overlay.style.display = 'block';
        }

        function closeModal() {
            const modal = document.getElementById('modal');
            const overlay = document.getElementById('overlay');
            modal.style.display = 'none';
            overlay.style.display = 'none';
        }

        function copySelectedURLs() {
            const checkboxes = document.querySelectorAll('.url-checkbox:checked');
            const urls = Array.from(checkboxes).map(cb => cb.getAttribute('data-url'));
            const textarea = document.createElement('textarea');
            textarea.value = urls.join('\n');
            document.body.appendChild(textarea);
            textarea.select();
            document.execCommand('copy');
            document.body.removeChild(textarea);
            alert('Selected URLs copied to clipboard!');
        }

        function sortTable(n) {
            const table = document.querySelector("table");
            const rows = Array.from(table.rows).slice(1); // Exclude header row
            const ascending = table.dataset.sortOrder !== 'ascending';

            rows.sort((a, b) => {
                const textA = a.cells[n].innerText || a.cells[n].textContent;
                const textB = b.cells[n].innerText || b.cells[n].textContent;

                return ascending
                    ? textA.localeCompare(textB, undefined, { numeric: true })
                    : textB.localeCompare(textA, undefined, { numeric: true });
            });

            table.dataset.sortOrder = ascending ? 'ascending' : 'descending';

            const tableBody = table.querySelector("tbody");
            tableBody.innerHTML = "";
            rows.forEach(row => tableBody.appendChild(row));
        }
    </script>
</head>
<body>
    <h1>Bug Bounty Report</h1>
    <button onclick="copySelectedURLs()">Copy Selected URLs</button>
    <table>
        <thead>
            <tr>
                <th onclick="sortTable(0)">Select</th>
                <th onclick="sortTable(1)">Serial</th>
                <th onclick="sortTable(2)">Status Code</th>
                <th onclick="sortTable(3)">Content Type</th>
                <th onclick="sortTable(4)">Content Length</th>
                <th onclick="sortTable(5)">URL</th>
                <th onclick="sortTable(6)">Screenshot</th>
                <th onclick="sortTable(7)">Response</th>
            </tr>
        </thead>
        <tbody>
            {{range .}}
            <tr>
                <td><input type="checkbox" class="url-checkbox" data-url="{{.URL}}"></td>
                <td>{{.Serial}}</td>
                <td>{{.StatusCode}}</td>
                <td>{{.ContentType}}</td>
                <td>{{.ContentLength}}</td>
                <td><a href="{{.URL}}" target="_blank">{{.URL}}</a></td>
                <td><a href="javascript:void(0);" onclick="showModal('{{.Screenshot}}', false)">View Screenshot</a></td>
                <td><a href="{{.ResponseBody}}" target="_blank">View Response</a></td>
            </tr>
            {{end}}
        </tbody>
    </table>

    <!-- Modal -->
    <div id="overlay" class="overlay" onclick="closeModal()"></div>
    <div id="modal" class="modal">
        <button onclick="closeModal()">Close</button>
        <div id="modalContent"></div>
    </div>

    <script>

        function showModal(contentPath, isResponse) {
            const modal = document.getElementById('modal');
            const modalContent = document.getElementById('modalContent');
            const overlay = document.getElementById('overlay');

            if (isResponse) {
                fetch(contentPath)
                    .then(response => response.text())
                    .then(data => {
                        modalContent.innerHTML = '<div style="max-height: 80vh; max-width: 100%; overflow: auto;"><pre>' + data + '</pre></div>';
                    })
                    .catch(error => {
                        modalContent.innerHTML = '<p style="color: red;">Error loading response content. Please try again.</p>';
                        console.error('Error fetching response content:', error);
                        });
                    } else {
                        modalContent.innerHTML = '<img src="' + contentPath + '" alt="Screenshot" style="max-width: 100%;">';
                    }

                    modal.style.display = 'block';
                    overlay.style.display = 'block';
                }

        function closeModal() {
            const modal = document.getElementById('modal');
            const overlay = document.getElementById('overlay');
            modal.style.display = 'none';
            overlay.style.display = 'none';
        }

        function copySelectedURLs() {
            const checkboxes = document.querySelectorAll('.url-checkbox:checked');
            const urls = Array.from(checkboxes).map(cb => cb.getAttribute('data-url'));
            const textarea = document.createElement('textarea');
            textarea.value = urls.join('\n');
            document.body.appendChild(textarea);
            textarea.select();
            document.execCommand('copy');
            document.body.removeChild(textarea);
            alert('Selected URLs copied to clipboard!');
        }

        function sortTable(n) {
            const table = document.querySelector("table");
            const rows = Array.from(table.rows).slice(1); // Exclude header row
            const ascending = table.dataset.sortOrder !== 'ascending';

            rows.sort((a, b) => {
                const textA = a.cells[n].innerText || a.cells[n].textContent;
                const textB = b.cells[n].innerText || b.cells[n].textContent;

                return ascending
                    ? textA.localeCompare(textB, undefined, { numeric: true })
                    : textB.localeCompare(textA, undefined, { numeric: true });
            });

            table.dataset.sortOrder = ascending ? 'ascending' : 'descending';

            const tableBody = table.querySelector("tbody");
            tableBody.innerHTML = "";
            rows.forEach(row => tableBody.appendChild(row));
        }
    </script>
</body>
</html>`

    report, err := template.New("report").Parse(tmpl)
    if err != nil {
        return err
    }

    f, err := os.Create(outputFile)
    if err != nil {
        return err
    }
    defer f.Close()

    return report.Execute(f, results)
}

