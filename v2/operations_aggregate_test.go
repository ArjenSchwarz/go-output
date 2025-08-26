package output

import (
	"context"
	"strings"
	"testing"
)

func TestAggregateOpValidation(t *testing.T) {
	t.Run("validates GroupBy with single column", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department"},
			aggregates: map[string]AggregateFunc{
				"count": CountAggregate(),
			},
		}

		err := groupByOp.Validate()
		if err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})

	t.Run("validates GroupBy with multiple columns", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department", "level"},
			aggregates: map[string]AggregateFunc{
				"total_salary": SumAggregate("salary"),
				"count":        CountAggregate(),
			},
		}

		err := groupByOp.Validate()
		if err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})

	t.Run("fails validation when no groupBy columns provided", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{},
			aggregates: map[string]AggregateFunc{
				"count": CountAggregate(),
			},
		}

		err := groupByOp.Validate()
		if err == nil {
			t.Fatal("expected validation error for empty groupBy columns")
		}

		if !strings.Contains(err.Error(), "groupBy operation requires at least one grouping column") {
			t.Errorf("expected error about groupBy requiring at least one column, got '%s'", err.Error())
		}
	})

	t.Run("fails validation when no aggregates provided", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy:    []string{"department"},
			aggregates: map[string]AggregateFunc{},
		}

		err := groupByOp.Validate()
		if err == nil {
			t.Fatal("expected validation error for empty aggregates")
		}

		if !strings.Contains(err.Error(), "groupBy operation requires at least one aggregate function") {
			t.Errorf("expected error about groupBy requiring at least one aggregate function, got '%s'", err.Error())
		}
	})
}

// TestAggregateOpApply tests GroupByOp Apply method functionality
func TestAggregateOpApply(t *testing.T) {
	t.Run("groups by single column with count aggregate", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department"},
			aggregates: map[string]AggregateFunc{
				"count": CountAggregate(),
			},
		}

		doc := New().
			Table("test", []Record{
				{"name": "Alice", "department": "HR", "salary": 50000},
				{"name": "Bob", "department": "IT", "salary": 75000},
				{"name": "Charlie", "department": "HR", "salary": 60000},
				{"name": "David", "department": "IT", "salary": 80000},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := groupByOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 2 {
			t.Errorf("expected 2 groups, got %d", len(resultTable.records))
		}

		// Verify groups and counts
		groupCounts := make(map[string]int)
		for _, record := range resultTable.records {
			department := record["department"].(string)
			count := record["count"].(int)
			groupCounts[department] = count
		}

		if groupCounts["HR"] != 2 {
			t.Errorf("expected HR count 2, got %d", groupCounts["HR"])
		}
		if groupCounts["IT"] != 2 {
			t.Errorf("expected IT count 2, got %d", groupCounts["IT"])
		}
	})

	t.Run("groups by single column with sum aggregate", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department"},
			aggregates: map[string]AggregateFunc{
				"total_salary": SumAggregate("salary"),
			},
		}

		doc := New().
			Table("test", []Record{
				{"name": "Alice", "department": "HR", "salary": 50000},
				{"name": "Bob", "department": "IT", "salary": 75000},
				{"name": "Charlie", "department": "HR", "salary": 60000},
				{"name": "David", "department": "IT", "salary": 80000},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := groupByOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 2 {
			t.Errorf("expected 2 groups, got %d", len(resultTable.records))
		}

		// Verify groups and sums
		groupSums := make(map[string]float64)
		for _, record := range resultTable.records {
			department := record["department"].(string)
			totalSalary := record["total_salary"].(float64)
			groupSums[department] = totalSalary
		}

		if groupSums["HR"] != 110000 {
			t.Errorf("expected HR total salary 110000, got %f", groupSums["HR"])
		}
		if groupSums["IT"] != 155000 {
			t.Errorf("expected IT total salary 155000, got %f", groupSums["IT"])
		}
	})

	t.Run("groups by multiple columns", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department", "level"},
			aggregates: map[string]AggregateFunc{
				"avg_salary": AverageAggregate("salary"),
				"count":      CountAggregate(),
			},
		}

		doc := New().
			Table("test", []Record{
				{"name": "Alice", "department": "HR", "level": "Junior", "salary": 50000},
				{"name": "Bob", "department": "IT", "level": "Senior", "salary": 80000},
				{"name": "Charlie", "department": "HR", "level": "Senior", "salary": 70000},
				{"name": "David", "department": "IT", "level": "Junior", "salary": 60000},
				{"name": "Eve", "department": "HR", "level": "Junior", "salary": 45000},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := groupByOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 4 {
			t.Errorf("expected 4 groups, got %d", len(resultTable.records))
		}

		// Verify we have the expected department/level combinations
		groups := make(map[string]map[string]bool)
		for _, record := range resultTable.records {
			department := record["department"].(string)
			level := record["level"].(string)
			count := record["count"].(int)

			if groups[department] == nil {
				groups[department] = make(map[string]bool)
			}
			groups[department][level] = true

			// HR Junior should have 2 people (avg 47500)
			if department == "HR" && level == "Junior" && count != 2 {
				t.Errorf("expected HR Junior count 2, got %d", count)
			}
			// IT Senior should have 1 person (avg 80000)
			if department == "IT" && level == "Senior" && count != 1 {
				t.Errorf("expected IT Senior count 1, got %d", count)
			}
		}
	})

	t.Run("supports multiple aggregate functions", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department"},
			aggregates: map[string]AggregateFunc{
				"count":      CountAggregate(),
				"total":      SumAggregate("salary"),
				"average":    AverageAggregate("salary"),
				"min_salary": MinAggregate("salary"),
				"max_salary": MaxAggregate("salary"),
			},
		}

		doc := New().
			Table("test", []Record{
				{"name": "Alice", "department": "HR", "salary": 50000},
				{"name": "Bob", "department": "HR", "salary": 60000},
				{"name": "Charlie", "department": "HR", "salary": 70000},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := groupByOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 1 {
			t.Errorf("expected 1 group, got %d", len(resultTable.records))
		}

		record := resultTable.records[0]
		if record["count"] != 3 {
			t.Errorf("expected count 3, got %v", record["count"])
		}
		if record["total"] != float64(180000) {
			t.Errorf("expected total 180000, got %v", record["total"])
		}
		if record["average"] != float64(60000) {
			t.Errorf("expected average 60000, got %v", record["average"])
		}
		if record["min_salary"] != float64(50000) {
			t.Errorf("expected min_salary 50000, got %v", record["min_salary"])
		}
		if record["max_salary"] != float64(70000) {
			t.Errorf("expected max_salary 70000, got %v", record["max_salary"])
		}
	})

	t.Run("handles different numeric types correctly", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"category"},
			aggregates: map[string]AggregateFunc{
				"sum_int":   SumAggregate("value_int"),
				"sum_float": SumAggregate("value_float"),
				"sum_int64": SumAggregate("value_int64"),
			},
		}

		doc := New().
			Table("test", []Record{
				{"category": "A", "value_int": 10, "value_float": 10.5, "value_int64": int64(100)},
				{"category": "A", "value_int": 20, "value_float": 20.5, "value_int64": int64(200)},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := groupByOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		record := resultTable.records[0]

		if record["sum_int"] != float64(30) {
			t.Errorf("expected sum_int 30, got %v", record["sum_int"])
		}
		if record["sum_float"] != float64(31) {
			t.Errorf("expected sum_float 31, got %v", record["sum_float"])
		}
		if record["sum_int64"] != float64(300) {
			t.Errorf("expected sum_int64 300, got %v", record["sum_int64"])
		}
	})

	t.Run("handles custom aggregate functions", func(t *testing.T) {
		// Custom aggregate that concatenates strings
		concatAggregate := func(records []Record, field string) any {
			var result []string
			for _, record := range records {
				if val, ok := record[field].(string); ok {
					result = append(result, val)
				}
			}
			if len(result) == 0 {
				return ""
			}
			// Simple string join
			joined := result[0]
			for i := 1; i < len(result); i++ {
				joined += "," + result[i]
			}
			return joined
		}

		groupByOp := &GroupByOp{
			groupBy: []string{"department"},
			aggregates: map[string]AggregateFunc{
				"names": concatAggregate,
			},
		}

		doc := New().
			Table("test", []Record{
				{"name": "Alice", "department": "HR"},
				{"name": "Bob", "department": "HR"},
				{"name": "Charlie", "department": "IT"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := groupByOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 2 {
			t.Errorf("expected 2 groups, got %d", len(resultTable.records))
		}

		// Verify custom aggregation worked
		namesByDept := make(map[string]string)
		for _, record := range resultTable.records {
			department := record["department"].(string)
			names := record["names"].(string)
			namesByDept[department] = names
		}

		// Note: order may vary, but we should have both names for HR
		hrNames := namesByDept["HR"]
		if !(hrNames == "Alice,Bob" || hrNames == "Bob,Alice") {
			t.Errorf("expected HR names to contain Alice and Bob, got '%s'", hrNames)
		}
		if namesByDept["IT"] != "Charlie" {
			t.Errorf("expected IT names 'Charlie', got '%s'", namesByDept["IT"])
		}
	})

	t.Run("preserves key order in generated schema", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department"},
			aggregates: map[string]AggregateFunc{
				"count":        CountAggregate(),
				"total_salary": SumAggregate("salary"),
			},
		}

		doc := New().
			Table("test", []Record{
				{"name": "Alice", "department": "HR", "salary": 50000},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := groupByOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if resultTable.schema == nil {
			t.Fatal("expected schema to be generated")
		}

		keyOrder := resultTable.schema.GetKeyOrder()
		// Should start with groupBy columns, followed by aggregate columns
		expectedStart := []string{"department"}
		for i, expected := range expectedStart {
			if i >= len(keyOrder) || keyOrder[i] != expected {
				t.Errorf("expected key order to start with %v, got %v", expectedStart, keyOrder)
				break
			}
		}

		// Should contain all aggregate columns
		hasCount := false
		hasTotalSalary := false
		for _, key := range keyOrder {
			if key == "count" {
				hasCount = true
			}
			if key == "total_salary" {
				hasTotalSalary = true
			}
		}
		if !hasCount {
			t.Error("expected 'count' in key order")
		}
		if !hasTotalSalary {
			t.Error("expected 'total_salary' in key order")
		}
	})

	t.Run("fails when applied to non-table content", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department"},
			aggregates: map[string]AggregateFunc{
				"count": CountAggregate(),
			},
		}

		doc := New().
			Text("Some text content").
			Build()

		textContent := doc.GetContents()[0]
		_, err := groupByOp.Apply(context.Background(), textContent)
		if err == nil {
			t.Fatal("expected error when applying groupBy to non-table content")
		}

		if !strings.Contains(err.Error(), "groupBy operation requires table content") {
			t.Errorf("expected error about groupBy operation requiring table content, got '%s'", err.Error())
		}
	})
}

// TestAddColumnOpValidation tests AddColumnOp validation logic
func intPtr(i int) *int {
	return &i
}

// TestBuiltInAggregateFunctions tests the built-in aggregate functions
func TestBuiltInAggregateFunctions(t *testing.T) {
	t.Run("CountAggregate counts records", func(t *testing.T) {
		countFunc := CountAggregate()
		records := []Record{
			{"id": 1}, {"id": 2}, {"id": 3},
		}

		result := countFunc(records, "")
		if result != 3 {
			t.Errorf("expected count 3, got %v", result)
		}
	})

	t.Run("SumAggregate sums numeric values", func(t *testing.T) {
		sumFunc := SumAggregate("value")
		records := []Record{
			{"value": 10},
			{"value": 20},
			{"value": 30},
		}

		result := sumFunc(records, "value")
		if result != float64(60) {
			t.Errorf("expected sum 60, got %v", result)
		}
	})

	t.Run("SumAggregate handles mixed numeric types", func(t *testing.T) {
		sumFunc := SumAggregate("value")
		records := []Record{
			{"value": 10},        // int
			{"value": 20.5},      // float64
			{"value": int64(30)}, // int64
		}

		result := sumFunc(records, "value")
		if result != float64(60.5) {
			t.Errorf("expected sum 60.5, got %v", result)
		}
	})

	t.Run("SumAggregate ignores non-numeric values", func(t *testing.T) {
		sumFunc := SumAggregate("value")
		records := []Record{
			{"value": 10},
			{"value": "invalid"},
			{"value": 20},
		}

		result := sumFunc(records, "value")
		if result != float64(30) {
			t.Errorf("expected sum 30 (ignoring invalid), got %v", result)
		}
	})

	t.Run("AverageAggregate calculates mean", func(t *testing.T) {
		avgFunc := AverageAggregate("value")
		records := []Record{
			{"value": 10},
			{"value": 20},
			{"value": 30},
		}

		result := avgFunc(records, "value")
		if result != float64(20) {
			t.Errorf("expected average 20, got %v", result)
		}
	})

	t.Run("MinAggregate finds minimum", func(t *testing.T) {
		minFunc := MinAggregate("value")
		records := []Record{
			{"value": 30},
			{"value": 10},
			{"value": 20},
		}

		result := minFunc(records, "value")
		if result != float64(10) {
			t.Errorf("expected min 10, got %v", result)
		}
	})

	t.Run("MaxAggregate finds maximum", func(t *testing.T) {
		maxFunc := MaxAggregate("value")
		records := []Record{
			{"value": 30},
			{"value": 10},
			{"value": 20},
		}

		result := maxFunc(records, "value")
		if result != float64(30) {
			t.Errorf("expected max 30, got %v", result)
		}
	})

	t.Run("aggregates handle empty record sets", func(t *testing.T) {
		records := []Record{}

		if CountAggregate()(records, "") != 0 {
			t.Error("expected count 0 for empty records")
		}
		if SumAggregate("value")(records, "value") != float64(0) {
			t.Error("expected sum 0 for empty records")
		}
		if AverageAggregate("value")(records, "value") != float64(0) {
			t.Error("expected average 0 for empty records")
		}
		if MinAggregate("value")(records, "value") != float64(0) {
			t.Error("expected min 0 for empty records")
		}
		if MaxAggregate("value")(records, "value") != float64(0) {
			t.Error("expected max 0 for empty records")
		}
	})
}
