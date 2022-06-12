package sheets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var savedTokenPath string
var savedConfig *oauth2.Config
var initialized bool = false

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config, tokenFilePath, authCodeInputUrl string) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tok, err := tokenFromFile(tokenFilePath)
	if err != nil {
		displayAuthInstructions(config, authCodeInputUrl)
		return nil
	}
	fmt.Println("token expiry: " + tok.Expiry.String())
	fmt.Println("refresh token: " + tok.RefreshToken)
	tokenSource := config.TokenSource(oauth2.NoContext, tok)
	newToken, err := tokenSource.Token()
	if err != nil {
		fmt.Println("Failed to get new token? Error: " + err.Error())
		return nil
	}

	fmt.Println(newToken.Expiry.String())
	fmt.Println(newToken.RefreshToken)
	//client := oauth2.NewClient(oauth2.NoContext, tokenSource)
	//savedToken, err = tokenSource.Token()
	// From the docs
	// Client returns an HTTP client using the provided token. The token will auto-refresh as necessary.
	// The underlying HTTP transport will be obtained using the provided context. The returned client and its Transport should not be modified.
	client := config.Client(oauth2.NoContext, tok)

	//config.TokenSource().Token

	// Save off the (potentially refreshed) token
	saveToken(tokenFilePath, tok)
	return client
}

// Request a token from the web, then returns the retrieved token.
func displayAuthInstructions(config *oauth2.Config, authCodeInputUrl string) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("\nGo to the following link in your browser and authorize API access  "+
		"authorization code: \n%v\n\n", authURL)
	// save off config so it can be accessed during later call from web handler
	savedConfig = config
}

func SetAuthCodeRetrievedFromWeb(authCode string) error {
	tok, err := savedConfig.Exchange(context.TODO(), authCode, oauth2.AccessTypeOffline)
	if err != nil {
		fmt.Println("Unable to retrieve token using auth code provided: " + err.Error())
		return err
	}
	err = saveToken(savedTokenPath, tok)
	if err != nil {
		fmt.Println("Unable to save access token. Error: " + err.Error())
	}
	return nil
}

// Retrieves a token from a local file.
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

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving oauth2 token file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(token)
	if err != nil {
		return err
	}
	initialized = true
	return nil
}

func Initialize(credentialsFilePath, tokenPath, authCodeInputUrl string) error {
	savedTokenPath = tokenPath
	b, err := ioutil.ReadFile(credentialsFilePath)
	if err != nil {
		fmt.Println("Unable to read client credentials json file: %v", err)
		return errors.New("Unable to read client credentials json file " + err.Error())
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets.readonly")
	if err != nil {
		fmt.Println("Unable to parse client secret file to config: %v", err)
		return errors.New("Unable to parse client secret file to config: " + err.Error())
	}

	client := getClient(config, tokenPath, authCodeInputUrl)

	if client == nil {
		fmt.Println("Initialization incomplete until the authorization code is provided via the url: " + authCodeInputUrl)
		initialized = false
	} else {
		fmt.Println("Initialization successful!")
		initialized = true
	}

	return nil
}

type LiftSession struct {
	Date           time.Time
	MinuteDuration int
	MileConversion float32
}

type SheetAthlete struct {
	Name          string
	StartRowIndex int
	StopRowIndex  int
}

func parseAthleteColumns(sheetsData sheets.ValueRange, startIndex, stopIndex int) ([]LiftSession, error) {
	liftSessions := []LiftSession{}
	for _, row := range sheetsData.Values {
		date := row[startIndex].(string)
		timeMinutes, _ := strconv.Atoi(row[startIndex+1].(string))
		miles, _ := strconv.ParseFloat(row[startIndex+2].(string), 32)
		if date == "" {
			continue
		}
		dateTime, err := time.Parse("1/2/2006", date)
		if err != nil {
			return liftSessions, err
		}
		liftSessions = append(liftSessions, LiftSession{
			Date:           dateTime,
			MinuteDuration: timeMinutes,
			MileConversion: float32(miles),
		})
	}
	return liftSessions, nil
}

func GetAthleteLiftData(spreadsheetId, credentialsFilePath, tokenPath, authCodeInputUrl string) (map[string][]LiftSession, error) {

	athletesInSheet := []SheetAthlete{
		SheetAthlete{"Leben", 0, 3},   // leben columns A - D (0 - 3)
		SheetAthlete{"Ben", 5, 8},     // Ben columns F - I (5 - 8)
		SheetAthlete{"Peter", 10, 13}, // Peter columns K - N (10 - 13)
	}

	athleteLifts := map[string][]LiftSession{}
	if !initialized {
		return athleteLifts, errors.New("Sheets not successfully initialized yet")
	}
	ctx := context.Background()
	b, err := ioutil.ReadFile(credentialsFilePath)
	if err != nil {
		fmt.Println("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets.readonly")
	if err != nil {
		fmt.Println("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config, tokenPath, authCodeInputUrl)

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		fmt.Println("Unable to retrieve Sheets client: %v", err)
	}

	// We just grab 300 rows and hope that is enough
	readRange := "Sheet1!A3:N300"

	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		return athleteLifts, err
	}

	if len(resp.Values) == 0 {
		fmt.Println("No data found.")
		return athleteLifts, nil
	}

	for _, athlete := range athletesInSheet {
		athleteLifts[athlete.Name], err = parseAthleteColumns(*resp, athlete.StartRowIndex, athlete.StopRowIndex)
		if err != nil {
			return athleteLifts, err
		}
		fmt.Println(athlete.Name + " total number of sessions: " + strconv.Itoa(len(athleteLifts[athlete.Name])))
	}
	return athleteLifts, nil
	// fmt.Println("Date, Time, Miles")
	// for _, row := range resp.Values {
	// 	// Leben A - D (0 - 3)
	// 	// Ben F - I (5 - 8)
	// 	// Peter K - N (10 - 13)
	// 	// fmt.Printf("%s, %d, %s\n", row[0], row[1])
	// 	parsedValue, _ := strconv.ParseFloat(row[1].(string), 32)
	// 	athletes[row[0].(string)] = float32(parsedValue)
	// }

	// Read the totals and make sure our code matches... just a check, mostly to verify ben's code isn't crazy
	// readRange := "Sheet1!Q2:R4"
	// resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	// if err != nil {
	// 	return athleteLifts, err
	// }

	// if len(resp.Values) == 0 {
	// 	fmt.Println("No data found.")
	// 	return athleteLifts, nil
	// } else {
	// 	fmt.Println("Name, Lifting miles")
	// 	for _, row := range resp.Values {
	// 		// Print columns A and E, which correspond to indices 0 and 4.
	// 		fmt.Printf("%s, %s\n", row[0], row[1])
	// 		parsedValue, _ := strconv.ParseFloat(row[1].(string), 32)
	// 		athletes[row[0].(string)] = float32(parsedValue)
	// 	}
	// }
	// return athletes, nil
}
