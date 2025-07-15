package drawio

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// CreateCSV creates the CSV complete with the header
func CreateCSV(drawIOHeader Header, headerRow []string, contents []map[string]string, filename string) {
	total := [][]string{}
	total = append(total, headerRow)
	for _, holder := range contents {
		values := make([]string, len(headerRow))
		for counter, key := range headerRow {
			if val, ok := holder[key]; ok {
				values[counter] = val
			}
		}
		total = append(total, values)
	}
	var target io.Writer
	if filename == "" {
		target = os.Stdout
	} else {
		file, err := os.Create(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			if cerr := file.Close(); cerr != nil {
				log.Println(cerr)
			}
		}()
		target = bufio.NewWriter(file)
	}
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s", drawIOHeader.String())
	if _, err := buf.WriteTo(target); err != nil {
		log.Println(err)
		return
	}
	w := csv.NewWriter(target)

	for _, record := range total {
		if err := w.Write(record); err != nil {
			log.Println("error writing record to csv:", err)
			return
		}
	}

	w.Flush()

	if err := w.Error(); err != nil {
		log.Println(err)
		return
	}
}

// GetHeaderAndContentsFromFile returns the headers of a CSV in a reverse map (name:column-id) and the remaining rows
// It filters away all of the comments
func GetHeaderAndContentsFromFile(filename string) (map[string]int, [][]string) {
	headerrow, contents := getContentsFromFile(filename)
	headers := make(map[string]int)
	for index, name := range headerrow {
		headers[name] = index
	}
	return headers, contents
}

// GetContentsFromFileAsStringMaps returns the CSV contents as a slice of string maps
func GetContentsFromFileAsStringMaps(filename string) []map[string]string {
	header, contents := getContentsFromFile(filename)
	result := make([]map[string]string, 0, len(contents))
	for _, row := range contents {
		resultrow := make(map[string]string)
		for index, value := range row {
			resultrow[header[index]] = value
		}
		result = append(result, resultrow)
	}
	return result
}

// getContentsFromFile returns the headers of a CSV in a string slice and the remaining rows separately
// It filters away all of the comments
func getContentsFromFile(filename string) ([]string, [][]string) {
	originalfile, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	originalString := string(originalfile)
	r := csv.NewReader(strings.NewReader(originalString))
	r.Comment = '#'
	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	// Return headers separate from records
	return records[0], records[1:]
}
