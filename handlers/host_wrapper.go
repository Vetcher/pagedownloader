package handlers

import (
	"net/http"
	"io/ioutil"
	logger "github.com/jbrodriguez/mlog"
	"time"
	"sync"
	"github.com/gosuri/uiprogress"
)

// If you want to add some urls to Queue after call `Start()`,
// just call `QueueMx.Lock()` before and `QueueMx.Unlock()` after adding
type HostHandler struct {
	Delay time.Duration
	Queue []string
	HostName string
	QueueMx sync.Mutex
}

type HostHandlerInterface interface {
	Start(bar *uiprogress.Bar)
	HandleResponse([]byte) error
	Init(string, []string, time.Duration)
}

func (hh *HostHandler) Init(host string, queue []string, delay time.Duration) {
	hh.HostName = host
	hh.Queue = queue
	hh.Delay = delay
}

// Checks HostHandler fields for correct working and starts cycle
func (hh *HostHandler) Start(bar *uiprogress.Bar) {
	if hh.HostName == "" {
		logger.Warning(" `HostName` is empty, please specify it")
		return
	} else if len(hh.Queue) == 0 {
		logger.Warning(hh.HostName + " `Queue` is empty, please add some urls")
		return
	}
	hh.startPipeline(bar)
}

// This function iterates over Queue and calls handlers
func (hh *HostHandler) startPipeline(bar *uiprogress.Bar) {
	queuetimer := time.NewTicker(hh.Delay)
	for len(hh.Queue) != 0 {
		select {
		case <- queuetimer.C:
			bar.Incr()
			// Get new url from queue
			new_url := hh.popUrlFromQueue()

			// Connect over HTTP to `new_url`
			answer, err := hh.newRequest(new_url)
			if err != nil {
				logger.Warning(err.Error())
				continue
			}

			// Give response's body to handler
			if err := hh.HandleResponse(answer); err != nil {
				logger.Warning(err.Error())
				continue
			}
		}
	}
}

// This function should be reimplemented for each host
func (hh *HostHandler) HandleResponse([]byte) (error) {
	return nil
}

// Get head (first) element and remove it from queue
func (hh *HostHandler) popUrlFromQueue() (_url string) {
	hh.QueueMx.Lock()
	_url = hh.Queue[0]
	hh.Queue = hh.Queue[1:]
	hh.QueueMx.Unlock()
	return
}

// create new Get request, returns []byte body representation and status (done/not) of request
// if err != nil means request moves bad
func (hh *HostHandler) newRequest(_url string) (answer []byte, err error)  {

	/*dir, name, err := parceurl(_url) // make filename 'name' and directory for file 'dir'
	if name == ";errname" {
		logger.Warning("parceurl error: " + err.Error())
		return
	}

	urlstruct, err := url.Parse(_url) // need host name
	if err != nil {
		logger.Warning("Can't parse url: " + err.Error())
		return
	}*/

	resp, err := http.Get(_url)
	if err != nil {
		logger.Warning("Get error: " + err.Error())
		return
	}

	answer, err = ioutil.ReadAll(resp.Body) // convert to []byte
	if err != nil {
		logger.Warning("Can't read from request: " + err.Error())
		return
	}

	return
}
