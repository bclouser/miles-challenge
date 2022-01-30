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

	"github.com/go-co-op/gocron"
)

const stravaUsersFileName = "strava_users.json"
const stravaApiClientFileName = "strava_api_client.json"

var APIClientConfig StravaAPIClient

// func stringDateToTime(date string) time.Time {
// 	out, err := time.Parse(date, date)
// 	if err != nil {
// 		fmt.Println("Failed to parse strava date into time object. Error: " + err.Error())
// 	}
// 	return out
// }

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
	currentDir, _ := os.Getwd()
	data, err := ioutil.ReadFile(currentDir + "/" + stravaUsersFileName)
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
	currentDir, _ := os.Getwd()
	if _, err := os.Stat(currentDir + "/" + stravaUsersFileName); err == nil {
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
	return ioutil.WriteFile(currentDir+"/"+stravaUsersFileName, fileBuf, 0755)
}

func Init() error {
	currentDir, _ := os.Getwd()
	if _, err := os.Stat(currentDir + "/" + stravaApiClientFileName); err != nil {
		fmt.Println("No strava api client config file found. Halting application")
		return errors.New("No strava api client config file found")
	}
	var err error
	APIClientConfig, err = ReadStravaConfig()
	if err != nil {
		fmt.Println("Failed to read in strava api config file")
		return errors.New("Faild to parse strava api client config file. Error: " + err.Error())
	}
	if _, err := os.Stat(currentDir + "/" + stravaUsersFileName); err == nil {
		fmt.Println("Stored config exists!")
		users, err := ReadUserCredentials()
		if err != nil {
			fmt.Println("Failed to read in credentials for users: " + err.Error())
			return err
		}
		fmt.Println(strconv.Itoa(len(users)) + " configured users...")
		// if len(users) > 0 {
		// 	reports, err := CreateDailyReport(users)
		// 	if err != nil {
		// 		fmt.Println("Failed to create reports: " + err.Error())
		// 		return err
		// 	}
		// 	fmt.Println("Miles this year: " + strconv.FormatFloat(float64(reports[0].YearToDate.RunMiles), 'f', 3, 32))
		// 	fmt.Println("Lifting miles this year: " + strconv.FormatFloat(float64(reports[0].YearToDate.LiftMiles), 'f', 3, 32))
		// 	fmt.Println("Total challenge miles this year: " + strconv.FormatFloat(float64(reports[0].YearToDate.LiftMiles+reports[0].YearToDate.RunMiles+reports[0].YearToDate.HikeMiles), 'f', 3, 32))
		// }
	} else {
		fmt.Println("Stored config does not exist. Add users to make it")
	}
	return nil
}

func main() {

	err := Init()
	if err != nil {
		fmt.Println("Failed to initialize. Error: " + err.Error())
		return
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Request from: " + html.EscapeString(r.URL.Path))
		query := r.URL.Query()
		fmt.Println(query)
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	http.HandleFunc("/exchange_token", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		auth_code := query["code"][0]
		formData := url.Values{
			"client_id":     {APIClientConfig.ClientID},
			"client_secret": {APIClientConfig.ClientSecret},
			"code":          {auth_code},
			"grant_type":    {"authorization_code"},
		}
		// Send request to strava to authorize user
		req, err := http.NewRequest(http.MethodPost, "https://www.strava.com/oauth/token", strings.NewReader(formData.Encode()))
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

		fmt.Fprintln(w, "Hello "+user.Athlete.Firstname+", thanks for registering. Your strava data will be included from now on")
		userReports, err := CreateDailyReport([]StravaUser{user})
		if err != nil {
			fmt.Println("Failed to Create Report: " + err.Error())
			http.Error(w, "Failed to create report. Error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		prettyJson, _ := json.MarshalIndent(&userReports[0], "", "    ")
		fmt.Fprintln(w, string(prettyJson[:]))
		//json.NewEncoder(w).Encode(&userReports[0])
	})

	//gocron.Every(1).Day().At("10:30").Do(DoDailyReport)
	fmt.Println("adding gocron")
	s := gocron.NewScheduler(time.UTC)
	s.Every(1).Minute().Do(DoDailyReport)
	s.StartAsync()

	sheets.GetSheetData()

	log.Fatal(http.ListenAndServe(":8081", nil))
}
