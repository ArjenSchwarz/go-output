package format

import (
	"reflect"
	"testing"
)

func defaultContentsForTests() []OutputHolder {
	contents := make([]OutputHolder, 0)

	value1 := OutputHolder{
		Contents: map[string]interface{}{
			"Export":      "awesome-stack-dev-s3-arn",
			"Value":       "arn:aws:s3:::fog-awesome-stack-dev",
			"Description": "ARN of the S3 bucket",
			"Stack":       "awesome-stack-dev",
			"Imported":    true,
			"Imported By": "demo-resources",
		},
	}
	value4 := OutputHolder{
		Contents: map[string]interface{}{
			"Export":      "demo-s3-bucket",
			"Value":       "fog-demo-bucket",
			"Description": "The S3 bucket used for demos but has an exceptionally long description so it can show a multi-line example",
			"Stack":       "demo-resources",
			"Imported":    false,
			"Imported By": "",
		},
	}

	contents = append(contents, value1)
	contents = append(contents, value4)
	return contents
}

// TestOutputArray_toMermaid only needs to test basic functionality,
// the rest is handled in the tests for the Mermaid class
func TestOutputArray_toMermaid(t *testing.T) {
	type fields struct {
		Settings *OutputSettings
		Contents []OutputHolder
		Keys     []string
	}
	keys := []string{"Export", "Description", "Stack", "Value", "Imported"}
	title := "Export values demo"
	output := fields{Keys: keys, Settings: NewOutputSettings()}
	output.Settings.Title = title
	output.Settings.SortKey = "Export"
	output.Settings.AddFromToColumns("Stack", "Imported By")
	output.Contents = defaultContentsForTests()

	result := []byte(`flowchart TB
	n1("awesome-stack-dev")
	n2("demo-resources")
	n1 --> n2`)

	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{"test", output, result},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := OutputArray{
				Settings: tt.fields.Settings,
				Contents: tt.fields.Contents,
				Keys:     tt.fields.Keys,
			}
			if got := output.toMermaid(); string(got) != string(tt.want) {
				t.Errorf("OutputArray.toMermaid() = \r\n%v, want \r\n%v", string(got), string(tt.want))
			}
		})
	}
}

func TestOutputArray_toString(t *testing.T) {
	type fields struct {
		Settings *OutputSettings
		Contents []OutputHolder
		Keys     []string
	}
	withEmoji := OutputSettings{
		UseEmoji: true,
	}
	noEmoji := OutputSettings{
		UseEmoji: false,
	}
	jsonFormat := OutputSettings{
		OutputFormat: "json",
	}
	tableFormat := OutputSettings{
		OutputFormat: "table",
	}
	type args struct {
		val interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{"Plain String", fields{Settings: &noEmoji}, args{val: "Plain String"}, "Plain String"},
		{"Integer 0", fields{Settings: &noEmoji}, args{val: 0}, "0"},
		{"Integer 1337", fields{Settings: &noEmoji}, args{val: 1337}, "1337"},
		{"NoEmoji Bool true", fields{Settings: &noEmoji}, args{val: true}, "Yes"},
		{"NoEmoji Bool false", fields{Settings: &noEmoji}, args{val: false}, "No"},
		{"Emoji Bool true", fields{Settings: &withEmoji}, args{val: true}, "✅"},
		{"Emoji Bool false", fields{Settings: &withEmoji}, args{val: false}, "❌"},
		{"Slice json format", fields{Settings: &jsonFormat}, args{val: []string{"first", "second"}}, "first, second"},
		{"Slice table format", fields{Settings: &tableFormat}, args{val: []string{"first", "second"}}, "first\nsecond"},
		{"Float64", fields{Settings: &noEmoji}, args{val: 123.456}, "123.456"},
		{"Float32", fields{Settings: &noEmoji}, args{val: float32(78.9)}, "78.9"},
		{"Int64", fields{Settings: &noEmoji}, args{val: int64(9223372036854775807)}, "9223372036854775807"},
		{"Uint", fields{Settings: &noEmoji}, args{val: uint(42)}, "42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &OutputArray{
				Settings: tt.fields.Settings,
				Contents: tt.fields.Contents,
				Keys:     tt.fields.Keys,
			}
			if got := output.toString(tt.args.val); got != tt.want {
				t.Errorf("OutputArray.toString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputArray_splitFromToValues(t *testing.T) {
	type fields struct {
		Settings *OutputSettings
		Contents []OutputHolder
		Keys     []string
	}
	keys := []string{"Export", "Description", "Stack", "Value", "Imported"}
	title := "Export values demo"
	manytoone := fields{Keys: keys, Settings: NewOutputSettings()}
	manytoone.Settings.Title = title
	manytoone.Settings.SortKey = "Export"
	manytoone.Settings.AddFromToColumns("Stack", "Imported By")

	value1 := OutputHolder{
		Contents: map[string]interface{}{
			"Stack":       "awesome-stack-dev",
			"Imported By": "demo-resources",
		},
	}
	value2 := OutputHolder{
		Contents: map[string]interface{}{
			"Stack":       "awesome-stack-test",
			"Imported By": "demo-resources",
		},
	}
	value3 := OutputHolder{
		Contents: map[string]interface{}{
			"Stack":       "awesome-stack-prod",
			"Imported By": "demo-resources",
		},
	}
	value4 := OutputHolder{
		Contents: map[string]interface{}{
			"Stack":       "demo-resources",
			"Imported By": "",
		},
	}
	value5 := OutputHolder{
		Contents: map[string]interface{}{
			"Stack":       "awesome-stack-prod",
			"Imported By": "demo-resources,awesome-stack-test",
		},
	}

	manytoonecontents := make([]OutputHolder, 0)
	manytoonecontents = append(manytoonecontents, value1)
	manytoonecontents = append(manytoonecontents, value2)
	manytoonecontents = append(manytoonecontents, value3)
	manytoonecontents = append(manytoonecontents, value4)
	manytoone.Contents = manytoonecontents

	onetomany := fields{Keys: keys, Settings: NewOutputSettings()}
	onetomany.Settings.Title = title
	onetomany.Settings.SortKey = "Export"
	onetomany.Settings.AddFromToColumns("Stack", "Imported By")
	onetomanycontents := make([]OutputHolder, 0)
	onetomanycontents = append(onetomanycontents, value2)
	onetomanycontents = append(onetomanycontents, value5)
	onetomany.Contents = onetomanycontents

	manytomany := fields{Keys: keys, Settings: NewOutputSettings()}
	manytomany.Settings.Title = title
	manytomany.Settings.SortKey = "Export"
	manytomany.Settings.AddFromToColumns("Stack", "Imported By")
	manytomanycontents := make([]OutputHolder, 0)
	manytomanycontents = append(manytomanycontents, value1)
	manytomanycontents = append(manytomanycontents, value2)
	manytomanycontents = append(manytomanycontents, value5)
	manytomanycontents = append(manytomanycontents, value4)
	manytomany.Contents = manytomanycontents

	manytooneresult := []fromToValues{
		{
			From: "awesome-stack-dev",
			To:   "demo-resources",
		},
		{
			From: "awesome-stack-test",
			To:   "demo-resources",
		},
		{
			From: "awesome-stack-prod",
			To:   "demo-resources",
		},
		{
			From: "demo-resources",
			To:   "",
		},
	}
	onetomanyresult := []fromToValues{
		{
			From: "awesome-stack-test",
			To:   "demo-resources",
		},
		{
			From: "awesome-stack-prod",
			To:   "demo-resources",
		},
		{
			From: "awesome-stack-prod",
			To:   "awesome-stack-test",
		},
	}
	manytomanyresult := []fromToValues{
		{
			From: "awesome-stack-dev",
			To:   "demo-resources",
		},
		{
			From: "awesome-stack-test",
			To:   "demo-resources",
		},
		{
			From: "awesome-stack-prod",
			To:   "demo-resources",
		},
		{
			From: "awesome-stack-prod",
			To:   "awesome-stack-test",
		},
		{
			From: "demo-resources",
			To:   "",
		},
	}

	tests := []struct {
		name   string
		fields fields
		want   []fromToValues
	}{
		{"Many to One", manytoone, manytooneresult},
		{"One to Many", onetomany, onetomanyresult},
		{"Many to Many", manytomany, manytomanyresult},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := OutputArray{
				Settings: tt.fields.Settings,
				Contents: tt.fields.Contents,
				Keys:     tt.fields.Keys,
			}
			if got := output.splitFromToValues(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OutputArray.splitFromToValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputArray_GetContentsMap(t *testing.T) {
	type fields struct {
		Settings *OutputSettings
		Contents []OutputHolder
		Keys     []string
	}

	keys := []string{"Name", "Value"}
	output := fields{Keys: keys, Settings: NewOutputSettings()}
	value1 := OutputHolder{
		Contents: map[string]interface{}{
			"Name":  "Test",
			"Value": "Something",
		},
	}
	output.Contents = append(output.Contents, value1)
	expectedOutput := make([]map[string]string, 0, len(output.Contents))
	expectedOutput = append(expectedOutput, map[string]string{
		"Name":  "Test",
		"Value": "Something",
	})

	output2 := fields{Keys: keys, Settings: NewOutputSettings()}
	value2 := OutputHolder{
		Contents: map[string]interface{}{
			"Name":  "Item 2",
			"Value": "Something else",
		},
	}
	output2.Contents = append(output2.Contents, value1)
	output2.Contents = append(output2.Contents, value2)
	expectedOutput2 := make([]map[string]string, 0)
	expectedOutput2 = append(expectedOutput2, map[string]string{
		"Name":  "Test",
		"Value": "Something",
	})
	expectedOutput2 = append(expectedOutput2, map[string]string{
		"Name":  "Item 2",
		"Value": "Something else",
	})

	outputnonString := fields{Keys: keys, Settings: NewOutputSettings()}
	value3 := OutputHolder{
		Contents: map[string]interface{}{
			"Name":  true,
			"Value": 42,
		},
	}
	outputnonString.Contents = append(outputnonString.Contents, value3)
	expectedOutputnonString := make([]map[string]string, 0)
	expectedOutputnonString = append(expectedOutputnonString, map[string]string{
		"Name":  "Yes",
		"Value": "42",
	})

	keys3 := []string{"Export", "Description", "Stack", "Value", "Imported", "Imported By"}
	output3 := fields{Keys: keys3, Settings: NewOutputSettings()}
	output3.Contents = defaultContentsForTests()
	expectedOutput3 := make([]map[string]string, 0, len(output3.Contents))
	expectedOutput3 = append(expectedOutput3, map[string]string{
		"Export":      "awesome-stack-dev-s3-arn",
		"Value":       "arn:aws:s3:::fog-awesome-stack-dev",
		"Description": "ARN of the S3 bucket",
		"Stack":       "awesome-stack-dev",
		"Imported":    "Yes",
		"Imported By": "demo-resources",
	})
	expectedOutput3 = append(expectedOutput3, map[string]string{
		"Export":      "demo-s3-bucket",
		"Value":       "fog-demo-bucket",
		"Description": "The S3 bucket used for demos but has an exceptionally long description so it can show a multi-line example",
		"Stack":       "demo-resources",
		"Imported":    "No",
		"Imported By": "",
	})

	tests := []struct {
		name   string
		fields fields
		want   []map[string]string
	}{
		{"Two fields One Object", output, expectedOutput},
		{"Two fields Two Objects", output2, expectedOutput2},
		{"Two fields One Object NonStrings - tests conversion", outputnonString, expectedOutputnonString},
		{"Many Fields Two Objects", output3, expectedOutput3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := OutputArray{
				Settings: tt.fields.Settings,
				Contents: tt.fields.Contents,
				Keys:     tt.fields.Keys,
			}
			if got := output.GetContentsMap(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OutputArray.GetContentsMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputArray_GetContentsMapRaw(t *testing.T) {
	type fields struct {
		Settings *OutputSettings
		Contents []OutputHolder
		Keys     []string
	}
	keys := []string{"Name", "Value"}
	output := fields{Keys: keys, Settings: NewOutputSettings()}
	value1 := OutputHolder{
		Contents: map[string]interface{}{
			"Name":  "Test",
			"Value": "Something",
		},
	}
	output.Contents = append(output.Contents, value1)
	expectedOutput := make([]map[string]interface{}, 0, len(output.Contents))
	expectedOutput = append(expectedOutput, map[string]interface{}{
		"Name":  "Test",
		"Value": "Something",
	})

	output2 := fields{Keys: keys, Settings: NewOutputSettings()}
	value2 := OutputHolder{
		Contents: map[string]interface{}{
			"Name":  "Item 2",
			"Value": "Something else",
		},
	}
	output2.Contents = append(output2.Contents, value1)
	output2.Contents = append(output2.Contents, value2)
	expectedOutput2 := make([]map[string]interface{}, 0)
	expectedOutput2 = append(expectedOutput2, map[string]interface{}{
		"Name":  "Test",
		"Value": "Something",
	})
	expectedOutput2 = append(expectedOutput2, map[string]interface{}{
		"Name":  "Item 2",
		"Value": "Something else",
	})

	outputnonString := fields{Keys: keys, Settings: NewOutputSettings()}
	value3 := OutputHolder{
		Contents: map[string]interface{}{
			"Name":  true,
			"Value": 42,
		},
	}
	outputnonString.Contents = append(outputnonString.Contents, value3)
	expectedOutputnonString := make([]map[string]interface{}, 0)
	expectedOutputnonString = append(expectedOutputnonString, map[string]interface{}{
		"Name":  true,
		"Value": 42,
	})

	keys3 := []string{"Export", "Description", "Stack", "Value", "Imported", "Imported By"}
	output3 := fields{Keys: keys3, Settings: NewOutputSettings()}
	output3.Contents = defaultContentsForTests()
	expectedOutput3 := make([]map[string]interface{}, 0, len(output3.Contents))
	expectedOutput3 = append(expectedOutput3, map[string]interface{}{
		"Export":      "awesome-stack-dev-s3-arn",
		"Value":       "arn:aws:s3:::fog-awesome-stack-dev",
		"Description": "ARN of the S3 bucket",
		"Stack":       "awesome-stack-dev",
		"Imported":    true,
		"Imported By": "demo-resources",
	})
	expectedOutput3 = append(expectedOutput3, map[string]interface{}{
		"Export":      "demo-s3-bucket",
		"Value":       "fog-demo-bucket",
		"Description": "The S3 bucket used for demos but has an exceptionally long description so it can show a multi-line example",
		"Stack":       "demo-resources",
		"Imported":    false,
		"Imported By": "",
	})

	tests := []struct {
		name   string
		fields fields
		want   []map[string]interface{}
	}{
		{"Two fields One Object", output, expectedOutput},
		{"Two fields Two Objects", output2, expectedOutput2},
		{"Two fields One Object NonStrings - tests no conversion", outputnonString, expectedOutputnonString},
		{"Many Fields Two Objects", output3, expectedOutput3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := OutputArray{
				Settings: tt.fields.Settings,
				Contents: tt.fields.Contents,
				Keys:     tt.fields.Keys,
			}
			if got := output.GetContentsMapRaw(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OutputArray.GetContentsMapRaw() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestOutputArray_toJSON only needs to test the json output part, the underlying logic is tested in other functions already
func TestOutputArray_toJSON(t *testing.T) {
	type fields struct {
		Settings *OutputSettings
		Contents []OutputHolder
		Keys     []string
	}
	keys := []string{"Name", "Value"}
	output := fields{Keys: keys, Settings: NewOutputSettings()}
	value1 := OutputHolder{
		Contents: map[string]interface{}{
			"Name":  "Test",
			"Value": "Something",
		},
	}
	output.Contents = append(output.Contents, value1)
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{"Basic value", output, []byte("[{\"Name\":\"Test\",\"Value\":\"Something\"}]")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := OutputArray{
				Settings: tt.fields.Settings,
				Contents: tt.fields.Contents,
				Keys:     tt.fields.Keys,
			}
			if got := output.toJSON(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OutputArray.toJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
