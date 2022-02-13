package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bclouser/miles-challenge/sheets"
	"github.com/bclouser/miles-challenge/slack"
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

func (a *AthleteCounts) Total() float32 {
	return a.RunMiles + a.HikeMiles + a.LiftMiles
}

type UserReport struct {
	AthleteID        int           `json:"athlete_id"`
	AthleteFirstName string        `json:"athlete_firstname"`
	YearToDate       AthleteCounts `json:"year_to_date"`
	Day              AthleteCounts `json:"day"`
}

// Does user1 have more total challenge miles than user2
func greater(user1, user2 AthleteCounts) bool {
	return user1.Total() > user2.Total()
}

// Does user1 have lessThanOrEqual total challenge miles than user2
func lessThan(user1, user2 AthleteCounts) bool {
	return user1.Total() < user2.Total()
}

// Does user1 have more total challenge miles than user2
func equal(user1, user2 AthleteCounts) bool {
	return user1.Total() == user2.Total()
}

func floatStr(in float32) string {
	return strconv.FormatFloat(float64(in), 'f', 2, 32)
}

func numberToPlaceStr(in int) string {
	switch in {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	default:
		/// Yeah yeah yeah, 21st, 22nd, 23rd. We should probably modulo but idgaf
		return strconv.Itoa(in) + "th"
	}
	return ""
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
	formattedReport := GenerateFormattedReport()
	// Send report to slack
	report := "   :man-running:  *The Daily Report!* :scroll:\n\n" + formattedReport
	slack.SendChannelMessage(config.SlackChannelHookUrl, report)
}

func GenerateFormattedReport() string {
	athleteReports := GenerateReport()
	formattedReport := ""
	for i, athlete := range athleteReports {
		userReport := "*    " + numberToPlaceStr(i+1) + "*    " + athlete.AthleteFirstName + "\n" +
			"    Miles Run Today:     " + floatStr(athlete.Day.RunMiles) + "\n" +
			"    Miles Hiked Today:   " + floatStr(athlete.Day.HikeMiles) + "\n" +
			"    Miles* Lifted Today: " + floatStr(athlete.Day.LiftMiles) + "\n" +
			"    ---   \n" +
			"    Miles Run this Year:     " + floatStr(athlete.YearToDate.RunMiles) + "\n" +
			"    Miles Hiked this Year:   " + floatStr(athlete.YearToDate.HikeMiles) + "\n" +
			"    Miles* Lifted this Year: " + floatStr(athlete.YearToDate.LiftMiles) + "\n" +
			"    Total Challenge Miles: *" + floatStr(athlete.YearToDate.Total()) + "*\n"

		// Add trailing line only if there is another user
		if i+1 != len(athleteReports) {
			userReport += "    -------------------------- \n"
		}

		formattedReport += userReport
	}
	return formattedReport
}

func GenerateReport() []UserReport {
	athleteReports := []UserReport{}
	// get Strava users from config
	users, err := ReadUserCredentials()
	if err != nil {
		fmt.Println("Failed to read users in from local credentials file. Error: " + err.Error())
		return athleteReports
	}
	athleteReports, err = GetStravaReport(users)
	if err != nil {
		fmt.Println("Failed to Create daily report. Error: " + err.Error())
		return athleteReports
	}

	// Get data from google sheets
	liftingReports, err := sheets.GetSheetData(config.GoogleSheetsID, config.GoogleCloudCredentialsFilePath, config.GoogleCloudSavedTokenPath, authCodeInputUrl)
	if err != nil {
		fmt.Println("Failed to retrieve lifting-miles from google sheet. Error: " + err.Error())
		return athleteReports
	}

	// Aggregate and sort data
	for i, report := range athleteReports {
		if value, ok := liftingReports[report.AthleteFirstName]; ok {
			athleteReports[i].YearToDate.LiftMiles += value

			// Dang, until we have dates, we cant calculate daily lift miles :(

		}
	}
	// Sort with greater so that  the first element is "first place"
	sort.SliceStable(athleteReports, func(i, j int) bool { return greater(athleteReports[i].YearToDate, athleteReports[j].YearToDate) })
	return athleteReports
}
