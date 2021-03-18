package azdo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff"

	log "github.com/sirupsen/logrus"
)

type AzDoClient struct {
	Client            *http.Client
	ApiVersion        string
	Name              string
	Address           string
	DefaultCollection string
	Projects		  []string
	AccessToken       string
}

func (az *AzDoClient) GetProjects() ([]Project, error) {
	log.WithFields(log.Fields{"serverName": az.Name}).Info("Get Projects")

	if len(az.Projects) != 0 {
		log.WithFields(log.Fields{"serverName": az.Name, "config": len(az.Projects)}).Info("Use Config for Projects")
		var projects = []Project{}
		for _, i := range az.Projects {
			var project = Project{Id: i, Name: i}
			projects = append(projects, project)
		}
		return projects, nil
	}

	var url = az.buildURL("_apis/projects?api-version=")

	if len(az.ApiVersion) != 0 {
		url = url + az.ApiVersion
	} else {
		url = url + "6.0"
	}

	req, err := http.NewRequest("GET", url, nil)

	req.SetBasicAuth("", az.AccessToken)

	responseData, err := az.makeRequest(req)

	if err != nil {
		return []Project{}, err
	}

	are := projectResponseEnvelope{}
	err = json.Unmarshal(responseData, &are)
	if err != nil {
		return []Project{}, err
	}

	return are.Projects, nil
}

func (az *AzDoClient) GetBuilds(projectName string) ([]Build, error) {

	log.WithFields(log.Fields{"serverName": az.Name, "project": projectName}).Info("Get Builds")

		var url = az.buildURL(projectName + "/_apis/build/builds?api-version=")
		
		if len(az.ApiVersion) != 0 {
			url = url + az.ApiVersion
		} else {
			url = url + "6.0"
		}

		log.WithFields(log.Fields{"serverName": az.Name, "project": projectName}).Info(url)

		req, err := http.NewRequest("GET", url, nil)

		req.SetBasicAuth("", az.AccessToken)

		responseData, err := az.makeRequest(req)

		if err != nil {
			log.Error(err)
			return []Build{}, err
		}

		are := buildResponseEnvelope{}
		err = json.Unmarshal(responseData, &are)
		if err != nil {
			log.Error(err)
			return []Build{}, err
		}

		var builds = are.Builds

		log.WithFields(log.Fields{"serverName": az.Name, "project": projectName}).Info(builds)
		return builds, nil
}

func (az *AzDoClient) makeRequest(req *http.Request) ([]byte, error) {

	var (
		responseData []byte
		err          error
	)

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 30 * time.Second

	notify := func(err error, ti time.Duration) {
		log.WithFields(log.Fields{"serverName": az.Name, "URL": req.URL, "error": err}).Warning("Retrying HTTP request")
	}

	retry := func() error {
		responseData, err = az.makeHTTPRequest(req)
		return err
	}

	e := backoff.RetryNotify(retry, b, notify)
	if e != nil {
		return []byte{}, e
	}

	return responseData, nil
}

func (az *AzDoClient) makeHTTPRequest(req *http.Request) ([]byte, error) {

	// Send request
	resp, err := az.Client.Do(req)
	if err != nil {
		return []byte{}, fmt.Errorf("Call to %v failed: %v", req.URL, err)
	}
	defer resp.Body.Close()
	log.WithFields(log.Fields{"serverName": az.Name, "URL": req.URL, "StatusCode": resp.StatusCode}).Trace("Made HTTP request")

	// Read body of response
	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("Failed to read body %v", err)
	}

	log.Debug(string(responseData))

	return responseData, nil
}

func (az *AzDoClient) buildURL(url string) string {
	var baseURL string
	if az.DefaultCollection != "" {
		baseURL = az.Address + "/" + az.DefaultCollection
	} else {
		baseURL = az.Address
	}

	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}

	return baseURL + url
}
