// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"seedfast/cli/internal/auth"
	"seedfast/cli/internal/keychain"
	"seedfast/cli/internal/manifest"

	"github.com/spf13/cobra"
)

// loginCmd represents the login command for device authentication.
// It initiates a browser-based authentication flow where the user completes login
// through a web interface, then polls the backend to verify completion.
// For the MVP implementation, it uses a magic link flow with polling verification.
var loginCmd = &cobra.Command{
	Use:     "login",
	Aliases: []string{"auth"},
	Short:   "Authenticate via browser and link this device",
	Long: `The login command initiates a device authentication flow. It generates a magic link
that the user must open in their browser to complete authentication. The command then
polls the backend service to detect when authentication is complete and stores the
resulting tokens securely.

The command supports automatic browser opening on Windows, macOS, and Linux systems.
If already logged in with valid credentials, it will skip the authentication flow.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		baseCtx := cmd.Context()
		ctx, cancel := context.WithTimeout(baseCtx, 5*time.Minute)
		defer cancel()

		// Fetch manifest from server
		m, err := manifest.GetEndpoints(ctx)
		if err != nil {
			return err
		}

		svc := auth.NewService(m.HTTPBaseURL(), m.HTTP)
		// If already logged in with a valid token, short-circuit
		if account, ok, _ := svc.WhoAmI(ctx); ok {
			fmt.Printf("Already logged in as %s\n", account)
			return nil
		}
		authURL, deviceID, pollEvery, err := svc.StartLogin(ctx)
		if err != nil {
			return err
		}
		fmt.Println("Open this link to complete login:")
		fmt.Printf("%s\n\n", authURL)

		// Try to open the user's default browser automatically while still printing the link
		openBrowser(authURL)

		// Spinner: stick-style to the left of the message; remove when done
		spinnerText := "Waiting for verification"
		frames := []string{"|", "/", "-", "\\"}
		stopSpinner := make(chan struct{})
		var spinnerWG sync.WaitGroup
		spinnerWG.Add(1)
		go func() {
			defer spinnerWG.Done()
			i := 0
			for {
				select {
				case <-stopSpinner:
					line := fmt.Sprintf("%s %s", frames[i%len(frames)], spinnerText)
					// Clear the spinner line completely, then return
					fmt.Printf("\r%*s\r", len(line), "")
					return
				case <-time.After(120 * time.Millisecond):
					line := fmt.Sprintf("%s %s", frames[i%len(frames)], spinnerText)
					fmt.Printf("\r%s", line)
					i++
				}
			}
		}()

		if pollEvery <= 0 {
			pollEvery = 3
		}
		ticker := time.NewTicker(time.Duration(pollEvery) * time.Second)
		defer ticker.Stop()

		// Immediate attempt without noisy per-attempt logging
		if account, ok, err := svc.PollLogin(ctx, deviceID); err == nil && ok {
			_ = auth.Save(auth.State{LoggedIn: true, Account: account})
			// Warm the cache for offline whoami support
			_ = svc.WarmCache(ctx)
			close(stopSpinner)
			spinnerWG.Wait()
			// Show friendly greeting with email
			showLoginGreeting(ctx, svc)
			return nil
		}
		for {
			select {
			case <-ctx.Done():
				close(stopSpinner)
				spinnerWG.Wait()
				_ = keychain.MustGetManager().ClearAuth()
				_ = auth.Clear()
				return fmt.Errorf("login timed out; cleaned up")
			case <-ticker.C:
				account, ok, err := svc.PollLogin(ctx, deviceID)
				if err != nil {
					continue
				}
				if !ok {
					continue
				}
				_ = auth.Save(auth.State{LoggedIn: true, Account: account})
				// Warm the cache for offline whoami support
				_ = svc.WarmCache(ctx)
				close(stopSpinner)
				spinnerWG.Wait()
				// Show friendly greeting with email
				showLoginGreeting(ctx, svc)
				return nil
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	// Seed random number generator for greeting selection
	rand.Seed(time.Now().UnixNano())
}

// openBrowser attempts to open the provided URL in the user's default browser.
// It uses platform-specific commands to launch the default browser:
//   - Windows: rundll32 url.dll,FileProtocolHandler
//   - macOS: open command
//   - Linux: xdg-open command
//
// The function starts the browser process but does not wait for it to complete.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

// showLoginGreeting displays a friendly greeting message with the user's email after login
func showLoginGreeting(ctx context.Context, svc *auth.Service) {
	// Try to get user data with email
	userData, err := svc.GetUserData(ctx)
	if err == nil && userData != nil {
		if email, ok := userData["email"].(string); ok && email != "" {
			fmt.Println(getRandomLoginGreeting(email))
			return
		}
		// Fallback to user_id
		if userID, ok := userData["user_id"].(string); ok && userID != "" {
			fmt.Println(getRandomLoginGreeting(userID))
			return
		}
	}
	// Generic success message if we can't get user data
	fmt.Println("✅ Login successful!")
}

// getRandomLoginGreeting returns a random greeting phrase with the user's identifier
func getRandomLoginGreeting(identifier string) string {
	greetings := []string{
		"🎉 Welcome back, %s!",
		"✨ Great to see you, %s!",
		"🚀 You're all set, %s!",
		"👋 Hello %s! Ready to seed?",
		"💫 Successfully authenticated as %s",
		"🌟 Welcome aboard, %s!",
		"⚡ Logged in as %s - let's go!",
		"✅ Authentication complete! Hi %s!",
		"🎯 You're in, %s!",
		"🔓 Access granted! Welcome %s!",
	}

	idx := rand.Intn(len(greetings))
	return fmt.Sprintf(greetings[idx], identifier)
}
