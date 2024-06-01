// aws-federated-headless-login

package main

import (
	"bufio"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/theckman/yacspin"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

var cfg = yacspin.Config{
	Frequency:         100 * time.Millisecond,
	CharSet:           yacspin.CharSets[59],
	Suffix:            "AWS Identity Center Sign in: ",
	SuffixAutoColon:   false,
	Message:           "",
	StopCharacter:     "✓",
	StopFailCharacter: "✗",
	StopMessage:       "Logged in successfully",
	StopFailMessage:   "Log in failed",
	StopColors:        []string{"fgGreen"},
}

var spinner, _ = yacspin.New(cfg)

func main() {
	// Define and parse command-line flags
	show := flag.Bool("show", false, "Show the browser (disable headless mode)")
	flag.Parse()

	// Handle SIGPIPE signal to prevent broken pipe error
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGPIPE)
	go func() {
		<-signalChan
		os.Exit(0)
	}()

	// start spinner
	spinner.Start()

	// get sso url from stdin
	url := getURLfromstdin()

	// start aws sso login
	ssoLogin(url, *show)

	// stop spinner
	spinner.Stop()

	// sleep before exiting
	time.Sleep(1 * time.Second)
}

// returns sso url from stdin.
func getURLfromstdin() string {
	// update spinner message
	spinner.Message("reading url from stdin")

	// read in from os.Stdin the output from aws cli
	scanner := bufio.NewScanner(os.Stdin)
	defer os.Stdin.Close() // Close os.Stdin when function exits

	// get the URL from the output
	url := ""
	for url == "" {
		scanner.Scan()
		text := scanner.Text()
		result, _ := regexp.Compile("^https.*user_code=([A-Z]{4}-?){2}")

		if result.MatchString(text) {
			url = text
		}
	}

	return url
}

func ssoLogin(url string, show bool) {
	// Set up the launcher with or without headless mode based on the `show` variable
	launchurl := launcher.New().Headless(!show).MustLaunch()

	// Update spinner message and pause spinner
	spinner.Message(color.MagentaString("init headless-browser \n"))
	spinner.Pause()

	// Connect to the browser
	browser := rod.New().ControlURL(launchurl).MustConnect().Trace(false)

	// Load Cookies
	loadCookies(*browser)

	// Try automation
	err := rod.Try(func() {
		spinner.Message("opening url")
		page := browser.MustPage(url)

		// pause spinner and update message
		spinner.Unpause()
		spinner.Message("clicking Confirm and continue button")

		// check for confirm and continue button
		confirmButton := page.MustElementR("button", "Confirm and continue")
		confirmButton.MustWaitEnabled().MustClick()
		time.Sleep(1 * time.Second)

		// Loop to find and click the "Allow" button
		spinner.Message("waiting for Allow button")
		var allowButton *rod.Element
		var err error
		for {
			allowButton, err = page.ElementR("button", "Allow")
			if err != nil || allowButton == nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			break
		}
		if allowButton != nil {
			allowButton.MustWaitVisible().MustClick()
		}

		// save the cookies
		saveCookies(*browser)
	})

	// any errors manage them
	if errors.Is(err, context.DeadlineExceeded) {
		panic("Timed out waiting for page")
	} else if err != nil {
		// Handle BrokenPipeError
		if err.Error() == "write on closed pipe" {
			os.Exit(0) // Exit gracefully
		}
		panic(err.Error())
	}
}

// print error message
func errormsg(errorMsg string) {
	yellow := color.New(color.FgYellow).SprintFunc()
	spinner.Message("Warn: " + yellow(errorMsg))
}

// print error message and exit
func panic(errorMsg string) {
	red := color.New(color.FgRed).SprintFunc()
	spinner.StopFailMessage(red("Fatal Panic and Exit. Error: " + errorMsg))
	spinner.StopFail()
	os.Exit(1)
}

// load cookies so we can re-use the auth from first full login
func loadCookies(browser rod.Browser) {
	spinner.Message("loading cookies")
	dirname, err := os.UserHomeDir()
	if err != nil {
		errormsg(err.Error())
	}

	data, _ := os.ReadFile(dirname + "/.aws-federated-headless-login")

	sEnc, _ := b64.StdEncoding.DecodeString(string(data))
	var cookie *proto.NetworkCookie
	json.Unmarshal(sEnc, &cookie)

	if cookie != nil {
		browser.MustSetCookies(cookie)
	}
}

// save the aws retuned authentication cookie so we don't keep getting prompted to login
func saveCookies(browser rod.Browser) {
	dirname, err := os.UserHomeDir()
	if err != nil {
		errormsg(err.Error())
	}

	cookies := (browser.MustGetCookies())

	for _, cookie := range cookies {
		if cookie.Name == "x-amz-sso_authn" {
			data, _ := json.Marshal(cookie)

			sEnc := b64.StdEncoding.EncodeToString([]byte(data))
			err = os.WriteFile(dirname+"/.aws-federated-headless-login", []byte(sEnc), 0644)

			if err != nil {
				errormsg("Failed to save x-amz-sso_authn cookie")
			}
			break
		}
	}
}
