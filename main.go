package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type sToken struct {
	Token string `json:"token"`
}

type Response struct {
	Data SearchJobCard `json:"data"`
}

type SearchJobCard struct {
	JobCard struct {
		NextToken any    `json:"nextToken"`
		Cards     []Card `json:"jobCards"`
		TypeName  string `json:"__typename"`
	} `json:"searchJobCardsByLocation"`
}

type Card struct {
	JobId    string `json:"jobId"`
	JobTitle string `json:"jobTitle"`
	City     string `json:"city"`
	State    string `json:"state"`
}

const (
	DATE  = "AWS_JOB_DATE"
	TOKEN = "AWS_JOB_SESSION_TOKEN"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	today := time.Now().Format(time.DateOnly)
	envDate := os.Getenv("AWS_JOB_DATE")

	if envDate != today {
		fmt.Printf(`Updating DATE variable: "%s" ~ "%s"`, envDate, today)
		os.Setenv(DATE, today)
		os.Setenv(TOKEN, getSessionToken().Token)
	}

	getJobCards()

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

func getJobCards() {

	fmt.Println("Getting jobs...")

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
        "geoQueryClause": {
          "lat": 39.2408565,
          "lng": -76.6799001,
          "unit": "mi",
          "distance": 30
        },
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
			links += `<li><a href="https://hiring.amazon.com/app#/jobDetail?jobId=` + card.JobId + `">` + card.JobTitle + ` (` + card.City + `, ` + card.State + `)` + `</a></li>`
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
	to := mail.NewEmail("Sebastian Villegas", "sebasvn2340@gmail.com")
	to1 := &mail.Email{Name: "Sebastian Villegas", Address: "sebasvn2340@gmail.com"}
	to2 := &mail.Email{Name: "Natalia Betancur", Address: "nbetancur1196@gmail.com"}
	personalization := new(mail.Personalization)
	personalization.To = append(personalization.To, to1, to2)
	plainTextContent := "Jobs"
	htmlContent := "<ul>" + links + "</ul>"
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
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
