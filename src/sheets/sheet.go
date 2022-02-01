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
	return config.Client(context.Background(), tok)
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
	tok, err := savedConfig.Exchange(context.TODO(), authCode)
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
	fmt.Printf("Saving credential file to: %s\n", path)
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
	fmt.Println("BEN SAYS: credentialsFilepath: " + credentialsFilePath)
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

	fmt.Println("Config looks like")
	fmt.Println(config)

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

func GetSheetData(spreadsheetId, credentialsFilePath, tokenPath, authCodeInputUrl string) (map[string]float32, error) {
	athletes := map[string]float32{}
	if !initialized {
		return athletes, errors.New("Sheets not successfully initialized yet")
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

	// Prints the names and majors of students in a sample spreadsheet:
	// https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit
	readRange := "Sheet1!Q2:R4"
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		return athletes, err
	}

	if len(resp.Values) == 0 {
		fmt.Println("No data found.")
		return athletes, nil
	} else {
		fmt.Println("Name, Lifting miles")
		for _, row := range resp.Values {
			// Print columns A and E, which correspond to indices 0 and 4.
			fmt.Printf("%s, %s\n", row[0], row[1])
			parsedValue, _ := strconv.ParseFloat(row[1].(string), 32)
			athletes[row[0].(string)] = float32(parsedValue)
		}
	}
	return athletes, nil
}
