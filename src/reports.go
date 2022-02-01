package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bclouser/miles-challenge/sheets"
)

type Report struct {
	MostMiles string `json:"most_miles"`
}

// Total miles Actually Run This Year (run_miles)
// Total miles counting towards challenge (challenge_miles)
// Total miles from lifting activities (lift_miles)
// Total miles from hiking activities (hike_miles)
// run_miles + lift_run_miles + hike_miles = challenge_miles

// Total miles

type AthleteCounts struct {
	RunMiles  float32 `json:"run_miles"`
	HikeMiles float32 `json:"hike_miles"`
	LiftMiles float32 `json:"lift_miles"`
}

type UserReport struct {
	AthleteID        int           `json:"athlete_id"`
	AthleteFirstName string        `json:"athlete_firstname"`
	YearToDate       AthleteCounts `json:"year_to_date"`
	Day              AthleteCounts `json:"day"`
}

func GetStravaReport(users []StravaUser) ([]UserReport, error) {
	reports := []UserReport{}
	now := time.Now()
	for _, user := range users {
		freshUser, err := RefreshToken(user, true)
		if err != nil {
			fmt.Println("Failed to refresh token. Error: " + err.Error())
			return reports, err
		}
		currentReport := UserReport{AthleteID: freshUser.Athlete.ID, AthleteFirstName: freshUser.Athlete.Firstname}

		// Get User's activity for this year
		activities, err := GetUserActivitiesForCurrentYear(freshUser.AccessToken)
		if err != nil {
			fmt.Println("Failed to get activites for user: " + freshUser.Athlete.Firstname + " error: " + err.Error())
		}

		totalActivities := len(activities)
		fmt.Println(strconv.Itoa(totalActivities) + " activities posted in the last year for: " + freshUser.Athlete.Firstname)
		for _, activity := range activities {
			fmt.Println("Date: " + activity.StartDateLocal.String() + " Name: " + activity.Name + ", type: " + activity.Type + ", Distance: " + strconv.FormatFloat(float64(metersToMiles(activity.Distance)), 'f', 3, 32))
			// Was this activity today?
			if now.Year() == activity.StartDateLocal.Year() && now.YearDay() == activity.StartDateLocal.YearDay() {
				if activity.Type == "Run" {
					// Truly determining if this is truly a run is more difficult... gotta look for names in titles
					if strings.Contains(activity.Name, "run") || strings.Contains(activity.Name, "Run") {
						currentReport.Day.RunMiles += metersToMiles(activity.Distance)
					} else {
						currentReport.Day.LiftMiles += metersToMiles(activity.Distance)
					}
				} else if activity.Type == "Hike" {
					currentReport.Day.HikeMiles += metersToMiles(activity.Distance)
				}
			}
			if activity.Type == "Run" {
				// Truly determining if this is truly a run is more difficult... gotta look for names in titles
				if strings.Contains(activity.Name, "run") || strings.Contains(activity.Name, "Run") {
					currentReport.YearToDate.RunMiles += metersToMiles(activity.Distance)
				} else {
					currentReport.YearToDate.LiftMiles += metersToMiles(activity.Distance)
				}
			} else if activity.Type == "Hike" {
				currentReport.YearToDate.HikeMiles += metersToMiles(activity.Distance)
			}
		}
		reports = append(reports, currentReport)
	}
	return reports, nil
}

func DoDailyReport() {
	// get Strava users from config
	users, err := ReadUserCredentials()
	if err != nil {
		fmt.Println("Failed to read users in from local credentials file. Error: " + err.Error())
		return
	}
	stravaAthleteReports, err := GetStravaReport(users)
	if err != nil {
		fmt.Println("Failed to Create daily report. Error: " + err.Error())
		return
	}

	// Get data from google sheets
	liftingReports, err := sheets.GetSheetData(config.GoogleSheetsID, config.GoogleCloudCredentialsFilePath, config.GoogleCloudSavedTokenPath, authCodeInputUrl)
	if err != nil {
		fmt.Println("Failed to retrieve lifting-miles from google sheet. Error: " + err.Error())
		return
	}

	// Aggregate data
	for i, report := range stravaAthleteReports {
		if value, ok := liftingReports[report.AthleteFirstName]; ok {
			stravaAthleteReports[i].YearToDate.LiftMiles += value

			// Dang, until we have dates, we cant calculate daily lift miles :(
		}

	}

	// At this point we have our report, we need to reach out to slack
	fmt.Println("Athlete Report: ")
	fmt.Println(stravaAthleteReports)
}
