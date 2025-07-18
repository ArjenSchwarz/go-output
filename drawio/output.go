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
func CreateCSV(drawIOHeader Header, headerRow []string, contents []map[string]string, filename string) error {
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
			return fmt.Errorf("failed to create file %s: %w", filename, err)
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
	_, err := buf.WriteTo(target)
	if err != nil {
		return fmt.Errorf("failed to write header to output: %w", err)
	}
	w := csv.NewWriter(target)

	for _, record := range total {
		if err := w.Write(record); err != nil {
			return fmt.Errorf("error writing record to csv: %w", err)
		}
	}

	w.Flush()

	if err := w.Error(); err != nil {
		return fmt.Errorf("error flushing csv writer: %w", err)
	}

	return nil
}

// GetHeaderAndContentsFromFile returns the headers of a CSV in a reverse map (name:column-id) and the remaining rows
// It filters away all of the comments
func GetHeaderAndContentsFromFile(filename string) (map[string]int, [][]string, error) {
	headerrow, contents, err := getContentsFromFile(filename)
	if err != nil {
		return nil, nil, err
	}
	headers := make(map[string]int)
	for index, name := range headerrow {
		headers[name] = index
	}
	return headers, contents, nil
}

// GetContentsFromFileAsStringMaps returns the CSV contents as a slice of string maps
func GetContentsFromFileAsStringMaps(filename string) ([]map[string]string, error) {
	header, contents, err := getContentsFromFile(filename)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]string, 0, len(contents))
	for _, row := range contents {
		resultrow := make(map[string]string)
		for index, value := range row {
			if index < len(header) {
				resultrow[header[index]] = value
			}
		}
		result = append(result, resultrow)
	}
	return result, nil
}

// getContentsFromFile returns the headers of a CSV in a string slice and the remaining rows separately
// It filters away all of the comments
func getContentsFromFile(filename string) ([]string, [][]string, error) {
	originalfile, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}
	originalString := string(originalfile)
	r := csv.NewReader(strings.NewReader(originalString))
	r.Comment = '#'
	records, err := r.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CSV from file %s: %w", filename, err)
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("CSV file %s is empty", filename)
	}
	// Return headers separate from records
	return records[0], records[1:], nil
}
