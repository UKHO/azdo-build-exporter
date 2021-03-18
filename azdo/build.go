package azdo
import (
	"time"
)

type buildResponseEnvelope struct {
	Count int `json:"count"`
	Builds  []Build `json:"value"`
}

type Build struct {
	Id   int    `json:"id"`
	Number string `json:"buildNumber"`
	Status string `json:"status"`
	Result string `json:"result"`
	ReceiveTime time.Time `json:"receiveTime"`
	QueueTime time.Time `json:"queueTime"`
	StartTime time.Time `json:"startTime"`
	FinishTime time.Time `json:"finishTime"`
	Definition Definition `json:"definition"`
}

type Definition struct {
	Id int `json:"id"`
	Name string `json:"name"`
}
