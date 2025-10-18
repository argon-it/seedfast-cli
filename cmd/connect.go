package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"seedfast/cli/internal/auth"
	"seedfast/cli/internal/keychain"
	"seedfast/cli/internal/terminal"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

// connectCmd represents the connect command for establishing database connections.
// It prompts the user for a PostgreSQL DSN and verifies connectivity before saving
// the connection details securely in the OS keychain.
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Configure and verify PostgreSQL database connection",
	Long: `The connect command prompts for a PostgreSQL DSN (Data Source Name) and verifies
the connection to ensure the database is accessible. The connection details are
securely stored in the OS keychain for future use.

Example DSN format: postgres://user:password@host:5432/database?sslmode=disable`,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := auth.Load()
		if err != nil || !st.LoggedIn {
			fmt.Println("⚠️  You need to be logged in to configure database connections.")
			fmt.Println("   Please run: seedfast login")
			return nil
		}
		ctx := cmd.Context()
		reader := bufio.NewReader(os.Stdin)
		promptText := "Enter Postgres DSN (e.g., postgres://user:pass@host:5432/db?sslmode=disable): "
		fmt.Print(promptText)
		dsn, _ := reader.ReadString('\n')
		dsn = strings.TrimSpace(dsn)

		// Clear the prompt and user input from terminal
		terminal.ClearPreviousLines(len(promptText) + len(dsn))

		if dsn == "" {
			return errors.New("DSN is required")
		}

		// Start lightweight inline spinner (Windows-friendly)
		startTime := time.Now()
		done := make(chan struct{})
		spinnerStopped := make(chan struct{})
		stopped := false
		stopSpinner := func() {
			if !stopped {
				close(done)
				<-spinnerStopped
				stopped = true
			}
		}
		go func() {
			defer close(spinnerStopped)
			frames := []string{"-", "\\", "|", "/"}
			i := 0
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					// Clear the line to remove any spinner remnants
					fmt.Print("\r")
					fmt.Print(strings.Repeat(" ", 60))
					fmt.Print("\r")
					return
				case <-ticker.C:
					frame := frames[i%len(frames)]
					i++
					fmt.Printf("\r%s verifying connection", frame)
				}
			}
		}()

		// Verify connection
		ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		pool, err := pgxpool.New(ctxPing, dsn)
		if err != nil {
			stopSpinner()
			fmt.Println("Invalid DSN format. Please check your connection string and try again.")
			fmt.Println("Example: postgres://user:password@host:5432/database?sslmode=disable")
			return err
		}
		defer pool.Close()
		if err := pool.Ping(ctxPing); err != nil {
			stopSpinner()
			fmt.Println("Connection failed. Please check your database credentials and network connection.")
			return err
		}

		// Ensure spinner runs for at least 2 seconds for better UX
		if elapsed := time.Since(startTime); elapsed < 2*time.Second {
			time.Sleep(2*time.Second - elapsed)
		}

		// Stop spinner and overwrite with success message
		stopSpinner()
		fmt.Println("✅ Database connection verified and saved!")
		fmt.Println("   You're ready to run 'seedfast seed'")

		// Save DSN securely in the OS keychain
		if err := keychain.MustGetManager().SaveDBDSN(dsn); err != nil {
			fmt.Println("❌ Failed to save connection details securely.")
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
}
