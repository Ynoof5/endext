package main

import (
        "bufio"
        "flag"
        "fmt"
        "io/ioutil"
        "net/http"
        "os"
        "regexp"
        "strings"
        "time"
)

var numericExtedEPs = 1 // Counter var for extracted endpoints

// getFileContent fetches the JS file content if it is accessible with a status code of 200.
func getFileContent(url string) ([]byte, error) {
        client := &http.Client{
                Timeout: 7 * time.Second,
        }
        req, err := http.NewRequest("GET", url, nil)
        if err != nil {
                return nil, err
        }
        req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:125.0) Gecko/20100101 Firefox/125.0")

        resp, err := client.Do(req)
        if err != nil {
                return nil, err
        }
        defer resp.Body.Close()

        if resp.StatusCode != 200 {
                return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
        }

        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return nil, err
        }
        return body, nil
}

// readRegexPatterns reads regex patterns from a file and returns them.
func readRegexPatterns(filename string) ([]*regexp.Regexp, error) {
        file, err := os.Open(filename)
        if err != nil {
                return nil, err
        }
        defer file.Close()

        var patterns []*regexp.Regexp
        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
                pattern, err := regexp.Compile(scanner.Text())
                if err != nil {
                        return nil, err
                }
                patterns = append(patterns, pattern)
        }
        if err := scanner.Err(); err != nil {
                return nil, err
        }
        return patterns, nil
}

// applyRegexes applies a list of regexes to the content and returns all matches.
func applyRegexes(content []byte, regexes []*regexp.Regexp) []string {
        var matches []string
        for _, regex := range regexes {
                matches = append(matches, regex.FindAllString(string(content), -1)...)
        }
        return matches
}

func isValidMatch(match string) bool {
        unwantedStrings := []string{"\"/$\"", "\"/*\"", "\"?\"", "\"/\"", "\"//\"", "`/`", "==="}
        for _, u := range unwantedStrings {
                if match == u || strings.Contains(match, "===") {
                        return false
                }
        }

        return !strings.ContainsAny(match, ":;{},()|[]!<>^*+ ")
}

func appendTextToFile(filename, content string) error {
        file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
        if err != nil {
                return fmt.Errorf("error opening file: %w", err)
        }
        defer file.Close()

        if _, err := file.WriteString(content + "\n"); err != nil {
                return fmt.Errorf("error writing to file: %w", err)
        }

        return nil
}

func Extract(url, outputFile string, isSilent bool) {
        content, err := getFileContent(url)
        if err != nil {
                return // Ignore inaccessible URLs
        }

        regexes, err := readRegexPatterns("regex.tmp")
        if err != nil {
                fmt.Println("[ ! ] Failed to read regex patterns : ", err)
                return
        }

        matches := applyRegexes(content, regexes)
        uniqueEndpoints := make(map[string]struct{}) // Set to store unique endpoints
        for _, match := range matches {
                if isValidMatch(match) {
                        uniqueEndpoints[match] = struct{}{}
                }
        }

        // Print the URL and its extracted unique endpoints
        if !isSilent {
                fmt.Printf("[ %v ] %v\n", numericExtedEPs, url)
                for endpoint := range uniqueEndpoints {
                        fmt.Println(endpoint)
                }
        }
        if outputFile != "" {
                err := appendTextToFile(outputFile, fmt.Sprintf("[ %v ] %v", numericExtedEPs, url))
                if err != nil {
                        fmt.Println("Error writing to file:", err)
                        return
                }
                for endpoint := range uniqueEndpoints {
                        err := appendTextToFile(outputFile, endpoint)
                        if err != nil {
                                fmt.Println("Error writing to file:", err)
                                return
                        }
                }
        }
        numericExtedEPs++
}


func main() {
        flagJsFile := flag.String("l", "", ".txt File containing JavaScript file URLs")
        flagSingleJsFile := flag.String("u", "", "Single JavaScript File Direct URL")
        flagOutputFile := flag.String("o", "", "Output To Save Endpoints")
        flagSilent := flag.Bool("s", false, "Silence Bitch")
        flag.Parse()

        if *flagJsFile == "" && *flagSingleJsFile == "" || *flagJsFile != "" && *flagSingleJsFile != "" {
                fmt.Println("Please use one of -u for single js file URL or -l for .txt file contains js files URLs.")
                return
        }

        startTime := time.Now() // time measuring start var

        if *flagJsFile != "" {
                jsfile, err := os.Open(*flagJsFile)
                if err != nil {
                        fmt.Println("Error opening file:", err)
                        return
                }
                defer jsfile.Close()

                scanner := bufio.NewScanner(jsfile)
                for scanner.Scan() {
                        go Extract(scanner.Text(), *flagOutputFile, *flagSilent)
                }
                if err := scanner.Err(); err != nil {
                        fmt.Println("Error scanning file:", err)
                        return
                }
        }
        if *flagSingleJsFile != "" {
                Extract(*flagSingleJsFile, *flagOutputFile, *flagSilent)
        }

        time.Sleep(7 * time.Second)
        elapsedTime := time.Since(startTime) - (7 * time.Second)
        fmt.Printf("Process took %d ms\n", elapsedTime.Milliseconds())
}
