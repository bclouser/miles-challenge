package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bclouser/miles-challenge/sheets"
	// "github.com/bclouser/miles-challenge/slack"
	"github.com/go-co-op/gocron"
	"github.com/gorilla/mux"
)

const stravaUsersFileName = "strava_users.json"
const stravaApiClientFileName = "strava_api_client.json"
const authCodeInputUrl = "https://miles-challenge.multiplewanda.com/api/gc/auth-code"

var APIClientConfig StravaAPIClient

// func stringDateToTime(date string) time.Time {
// 	out, err := time.Parse(date, date)
// 	if err != nil {
// 		fmt.Println("Failed to parse strava date into time object. Error: " + err.Error())
// 	}
// 	return out
// }

type Config struct {
	SlackChannelHookUrl            string
	StravaAPIClientID              string
	StravaAPIClientSecret          string
	StravaAPITokenEndpoint         string
	GoogleSheetsID                 string
	GoogleCloudCredentialsFilePath string
	GoogleCloudSavedTokenPath      string // Where the saved token will be stored
	NonVolatileStorageDir          string
}

var config Config

func metersToMiles(meters float32) float32 {
	const metersPerMile float32 = 1609.344
	return meters / metersPerMile
}

// Refresh Token will get a new token and replace the existing tokens in the stored config file
func RefreshToken(user StravaUser, persist bool) (StravaUser, error) {
	formData := url.Values{
		"client_id":     {APIClientConfig.ClientID},
		"client_secret": {APIClientConfig.ClientSecret},
		"refresh_token": {user.RefreshToken},
		"grant_type":    {"refresh_token"},
	}
	// Send request to strava to authorize user
	req, err := http.NewRequest(http.MethodPost, APIClientConfig.TokenEndpoint, strings.NewReader(formData.Encode()))
	if err != nil {
		fmt.Println("Failed to create request to strava")
		return user, err
	}
	req.Header.Add("Content-Type", "multipart/form-data")
	respBuf := bytes.Buffer{}
	client := &http.Client{}
	resp, err := client.Do(req)
	// Non nil errors means the http request didn't get off the ground. It doesn't mean non 2XX
	if err != nil {
		fmt.Println("Failed to send out http request")
		return user, err
	}

	respBuf.ReadFrom(resp.Body)
	resp.Body.Close()

	if resp.StatusCode >= 300 {
		fmt.Println("Request returned http status: " + resp.Status)
		return user, errors.New("Request returned non 200 status " + resp.Status)
	}
	freshTokenUser := StravaUser{}
	err = json.Unmarshal(respBuf.Bytes(), &freshTokenUser)
	if err != nil {
		fmt.Println("Failed to unmarshal json from strava response into user. Error: " + err.Error())
		return user, err
	}
	user.AccessToken = freshTokenUser.AccessToken
	user.RefreshToken = freshTokenUser.RefreshToken

	if persist {
		if AddUserCredentials(user) != nil {
			fmt.Println("Failed to add updated user to credentials file")
			return user, err
		}
	}
	return user, nil
}

func GetUserActivitiesForCurrentYear(accessToken string) ([]SummaryActivity, error) {
	// "https://www.strava.com/api/v3/athlete/activities?before=&after=&page=&per_page=" "Authorization: Bearer [[token]]"
	activities := []SummaryActivity{}
	beginYear, _ := time.Parse("2006-01-02", "2021-12-31")
	const pageLen = 100

	// Deal with pagination
	for i := 0; i < 100; i++ {
		pageActivities := []SummaryActivity{}
		params := url.Values{}
		params.Add("after", strconv.FormatInt(beginYear.Unix(), 10))
		params.Add("per_page", strconv.Itoa(pageLen))
		params.Add("page", strconv.Itoa(1+i))
		// fmt.Println("Params look like: " + params.Encode())
		req, err := http.NewRequest(http.MethodGet, "https://www.strava.com/api/v3/athlete/activities?"+params.Encode(), nil)
		if err != nil {
			fmt.Println("Failed to create request to Get user activities on strava. Error " + err.Error())
			return activities, err
		}
		req.Header.Add("Authorization", "Bearer "+accessToken)
		respBuf := bytes.Buffer{}
		client := &http.Client{}
		resp, err := client.Do(req)
		// Non nil errors means the http request didn't get off the ground. It doesn't mean non 2XX
		if err != nil {
			fmt.Println("Failed to send out http request. Error: " + err.Error())
			return activities, err
		}

		respBuf.ReadFrom(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 300 {
			fmt.Println("Request returned http status: " + resp.Status)
			fmt.Println(respBuf.String())
			return activities, err
		}
		err = json.Unmarshal(respBuf.Bytes(), &pageActivities)
		if err != nil {
			fmt.Println("Failed to unmarshal json strava activities into structs. Error: " + err.Error())
			return activities, err
		}
		activities = append(activities, pageActivities...)
		if len(pageActivities) < pageLen {
			break
		}
	}
	return activities, nil
}

// Function legacy... We don't have a strava config file anymore
func ReadStravaConfig() (StravaAPIClient, error) {
	apiConfig := StravaAPIClient{}
	currentDir, _ := os.Getwd()
	data, err := ioutil.ReadFile(currentDir + "/" + stravaApiClientFileName)
	if err != nil {
		fmt.Println("Error reading strava api client config from file", err.Error())
		return apiConfig, err
	}
	err = json.Unmarshal(data, &apiConfig)
	return apiConfig, err
}

func ReadUserCredentials() ([]StravaUser, error) {
	data, err := ioutil.ReadFile(config.NonVolatileStorageDir + "/" + stravaUsersFileName)
	if err != nil {
		fmt.Println("Error reading strava users from file", err.Error())
		return nil, err
	}
	usersFile := []StravaUser{}
	err = json.Unmarshal(data, &usersFile)
	return usersFile, err
}

func AddUserCredentials(user StravaUser) error {
	// Should we care about duplicates????
	users := []StravaUser{}
	if _, err := os.Stat(config.NonVolatileStorageDir + "/" + stravaUsersFileName); err == nil {
		existingUsers, err := ReadUserCredentials()
		if err != nil {
			fmt.Println("Failed to read in existing strava users from file. Error: " + err.Error())
			return err
		}
		users = append(users, existingUsers...)
	}
	overWritten := false
	if len(users) > 0 {
		for i, existingUser := range users {
			if user.Athlete.ID == existingUser.Athlete.ID {
				fmt.Println("User with ID: " + strconv.Itoa(user.Athlete.ID) + " already exists in stored config. Overwriting...")
				// I am not actually sure this works... the whole modifying a list inside a for-loop thing
				users[i] = user
				overWritten = true
			}
		}
	}
	if !overWritten {
		users = append(users, user)
	}
	fileBuf, err := json.Marshal(users)
	if err != nil {
		fmt.Println("Failed to marshal file as json: " + err.Error())
		return err
	}
	return ioutil.WriteFile(config.NonVolatileStorageDir+"/"+stravaUsersFileName, fileBuf, 0755)
}

func Init() error {
	config.SlackChannelHookUrl = os.Getenv("SLACK_CHANNEL_HOOK_URL")
	config.StravaAPIClientID = os.Getenv("STRAVA_API_CLIENT_ID")
	config.StravaAPIClientSecret = os.Getenv("STRAVA_API_CLIENT_SECRET")
	config.StravaAPITokenEndpoint = os.Getenv("STRAVA_TOKEN_ENDPOINT")
	config.GoogleSheetsID = os.Getenv("GOOGLE_SHEETS_SHEET_ID")
	config.GoogleCloudCredentialsFilePath = os.Getenv("GOOGLE_CLOUD_CREDENTIALS_PATH")
	config.NonVolatileStorageDir = os.Getenv("NON_VOLATILE_STORAGE_DIR")

	if config.SlackChannelHookUrl == "" {
		return errors.New("Error: `SLACK_CHANNEL_HOOK_URL` env variable not set")
	}
	if config.StravaAPIClientID == "" {
		return errors.New("Error: `STRAVA_API_CLIENT_ID` env variable not set")
	}
	if config.StravaAPIClientSecret == "" {
		return errors.New("Error: `STRAVA_API_CLIENT_SECRET` env variable not set")
	}
	if config.StravaAPITokenEndpoint == "" {
		return errors.New("Error: `STRAVA_TOKEN_ENDPOINT` env variable not set")
	}
	if config.GoogleSheetsID == "" {
		return errors.New("Error: `GOOGLE_SHEETS_SHEET_ID` env variable not set")
	}
	if config.GoogleCloudCredentialsFilePath == "" {
		return errors.New("Error: `GOOGLE_CLOUD_CREDENTIALS_PATH` env variable not set")
	}
	if config.NonVolatileStorageDir == "" {
		return errors.New("Error: `NON_VOLATILE_STORAGE_DIR` env variable not set")
	}

	config.GoogleCloudSavedTokenPath = config.NonVolatileStorageDir + "/gc-token.json"

	APIClientConfig.ClientID = config.StravaAPIClientID
	APIClientConfig.ClientSecret = config.StravaAPIClientSecret
	APIClientConfig.TokenEndpoint = config.StravaAPITokenEndpoint

	if _, err := os.Stat(config.NonVolatileStorageDir + "/" + stravaUsersFileName); err == nil {
		fmt.Println("Stored strava users file exists!")
		users, err := ReadUserCredentials()
		if err != nil {
			fmt.Println("Failed to read in credentials for users: " + err.Error())
			return err
		}
		fmt.Println(strconv.Itoa(len(users)) + " configured users...")
	} else {
		fmt.Println("No strava user's file found. Add users to create it")
	}

	// Initialize google cloud api stuffs
	err := sheets.Initialize(config.GoogleCloudCredentialsFilePath,
		config.GoogleCloudSavedTokenPath,
		authCodeInputUrl)

	return err

}

func main() {
	err := Init()
	if err != nil {
		fmt.Println("Initialization failure: " + err.Error())
		return
	}

	rtr := mux.NewRouter()

	rtr.HandleFunc("/api/gc/auth-code", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		code := query.Get("code")
		if code == "" {
			http.Error(w, "Missing code in query params", http.StatusBadRequest)
			return
		}
		scope := query.Get("scope")
		if scope == "" {
			http.Error(w, "Missing scope in query params", http.StatusBadRequest)
			return
		}

		err := sheets.SetAuthCodeRetrievedFromWeb(code)
		if err != nil {
			fmt.Println("Failed to get token from auth code: " + err.Error())
		}
	}).Methods("GET")

	rtr.HandleFunc("/api/slack/post-report", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Request from: " + html.EscapeString(r.URL.Path))
		report := "*    Requested Report!* \n\n" + GenerateFormattedReport()

		// reqStruct := struct {
		// 	Text string `json:"text"`
		// }{Text: report}

		// data, _ := json.Marshal(reqStruct)
		fmt.Fprintln(w, report)
	})

	rtr.HandleFunc("/api/strava/auth-code", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		auth_code := query["code"][0]
		formData := url.Values{
			"client_id":     {APIClientConfig.ClientID},
			"client_secret": {APIClientConfig.ClientSecret},
			"code":          {auth_code},
			"grant_type":    {"authorization_code"},
		}
		// Send request to strava to authorize user
		req, err := http.NewRequest(http.MethodPost, APIClientConfig.TokenEndpoint, strings.NewReader(formData.Encode()))
		if err != nil {
			fmt.Println("Failed to create request to strava")
			http.Error(w, "Failed to create request to strava", http.StatusInternalServerError)
			return
		}
		req.Header.Add("Content-Type", "multipart/form-data")
		respBuf := bytes.Buffer{}
		client := &http.Client{}
		resp, err := client.Do(req)
		// Non nil errors means the http request didn't get off the ground. It doesn't mean non 2XX
		if err != nil {
			fmt.Println("Failed to send out http request")
			http.Error(w, "Failed to send out http request", http.StatusInternalServerError)
		}

		respBuf.ReadFrom(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 300 {
			fmt.Println("Request returned http status: " + resp.Status)
			http.Error(w, "Request to strava returned invalid response: "+resp.Status, resp.StatusCode)
		}
		user := StravaUser{Athlete: StravaAthlete{}}
		err = json.Unmarshal(respBuf.Bytes(), &user)
		if err != nil {
			fmt.Println("Failed to unmarshal json from strava response into structs. Error: " + err.Error())
			http.Error(w, "Failed to unmarshal json from strava response into structs. Error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = AddUserCredentials(user)
		if err != nil {
			fmt.Println("Failed to add user to local credentials file. Error: " + err.Error())
			http.Error(w, "Failed to add user to local credentials file. Error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintln(w, "Hello "+user.Athlete.Firstname+", thanks for registering. Your strava data will be included in the challange from now on")
		userReports, err := GetStravaReport([]StravaUser{user})
		if err != nil {
			fmt.Println("Failed to Create Report: " + err.Error())
			http.Error(w, "Failed to create report. Error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		prettyJson, _ := json.MarshalIndent(&userReports[0], "", "    ")
		fmt.Fprintln(w, "Current Data From Strava:")
		fmt.Fprintln(w, string(prettyJson[:]))
	})

	rtr.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Unmatched request for: " + r.Method + " " + html.EscapeString(r.URL.Path))
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	http.Handle("/", rtr)

	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		fmt.Println("Failed to load timezone America/New_York. Error: " + err.Error())
		return
	}
	s := gocron.NewScheduler(newYork)
	// Daily at 8:30 pm
	s.Every(1).Day().At("20:30").Do(DoDailyReport)
	s.StartAsync()

	sheets.GetSheetData(config.GoogleSheetsID, config.GoogleCloudCredentialsFilePath, config.GoogleCloudSavedTokenPath, authCodeInputUrl)

	fmt.Println("Starting web server... on port 8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
