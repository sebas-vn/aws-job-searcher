package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"googlemaps.github.io/maps"
)

// constants drawn from environment file
const (
	DATE         = "AWS_JOB_DATE"
	TOKEN        = "AWS_JOB_SESSION_TOKEN"
	GOOGLE_TOKEN = "GOOGLE_API_KEY"
	FILE_PATH    = "FILE_PATH"
	EMAIL        = "EMAIL"
	ZIPCODE      = "ZIPCODE"
)

func main() {

	today := time.Now().Format(time.DateOnly)
	fmt.Printf("\nExecuting script: \"%s\"", today)

	// load environments if there is .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env file found")
	}

	// setting start time
	startTime := time.Now()

	// getting unauthenitcated session token
	os.Setenv(TOKEN, getSessionToken().Token)

	// get saved data from gelocation request in file
	if data, err := os.ReadFile(os.Getenv(FILE_PATH) + "/string.txt"); err == nil {
		fmt.Println("\nReusing saved result:", string(data))
		getJobCards(string(data))
	} else {
		fmt.Println("\nGetting geolocation data...")
		getGeoQuery()
	}

	endTime := time.Now()
	fmt.Printf("Time to run %v", endTime.Sub(startTime))

}

// save GeoQuery API response to file in volume
func saveToFile(jsonString string) {
	fmt.Printf("Saving geolocation response to %s", os.Getenv(FILE_PATH)+"/string.txt")

	// Ensure the directory exists
	os.MkdirAll("/app/data", os.ModePerm)

	// Write to file
	if err := os.WriteFile(os.Getenv(FILE_PATH)+"/string.txt", []byte(jsonString), 0644); err != nil {
		fmt.Println("Error writing file:", err)
	}

	// once saved get jobs
	getJobCards(jsonString)

}

func getSessionToken() *sToken {
	// Log: Retrieving token -- mm/dd/yyyy 00:00:00
	resp, err := http.Get("https://auth.hiring.amazon.com/api/csrf?countryCode=US")
	if err != nil {
		// Error Log: Error retrieving token, notifying admin...
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("\nResponse Status: ", resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var sesToken = &sToken{}
	unmarshalErr := json.Unmarshal(body, sesToken)
	if unmarshalErr != nil {
		// Error Log: Error converting token string to token struct
	}

	return sesToken
}

func getGeoQuery() {
	c, err := maps.NewClient(maps.WithAPIKey(os.Getenv(GOOGLE_TOKEN)))
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}

	r := &maps.GeocodingRequest{
		Address: os.Getenv(ZIPCODE), // change this for environment variable
	}

	geocodingResponse, err := c.Geocode(context.Background(), r)
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}

	geoQueryClause := &GeoQueryClause{
		Lat:      geocodingResponse[0].Geometry.Location.Lat,
		Lng:      geocodingResponse[0].Geometry.Location.Lng,
		Unit:     "mi",
		Distance: 30,
	}

	jsonData, err := json.Marshal(geoQueryClause)
	if err != nil {
		fmt.Println("Error marshaling geoqueryResponse")
		return
	}

	jsonString := string(jsonData)

	// save string into file to check later
	saveToFile(jsonString)
}

func getJobCards(jsonString string) {

	fmt.Println("\nGetting jobs...")

	payload := `{
    "operationName": "searchJobCardsByLocation",
    "query": "query searchJobCardsByLocation($searchJobRequest: SearchJobRequest!) {\n  searchJobCardsByLocation(searchJobRequest: $searchJobRequest) {\n    nextToken\n    jobCards {\n      jobId\n      language\n      dataSource\n      requisitionType\n      jobTitle\n      jobType\n      employmentType\n      city\n      state\n      postalCode\n      locationName\n      totalPayRateMin\n      totalPayRateMax\n      tagLine\n      bannerText\n      image\n      jobPreviewVideo\n      distance\n      featuredJob\n      bonusJob\n      bonusPay\n      scheduleCount\n      currencyCode\n      geoClusterDescription\n      surgePay\n      jobTypeL10N\n      employmentTypeL10N\n      bonusPayL10N\n      surgePayL10N\n      totalPayRateMinL10N\n      totalPayRateMaxL10N\n      distanceL10N\n      monthlyBasePayMin\n      monthlyBasePayMinL10N\n      monthlyBasePayMax\n      monthlyBasePayMaxL10N\n      jobContainerJobMetaL1\n      virtualLocation\n      poolingEnabled\n      __typename\n    }\n    __typename\n  }\n}\n",
    "variables": {
      "searchJobRequest": {
        "locale": "en-US",
        "country": "United States",
        "keyWords": "",
        "equalFilters": [],
        "containFilters": [
          {
            "key": "isPrivateSchedule",
            "val": [
              "false"
            ]
          }
        ],
        "geoQueryClause": ` + jsonString + `,
        "dateFilters": [
          {
            "key": "firstDayOnSite",
            "range": {
              "startDate": "` + time.Now().Format(time.DateOnly) + `"
            }
          }
        ],
        "sorters": [],
        "pageSize": 100,
        "consolidateSchedule": true
      }
    }
  }`

	authHeader := "Bearer Status|unauthenticated|Session|" + os.Getenv(TOKEN)

	newRequest, err := http.NewRequest("POST", "https://e5mquma77feepi2bdn4d6h3mpu.appsync-api.us-east-1.amazonaws.com/graphql", bytes.NewBuffer([]byte(payload)))
	newRequest.Header.Set("Content-Type", "application/json")
	newRequest.Header.Set("Authorization", authHeader)
	if err != nil {
		fmt.Printf("Error in generating new request -- %v", err)
	}

	client := &http.Client{Timeout: time.Second * 5}
	response, err := client.Do(newRequest)
	if err != nil {
		fmt.Printf("There was an error executing the request -- %v", err)
	}

	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Data Read Error -- %v ", err)
	}

	var jobs Response
	err = json.Unmarshal(responseData, &jobs)
	if err != nil {
		// Log Error
		panic(err)
	}

	if len(jobs.Data.JobCard.Cards) > 0 {

		fmt.Printf("Jobs found: %d \n", len(jobs.Data.JobCard.Cards))
		var links string
		for i := 0; i < len(jobs.Data.JobCard.Cards); i++ {
			card := jobs.Data.JobCard.Cards[0]
			links += `<li><a href="https://hiring.amazon.com/app#/jobDetail?jobId=` + card.JobId + `&locale=en-US">` + card.JobTitle + ` (` + card.City + `, ` + card.State + `)` + `</a></li>`
		}

		err = sendEmail(links, jobs.Data)
		if err != nil {
			fmt.Printf("\nError sending email -- %v", err)
		}
	} else {
		fmt.Println("No jobs found.")
	}
}

func sendEmail(links string, cards SearchJobCard) error {

	from := mail.NewEmail("Isladfantasia Server", "isladfantasia.server@gmail.com")
	subject := "NEW AMAZON FULFILLMENT JOBS - " + strconv.Itoa(len(cards.JobCard.Cards))
	personalization := new(mail.Personalization)

	// loop through emails set in environment
	emails := strings.Split(os.Getenv(EMAIL), ",")
	for _, value := range emails {
		to := &mail.Email{Address: strings.TrimSpace(value)}
		personalization.To = append(personalization.To, to)
	}

	plainTextContent := "Jobs"
	htmlContent := "<ul>" + links + "</ul>"
	message := mail.NewSingleEmail(from, subject, personalization.To[0], plainTextContent, htmlContent)
	message.AddPersonalizations(personalization)

	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
	}

	return err
}
