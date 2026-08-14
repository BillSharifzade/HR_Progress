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

		createMissing = flag.Bool("create-missing", false,
			"create an employee for a respondent found in neither the database nor 1F (requires -onef)")
		oneFFile = flag.String("onef", "",
			"raw 1F payload, used to tell an unknown person from one whose sync is failing")
	)
	flag.Parse()

	if *file == "" || *dsn == "" {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "\n-file is required, and DATABASE_URL (or -dsn) must be set")
		os.Exit(2)
	}
	if *createMissing && *oneFFile == "" {
		// Without the roster we cannot distinguish "nobody has this person" from
		// "1F has them but the sync is failing", and creating a local user in the
		// second case duplicates them permanently — the sync keys on
		// one_f_user_id and will never recognise the row we made.
		fmt.Fprintln(os.Stderr, "-create-missing requires -onef: refusing to create users without checking 1F first")
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

	importer := profileimport.New(pool)
	if *createMissing {
		roster, err := profileimport.LoadOneFRoster(*oneFFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "read 1F payload:", err)
			os.Exit(1)
		}
		fmt.Printf("1F roster: %d people (used to avoid creating duplicates)\n", roster.Size())
		importer = importer.WithCreateMissing(roster)
	}

	report, err := importer.Run(ctx, parsed, !*commit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "import failed:", err)
		os.Exit(1)
	}

	fmt.Print(report)
	if !*commit {
		fmt.Println("\nNothing was written. Re-run with -commit to apply.")
	}
}
