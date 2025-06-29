package drawio

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
)

//go:embed shapes/aws.json
var awsraw []byte
var awsshapes map[string]map[string]string

// AWSShape returns the shape for a desired service.
// It logs an error when the requested shape cannot be found.
// For error handling the new GetAWSShape function should be used.
func AWSShape(group string, title string) string {
	shape, err := GetAWSShape(group, title)
	if err != nil {
		log.Println(err)
	}
	return shape
}

// GetAWSShape returns the shape for a desired service and an error when it is not found.
func GetAWSShape(group, title string) (string, error) {
	shapes := AllAWSShapes()
	groupShapes, ok := shapes[group]
	if !ok {
		return "", fmt.Errorf("shape group %q not found", group)
	}
	shape, ok := groupShapes[title]
	if !ok {
		return "", fmt.Errorf("shape %q not found in group %q", title, group)
	}
	return shape, nil
}

// AllAWSShapes returns the full map of shapes
func AllAWSShapes() map[string]map[string]string {
	if awsshapes == nil {
		err := json.Unmarshal(awsraw, &awsshapes)
		if err != nil {
			log.Println(err)
		}
	}
	return awsshapes
}
