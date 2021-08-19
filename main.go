package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/costexplorer"
)

var (
	accessToken string
	accessPoint *url.URL
	timeh       *TimeHelper
)

type TimeHelper struct {
	Location *time.Location
	Now      time.Time
}

func init() {
	if accessToken = os.Getenv("AWS_COST_LINE_NOTIFY_TOKEN"); accessToken == "" {
		log.Fatalf("AWS_COST_LINE_NOTIFY_TOKEN is not defined")
	}
	var err error
	accessPoint, err = url.ParseRequestURI("https://notify-api.line.me/api/notify")
	if err != nil {
		log.Fatal(err)
	}
	timeh, err = newTimeHelper("Asia/Tokyo")
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	now := timeh.Now.Format("2006-01-02")
	cost, err := getCostPeriod("MONTHLY", timeh.GetFirstOfMonth(), now)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(cost.ResultsByTime[0].Total)
	cost, err = getCostPeriod("DAILY", timeh.GetYesterday(), now)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(cost.ResultsByTime[0].Total)
	err = notify(cost.String())
	if err != nil {
		log.Fatal(err)
	}
}

func notify(message string) error {
	client := &http.Client{}
	form := url.Values{}
	form.Add("message", message)
	body := strings.NewReader(form.Encode())
	req, err := http.NewRequest("POST", accessPoint.String(), body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	_, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func getCostPeriod(granularity string, start string, end string) (res *costexplorer.GetCostAndUsageOutput, err error) {
	svc := costexplorer.New(session.Must(session.NewSession()))
	metric := "UnblendedCost"
	timePeriod := costexplorer.DateInterval{
		Start: aws.String(start),
		End:   aws.String(end),
	}
	input := costexplorer.GetCostAndUsageInput{}
	input.Granularity = aws.String(granularity)
	input.Metrics = []*string{&metric}
	input.TimePeriod = &timePeriod
	res, err = svc.GetCostAndUsage(&input)
	return res, err
}

func newTimeHelper(locationName string) (*TimeHelper, error) {
	location, err := time.LoadLocation(locationName)
	return &TimeHelper{
		Location: location,
		Now:      time.Now().In(location),
	}, err
}

func (timeh TimeHelper) GetFirstOfMonth() string {
	return time.Date(timeh.Now.Year(), timeh.Now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

func (timeh TimeHelper) GetLastOfMonth() string {
	return time.Date(timeh.Now.Year(), timeh.Now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1).Format("2006-01-02")
}

func (timeh TimeHelper) GetYesterday() string {
	return timeh.Now.AddDate(0, 0, -1).Format("2006-01-02")
}
