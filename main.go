package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/playwright-community/playwright-go"
)

type GithubApp struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	page    playwright.Page
}

const (
	GITHUB_URL       string = "https://github.com/login"
	EMAIL_LOCATOR           = "input#login_field"
	PASSWORD_LOCATOR        = "input#password"
	TOTP_LOCATOR            = "input#app_totp"
)

func main() {
	// initialize github app
	app := &GithubApp{}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", app.githubLogin)
	mux.HandleFunc("POST /twoauth", app.twoAuth)
	fmt.Println("Server started at port :9001")
	http.ListenAndServe(":9001", mux)
}

func (a *GithubApp) Initialize() error {
	err := playwright.Install(&playwright.RunOptions{
		SkipInstallBrowsers: true,
	})
	if err != nil {
		return fmt.Errorf("could not install playwright dependencies: %v", err)
	}
	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("could not start playwright: %v", err)
	}
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Channel:  playwright.String("chrome"),
		Headless: playwright.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %v", err)
	}
	page, err := browser.NewPage()
	if err != nil {
		return fmt.Errorf("could not create page: %v", err)
	}

	a.pw = pw
	a.browser = browser
	a.page = page
	return nil
}

func (a *GithubApp) githubLogin(res http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		writeJSON(res, map[string]interface{}{
			"status": http.StatusInternalServerError,
			"error":  err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	email := req.PostFormValue("email")
	password := req.PostFormValue("password")

	if email == "" || password == "" {
		writeJSON(res, map[string]interface{}{
			"status": http.StatusBadRequest,
			"error":  "email / password is required",
		}, http.StatusBadRequest)
		return
	}

	// initialize playwright configuration
	if err := a.Initialize(); err != nil {
		writeJSON(res, map[string]interface{}{
			"status": http.StatusInternalServerError,
			"error":  err.Error(),
		}, http.StatusInternalServerError)
		a.ShutdownPlaywright()
		return
	}

	if _, err := a.page.Goto(GITHUB_URL); err != nil {
		writeJSON(res, map[string]interface{}{
			"error": fmt.Sprintf("could not goto: %v", err),
		}, http.StatusInternalServerError)
		a.ShutdownPlaywright()
		return
	}

	if err := a.page.Locator(EMAIL_LOCATOR).Fill(email); err != nil {
		writeJSON(res, map[string]interface{}{
			"status": http.StatusInternalServerError,
			"error":  fmt.Sprintf("could not fill field: %v", err),
		}, http.StatusInternalServerError)
		a.ShutdownPlaywright()
		return
	}

	if err := a.page.Locator(PASSWORD_LOCATOR).Fill(password); err != nil {
		writeJSON(res, map[string]interface{}{
			"status": http.StatusInternalServerError,
			"error":  fmt.Sprintf("could not fill field: %v", err),
		}, http.StatusInternalServerError)
		a.ShutdownPlaywright()
		return
	}

	if err := a.page.Locator("input.btn.btn-primary.btn-block.js-sign-in-button").Click(); err != nil {
		writeJSON(res, map[string]interface{}{
			"status": http.StatusInternalServerError,
			"error":  fmt.Sprintf("could not click button: %v", err),
		}, http.StatusInternalServerError)
		a.ShutdownPlaywright()
		return
	}

	err := a.page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})
	if err != nil {
		writeJSON(res, map[string]interface{}{
			"status": http.StatusInternalServerError,
			"error":  fmt.Sprintf("could not wait for load state: %v", err),
		}, http.StatusInternalServerError)
		a.ShutdownPlaywright()
		return
	}

	errorMsg := a.page.Locator("div.js-flash-alert")
	err = playwright.NewPlaywrightAssertions(1000).Locator(errorMsg).ToBeVisible()
	if err == nil {
		// get error message from element if it's visible
		content, err := errorMsg.First().TextContent()
		if err != nil {
			writeJSON(res, map[string]interface{}{
				"status": http.StatusInternalServerError,
				"error":  fmt.Sprintf("could not wait for load state: %v", err),
			}, http.StatusInternalServerError)
			a.ShutdownPlaywright()
			return
		}

		writeJSON(res, map[string]interface{}{
			"status": http.StatusBadRequest,
			"error":  strings.TrimSpace(content),
		}, http.StatusBadRequest)
		a.ShutdownPlaywright()
		return

	}

	writeJSON(res, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Login successful, moving to next",
	}, http.StatusOK)

}

func (a *GithubApp) twoAuth(res http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		writeJSON(res, map[string]interface{}{
			"status": http.StatusInternalServerError,
			"error":  err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	code := req.PostFormValue("code")
	if code == "" {
		writeJSON(res, map[string]interface{}{
			"status": http.StatusBadRequest,
			"error":  "email / password is required",
		}, http.StatusBadRequest)
		return
	}

	if err := a.page.Locator(TOTP_LOCATOR).Fill(code); err != nil {
		writeJSON(res, map[string]interface{}{
			"status": http.StatusInternalServerError,
			"error":  fmt.Sprintf("could not fill field: %v", err),
		}, http.StatusInternalServerError)
		a.ShutdownPlaywright()
		return
	}

	err := a.page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})
	if err != nil {
		writeJSON(res, map[string]interface{}{
			"status": http.StatusInternalServerError,
			"error":  fmt.Sprintf("could not wait for load state: %v", err),
		}, http.StatusInternalServerError)
		a.ShutdownPlaywright()
		return
	}

	errorMsg := a.page.Locator("div.js-flash-alert")
	err = playwright.NewPlaywrightAssertions(1000).Locator(errorMsg).ToBeVisible()
	if err == nil {
		// get error message from element if it's visible
		content, err := errorMsg.First().TextContent()
		if err != nil {
			writeJSON(res, map[string]interface{}{
				"status": http.StatusInternalServerError,
				"error":  fmt.Sprintf("could not wait for load state: %v", err),
			}, http.StatusInternalServerError)
			a.ShutdownPlaywright()
			return
		}

		writeJSON(res, map[string]interface{}{
			"status": http.StatusBadRequest,
			"error":  strings.TrimSpace(content),
		}, http.StatusBadRequest)
		return
	}

	// wait for new page to load
	a.page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})

	// get username
	username, err := a.page.Locator("span.Button-label > span.color-fg-muted + span").First().TextContent()
	fmt.Println(err)
	if err != nil {
		writeJSON(res, map[string]interface{}{
			"status": http.StatusInternalServerError,
			"error":  fmt.Sprintf("could not wait for load state: %v", err),
		}, http.StatusInternalServerError)
		a.ShutdownPlaywright()
		return
	}

	a.page.Screenshot(playwright.PageScreenshotOptions{
		FullPage: playwright.Bool(true),
		Path:     playwright.String(fmt.Sprintf("%s.png", username)),
	})

	// get github cookie
	cookie, err := a.page.Context().Cookies(a.page.URL())
	if err != nil {
		writeJSON(res, map[string]interface{}{
			"status": http.StatusInternalServerError,
			"error":  fmt.Sprintf("could not wait for load state: %v", err),
		}, http.StatusInternalServerError)
		a.ShutdownPlaywright()
		return
	}

	// close playwright after successfull login
	a.ShutdownPlaywright()

	writeJSON(res, map[string]interface{}{
		"status":   http.StatusOK,
		"username": username,
		"message":  "Login complete",
		"cookie":   cookie,
	}, http.StatusOK)
}

func writeJSON(res http.ResponseWriter, param any, statusCode int) {
	res.WriteHeader(statusCode)
	json.NewEncoder(res).Encode(&param)
}

func (a *GithubApp) ShutdownPlaywright() {
	if err := a.page.Close(); err != nil {
		panic(err)
	}

	if err := a.browser.Close(); err != nil {
		panic(err)
	}

	if err := a.pw.Stop(); err != nil {
		panic(err)
	}
}
