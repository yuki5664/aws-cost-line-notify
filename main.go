package main

import (
	"bytes"
	"html/template"
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

type Tmpl struct {
	Start string
	End   string
	Cost  string
	Unit  string
}

const tmplStr = `

{{.Start}} - {{.End}}
{{.Cost}} {{ .Unit }}
`

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
	cost, err := getCostDate()
	if err != nil {
		log.Fatal(err)
	}
	msg, err := getCostMessage(cost)
	if err != nil {
		log.Fatal(err)
	}
	cost, err = getCostMonth()
	if err != nil {
		log.Fatal(err)
	}
	nmsg, err := getCostMessage(cost)
	if err != nil {
		log.Fatal(err)
	}
	err = notify(msg + nmsg)
	if err != nil {
		log.Fatal(err)
	}
}

func getCostMessage(cost *costexplorer.GetCostAndUsageOutput) (string, error) {
	var tmplArg Tmpl
	var tmplWriter bytes.Buffer
	data := cost.ResultsByTime[0]
	tmplArg.Start = *data.TimePeriod.Start
	tmplArg.End = *data.TimePeriod.End
	tmplArg.Cost = *data.Total["UnblendedCost"].Amount
	tmplArg.Unit = *data.Total["UnblendedCost"].Unit
	tmpl := template.Must(template.New("cost-notify").Parse(tmplStr))
	if err := tmpl.Execute(&tmplWriter, tmplArg); err != nil {
		return "", err
	}
	return tmplWriter.String(), nil
}

func getCostDate() (*costexplorer.GetCostAndUsageOutput, error) {
	return getCostPeriod("DAILY", timeh.GetYesterday(), timeh.Now.Format("2006-01-02"))
}

func getCostMonth() (*costexplorer.GetCostAndUsageOutput, error) {
	return getCostPeriod("MONTHLY", timeh.GetFirstOfMonth(), timeh.Now.Format("2006-01-02"))
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
	return err
}

func getCostPeriod(granularity string, start string, end string) (*costexplorer.GetCostAndUsageOutput, error) {
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
	return svc.GetCostAndUsage(&input)
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
