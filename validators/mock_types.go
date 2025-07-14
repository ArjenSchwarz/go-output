package validators

import "github.com/aws/aws-sdk-go-v2/service/s3"

// Mock types for validator testing
type mockOutputSettings struct {
	OutputFormat    string
	OutputFile      string
	S3Bucket        mockS3Output
	FromToColumns   *mockFromToColumns
	MermaidSettings *mockMermaidSettings
}

type mockS3Output struct {
	S3Client *s3.Client
	Bucket   string
	Path     string
}

type mockFromToColumns struct {
	From  string
	To    string
	Label string
}

func (m *mockFromToColumns) GetFrom() string  { return m.From }
func (m *mockFromToColumns) GetTo() string    { return m.To }
func (m *mockFromToColumns) GetLabel() string { return m.Label }

type mockMermaidSettings struct {
	ChartType string
}

func (m *mockMermaidSettings) GetChartType() string { return m.ChartType }

// Interface implementations for mocks
func (m *mockOutputSettings) GetOutputFormat() string {
	return m.OutputFormat
}

func (m *mockOutputSettings) GetOutputFile() string {
	return m.OutputFile
}

func (m *mockOutputSettings) GetS3Bucket() S3Output {
	return &m.S3Bucket
}

func (m *mockOutputSettings) GetFromToColumns() *FromToColumns {
	return nil // Simplify for now
}

func (m *mockOutputSettings) GetMermaidSettings() *MermaidSettings {
	return nil // Simplify for now
}

func (m *mockS3Output) GetS3Client() *s3.Client {
	return m.S3Client
}

func (m *mockS3Output) GetBucket() string {
	return m.Bucket
}

func (m *mockS3Output) GetPath() string {
	return m.Path
}
