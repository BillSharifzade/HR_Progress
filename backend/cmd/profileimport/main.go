// Command profileimport loads the digital-profile questionnaire into employee
// profiles.
//
// It reads the JSON intermediate produced by scripts/form_to_json.py, matches
// each response to an employee by name, and writes hobbies, the structured
// profile, languages, prior employment, open-ended answers and certificate
// links. Defaults to a dry run: matching happens for real, the writes are
// rolled back, and the report shows exactly what a commit would do.
//
//	profileimport -file form.json                # dry run
//	profileimport -file form.json -commit        # write
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"hrprogress/internal/db"
	"hrprogress/internal/profileimport"
)

func main() {
	var (
		file    = flag.String("file", "", "path to the JSON produced by form_to_json.py (required)")
		commit  = flag.Bool("commit", false, "write the changes; without this the run is a dry run")
		dsn     = flag.String("dsn", os.Getenv("DATABASE_URL"), "Postgres connection string")
		timeout = flag.Duration("timeout", 5*time.Minute, "overall timeout")
	)
	flag.Parse()

	if *file == "" || *dsn == "" {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "\n-file is required, and DATABASE_URL (or -dsn) must be set")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	parsed, err := profileimport.LoadFile(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read form file:", err)
		os.Exit(1)
	}

	pool, err := db.NewPool(ctx, *dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}
	defer pool.Close()

	report, err := profileimport.New(pool).Run(ctx, parsed, !*commit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "import failed:", err)
		os.Exit(1)
	}

	fmt.Print(report)
	if !*commit {
		fmt.Println("\nNothing was written. Re-run with -commit to apply.")
	}
}
