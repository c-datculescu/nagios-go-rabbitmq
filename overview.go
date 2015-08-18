package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jessevdk/go-flags"
)

type options struct {
	Host     string `short:"h" long:"host" description:"The host of the rabbitmq server being monitored. For clusters please use all the hostnames in a comma separated list." default:"localhost"`
	Port     string `short:"P" long:"port" description:"The port on which the server can be accessed." default:"15672"`
	Username string `short:"u" long:"username" description:"The username used for accessing rabbitmq web api." default:"guest"`
	Password string `short:"p" long:"password" description:"The password for the account used to access the web api." default:"guest"`
	Warning  string `short:"w" long:"warning" default:"10000,10000" description:"Threshold for warnings."`
	Critical string `short:"c" long:"critical" default:"50000,50000" description:"Threshold for critical."`
	Secure   bool   `short:"s" long:"secure" default:"false" description:"Use http or https when accessing the api."`
}

/*
Overview representation from the api
*/
type Overview struct {
	QueueTotals QueueTotals `json:"queue_totals"`
}

/*
QueueTotals represents the queue_totals substructure
*/
type QueueTotals struct {
	MessagesUnack int `json:"messages_unacknowledged"`
	MessagesReady int `json:"messages_ready"`
}

/*
limitMap calculates the limits from a comma separated string
*/
func limitMap(str string) ([]int, error) {
	warningLimits := strings.Split(str, ",")
	warning := []int{}
	if len(warningLimits) != 2 {
		err := errors.New("A list of two integers is required for limits.")
		return nil, err
	}
	for _, value := range warningLimits {
		tmpWarning, err := strconv.Atoi(value)
		if err != nil {
			log.Println(err.Error())
			return nil, err
		}
		warning = append(warning, tmpWarning)
	}

	return warning, nil
}

/*
processHost processes the host and returns the overview from it
*/
func processHost(opt *options, host string) (*Overview, error) {
	prefix := "http"
	if opt.Secure == true {
		prefix = "https"
	}
	uri := prefix + "://" + host + ":" + opt.Port + "/api/overview"
	client := http.Client{}

	request, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	request.SetBasicAuth(opt.Username, opt.Password)
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	over := &Overview{}
	err = json.Unmarshal(body, &over)
	if err != nil {
		return nil, err
	}

	return over, nil
}

/*
processOverview processes the overview thresholds
*/
func processOverview(over *Overview, warning, critical []int) {
	rdy, unack := strconv.Itoa(over.QueueTotals.MessagesReady), strconv.Itoa(over.QueueTotals.MessagesUnack)

	// check errors first
	if over.QueueTotals.MessagesReady >= critical[0] {
		fmt.Println("CRITICAL " + rdy + " messages ready")
	} else if over.QueueTotals.MessagesReady >= warning[0] {
		fmt.Println("WARNING " + rdy + " messages ready")
	} else {
		fmt.Println("OK " + rdy + " messages ready")
	}

	if over.QueueTotals.MessagesUnack >= critical[1] {
		fmt.Println("CRITICAL " + unack + " messages unacknowledged")
	} else if over.QueueTotals.MessagesUnack >= warning[1] {
		fmt.Println("WARNING " + unack + " messages unacknowledged")
	} else {
		fmt.Println("OK " + unack + " messages unacknowledged")
	}

}

func main() {
	opt := &options{}
	_, err := flags.Parse(opt)
	if err != nil {
		return
	}

	warningLimits, err := limitMap(opt.Warning)
	if err != nil {
		log.Println(err.Error())
		return
	}

	criticalLimits, err := limitMap(opt.Critical)
	if err != nil {
		log.Println(err.Error())
		return
	}
	hosts := strings.Split(opt.Host, ",")

	// loop through all hosts and check if we can access the overview page
	for _, value := range hosts {
		over, err := processHost(opt, value)
		if err != nil {
			log.Println(err.Error())
			return
		}
		processOverview(over, warningLimits, criticalLimits)
	}
}
