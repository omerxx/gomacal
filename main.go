package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

const (
	credentialsFile = "/tmp/credentials.json"
	tokenFile       = "/tmp/token.json"
)

func extractMeetingLink(event *calendar.Event) string {
	if event.ConferenceData != nil {
		for _, entry := range event.ConferenceData.EntryPoints {
			if entry.EntryPointType == "video" {
				return entry.Uri
			}
		}
	}

	if event.Location != "" && (strings.HasPrefix(event.Location, "http://") ||
		strings.HasPrefix(event.Location, "https://")) {
		return event.Location
	}

	if event.Description != "" {
		desc := strings.ToLower(event.Description)
		lines := strings.Split(desc, "\n")
		for _, line := range lines {
			if strings.Contains(line, "zoom.") ||
				strings.Contains(line, "teams.") ||
				strings.Contains(line, "meet.google.") ||
				strings.Contains(line, "webex.") {
				words := strings.Fields(line)
				for _, word := range words {
					if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
						return word
					}
				}
			}
		}
	}

	return ""
}

func formatEventOutput(event *calendar.Event, startTime time.Time, now time.Time) string {
	timeUntilStart := startTime.Sub(now)
	meetingLink := extractMeetingLink(event)

	var timeInfo string
	if timeUntilStart > 0 {
		timeInfo = fmt.Sprintf("starts in %v", timeUntilStart.Round(time.Minute))
	} else {
		timeSinceStart := now.Sub(startTime)
		timeInfo = fmt.Sprintf("started %v ago", timeSinceStart.Round(time.Minute))
	}

	return fmt.Sprintf("%s § %s § %s", event.Summary, timeInfo, meetingLink)
}

func main() {
	nextPtr := flag.String("next", "", "Show events within the specified duration (e.g., 5m, 1h)")
	flag.Parse()

	ctx := context.Background()

	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		fmt.Printf("Unable to read credentials file: %v\n", err)
		return
	}

	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		fmt.Printf("Unable to parse credentials: %v\n", err)
		return
	}

	client := getClient(ctx, config)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		fmt.Printf("Unable to create Calendar service: %v\n", err)
		return
	}

	now := time.Now()
	var duration time.Duration

	if *nextPtr != "" {
		duration, err = time.ParseDuration(*nextPtr)
		if err != nil {
			fmt.Printf("Invalid duration format: %v\n", err)
			return
		}
	} else {
		duration = 24 * time.Hour
	}

	startTime := now.Add(-duration)
	endTime := now.Add(duration)

	events, err := srv.Events.List("primary").
		TimeMin(startTime.Format(time.RFC3339)).
		TimeMax(endTime.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		Do()

	if err != nil {
		fmt.Printf("Unable to retrieve events: %v\n", err)
		return
	}

	relevantEvents := make([]*calendar.Event, 0)

	for _, item := range events.Items {
		startTime, err := time.Parse(time.RFC3339, item.Start.DateTime)
		if err != nil {
			continue
		}

		timeSinceStart := now.Sub(startTime)
		timeUntilStart := startTime.Sub(now)

		if (timeUntilStart >= 0 && timeUntilStart <= duration) ||
			(timeSinceStart >= 0 && timeSinceStart <= duration) {
			relevantEvents = append(relevantEvents, item)
		}
	}

	if len(relevantEvents) == 0 {
		if *nextPtr != "" {
			fmt.Printf("No events found within %s of current time\n", *nextPtr)
		} else {
			fmt.Println("No events found today")
		}
		return
	}

	for _, item := range relevantEvents {
		startTime, _ := time.Parse(time.RFC3339, item.Start.DateTime)
		fmt.Println(formatEventOutput(item, startTime, now))
	}
}

func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		tok = getTokenFromWeb(ctx, config)
		saveToken(tokenFile, tok)
	}
	return config.Client(ctx, tok)
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	codeChan := make(chan string)
	server := &http.Server{Addr: ":8080"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			fmt.Fprintf(w, "Authorization successful! You can close this window.")
			codeChan <- code
			go func() {
				time.Sleep(time.Second)
				server.Shutdown(ctx)
			}()
		}
	})

	fmt.Printf("Opening browser for authorization...\n")
	err := openBrowser(authURL)
	if err != nil {
		fmt.Printf("Could not open browser automatically. Please visit:\n%v\n", authURL)
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	code := <-codeChan

	tok, err := config.Exchange(ctx, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token: %v", err)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(file string, token *oauth2.Token) {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache OAuth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func openBrowser(url string) error {
	var err error
	switch os := runtime.GOOS; os {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}
