package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArjenSchwarz/go-output/v2/migrate"
)

func main() {
	var (
		sourceDir  = flag.String("source", "", "Source directory containing v1 code to migrate")
		targetDir  = flag.String("target", "", "Target directory for migrated v2 code (optional, defaults to source)")
		singleFile = flag.String("file", "", "Single file to migrate (alternative to -source)")
		dryRun     = flag.Bool("dry-run", false, "Show what would be changed without writing files")
		verbose    = flag.Bool("verbose", false, "Show detailed migration information")
		showHelp   = flag.Bool("help", false, "Show this help message")
	)

	flag.Parse()

	if *showHelp {
		printUsage()
		return
	}

	if *sourceDir == "" && *singleFile == "" {
		fmt.Fprintf(os.Stderr, "Error: Either -source or -file must be specified\n\n")
		printUsage()
		os.Exit(1)
	}

	migrator := migrate.New()

	if *singleFile != "" {
		err := migrateSingleFile(migrator, *singleFile, *targetDir, *dryRun, *verbose)
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	} else {
		err := migrateDirectory(migrator, *sourceDir, *targetDir, *dryRun, *verbose)
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}
}

func printUsage() {
	fmt.Println("Go-Output v1 to v2 Migration Tool")
	fmt.Println("==================================")
	fmt.Println()
	fmt.Println("This tool automatically migrates Go code from go-output v1 API to v2 API.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  migrate -source <directory>     Migrate all .go files in directory")
	fmt.Println("  migrate -file <file.go>         Migrate a single file")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -source <dir>      Source directory containing v1 code")
	fmt.Println("  -file <file>       Single file to migrate")
	fmt.Println("  -target <dir>      Target directory (defaults to source directory)")
	fmt.Println("  -dry-run           Show changes without writing files")
	fmt.Println("  -verbose           Show detailed migration information")
	fmt.Println("  -help              Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  migrate -source ./myproject -dry-run")
	fmt.Println("  migrate -file main.go -target ./v2-version")
	fmt.Println("  migrate -source ./src -verbose")
}

func migrateSingleFile(migrator *migrate.Migrator, filename, targetDir string, dryRun, verbose bool) error {
	if verbose {
		fmt.Printf("Migrating file: %s\n", filename)
	}

	result, err := migrator.MigrateFile(filename)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	if verbose {
		printMigrationResult(result)
	}

	if !dryRun {
		targetFile := filename
		if targetDir != "" {
			targetFile = filepath.Join(targetDir, filepath.Base(filename))
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return fmt.Errorf("creating target directory: %w", err)
			}
		}

		if err := os.WriteFile(targetFile, []byte(result.TransformedFile), 0644); err != nil {
			return fmt.Errorf("writing migrated file: %w", err)
		}

		fmt.Printf("Migrated: %s -> %s\n", filename, targetFile)
	} else {
		fmt.Printf("Would migrate: %s\n", filename)
		if verbose {
			fmt.Println("--- Transformed Code ---")
			fmt.Println(result.TransformedFile)
			fmt.Println("--- End Transformed Code ---")
		}
	}

	return nil
}

func migrateDirectory(migrator *migrate.Migrator, sourceDir, targetDir string, dryRun, verbose bool) error {
	if targetDir == "" {
		targetDir = sourceDir
	}

	if verbose {
		fmt.Printf("Migrating directory: %s -> %s\n", sourceDir, targetDir)
	}

	results, err := migrator.MigrateDirectory(sourceDir)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	var totalFiles, successfulFiles int
	var allErrors []error

	for _, result := range results {
		totalFiles++

		if verbose {
			printMigrationResult(result)
		}

		if len(result.Errors) > 0 {
			allErrors = append(allErrors, result.Errors...)
			fmt.Printf("Errors in %s:\n", result.OriginalFile)
			for _, err := range result.Errors {
				fmt.Printf("  - %v\n", err)
			}
		} else {
			successfulFiles++
		}

		if !dryRun {
			// Determine target file path
			relPath, err := filepath.Rel(sourceDir, result.OriginalFile)
			if err != nil {
				return fmt.Errorf("calculating relative path: %w", err)
			}

			targetFile := filepath.Join(targetDir, relPath)
			targetDirPath := filepath.Dir(targetFile)

			if err := os.MkdirAll(targetDirPath, 0755); err != nil {
				return fmt.Errorf("creating target directory %s: %w", targetDirPath, err)
			}

			if err := os.WriteFile(targetFile, []byte(result.TransformedFile), 0644); err != nil {
				return fmt.Errorf("writing migrated file %s: %w", targetFile, err)
			}

			fmt.Printf("Migrated: %s -> %s\n", result.OriginalFile, targetFile)
		}
	}

	fmt.Printf("\nMigration Summary:\n")
	fmt.Printf("Total files: %d\n", totalFiles)
	fmt.Printf("Successful: %d\n", successfulFiles)
	if len(allErrors) > 0 {
		fmt.Printf("Errors: %d\n", len(allErrors))
	}

	if dryRun {
		fmt.Println("(Dry run - no files were modified)")
	}

	return nil
}

func printMigrationResult(result *migrate.MigrationResult) {
	fmt.Printf("\nFile: %s\n", result.OriginalFile)

	if len(result.PatternsFound) > 0 {
		fmt.Printf("Patterns found: %s\n", strings.Join(result.PatternsFound, ", "))
	}

	if len(result.RulesApplied) > 0 {
		fmt.Printf("Rules applied: %s\n", strings.Join(result.RulesApplied, ", "))
	}

	if len(result.Warnings) > 0 {
		fmt.Printf("Warnings:\n")
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("Errors:\n")
		for _, err := range result.Errors {
			fmt.Printf("  - %v\n", err)
		}
	}
}
