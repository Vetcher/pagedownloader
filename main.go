package main

import (
    "encoding/xml"
    "encoding/json"
    "net/http"
    "net/url"
    "io/ioutil"
    "time"
    "os"
    "path"
    logger "github.com/jbrodriguez/mlog"
	"errors"
	"github.com/gosuri/uiprogress"
	"sync"
	"github.com/vetcher/pagedownloader/handlers"
	"strconv"
)

const (
    default_threads int = 0
    default_delay int = 5
    default_logmode int = 0
    default_logswitch bool = false
)

const SETTINGS_FILE_DEFAULT_STRING string = "" +
		"{\n" +
		"\"multi_thread\": 0,\n" +
		"\"delay\": 5,\n" +
		"\"logmode\": 0,\n" +
		"\"logswitch\": false\n" +
		"}"
const URL_FILE_DEFAULT_STRING string = "" +
		"{\n" +
		"\"sitemaps\": [],\n" +
		"\"pages\": []\n" +
		"}"

type SettingsJSON struct {
	Multi_thread int `json:"multi_thread"`
	Delay        int `json:"delay"`
	LogMode      int `json:"logmode"`
	LogSwitch    bool `json:"logswitch"`
}

// return vars about init setups
// 1. multi_thread: 0 if should use one thread, else 1
// 2. delay: delay in seconds between requests
// 3. logmode: mode of logging (0/1/2)
// 4. logswitch: true for activate logger
func InitSettings() (threads int, delay int, logmode int, logswitch bool)  {
    // open settings
    settingsfile, err := os.Open("settings.cfg")
    threads, delay, logmode, logswitch = default_threads, default_delay, default_logmode, default_logswitch

	// Always print settings
	defer logger.Info("multi_thread: %d, delay: %d, logmode: %d, logswitch: %t", threads, delay, logmode, logswitch)

    if err != nil {
      settingsfile, err = os.Create("settings.cfg")
      if err != nil {
          logger.Fatal("No 'setting.cfg' file. Can not create it!. Using default settings.")
	      return
      } else {
          logger.Warning("No 'setting.cfg' file. It was created. Using default settings.")
          settingsfile.Write([]byte(SETTINGS_FILE_DEFAULT_STRING)) // default settings file
          settingsfile.Close()

          return
      }
    }
    jtext, err := ioutil.ReadAll(settingsfile)
    if err != nil {
        logger.Warning("Read error. Using default settings.")
        return
    }
    settingsfile.Close()
    var s SettingsJSON
    err = json.Unmarshal(jtext, &s)
    if err != nil {
        logger.Warning("Wrong settings file format. Look JSON spec. Using default settings.")
	    logger.Warning(err.Error())
        return
    }
    // set variables
    if s.Multi_thread != 0 {
        threads = 1
    } else {
        threads = 0
    }
    if s.Delay <= 0 {
        delay = 20
    } else {
        delay = s.Delay
    }
    logmode = s.LogMode
    logswitch = s.LogSwitch

    return
}

// split url to path and filename
func parceurl(_url string) (string, string, error)  { // directory, filename
    urlstruct, err := url.Parse(_url)
    if err != nil {
        logger.Warning("\tCan't parse url: " + err.Error())
        return "./", ";errname", err
    }
    return "./data/" + urlstruct.Host + path.Dir(urlstruct.Path), path.Base(urlstruct.Path), nil
}

// TODO: here should be requests to DB
// Check database for file
func ShouldLinkBeDownloaded(url string) (bool)  {
    _path, _name, err := parceurl(url) // make filename 'name' and directory for file 'dir'
    if _name == ";errname" || err != nil {
        return false
    }

    // save current dir
    main_dir, err := os.Getwd()
    if err != nil {
        return false
    }
	defer os.Chdir(main_dir)
    err = os.Chdir(_path)
    if err != nil {
        return true
    }
    _, err = os.Open(_name)
    if err != nil {
        return true
    }
    return false
}

type UrlsJSON struct {
    Sitemaps []string `json:"sitemaps"`
    Pages    []string `json:"pages"`
}

// Open file `urls.cfg` and return []byte representation of it
func GetDataFromUrlsFile() ([]byte, error) {
	const urls_config = "urls.cfg"
	urlsfile, err := os.Open(urls_config)
	if err != nil {
		urlsfile, err = os.Create(urls_config)
		defer urlsfile.Close()
		if err != nil {
			logger.Warning("No %s file. Can not create it!.", urls_config)
		} else {
			logger.Warning("No %s file. It was created.", urls_config)
			var t UrlsJSON
			f, err := json.Marshal(t)
			if err != nil {
				logger.Error(err)
				panic(err)
			}
			urlsfile.Write(f)
			urlsfile.Close()
		}
		return nil, errors.New("Can't open " + urls_config)
	}
	defer urlsfile.Close()
	jtext, err := ioutil.ReadAll(urlsfile)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	return jtext, nil
}

// create queue with urls
// parce 'urls.cfg' for default urls
// list of sitemaps, list of urls
func MakeFirstList() ([]string, []string) {
	jtext, err := GetDataFromUrlsFile()
    var s UrlsJSON
    err = json.Unmarshal(jtext, &s)
    if err != nil {
        logger.Warning("Wrong settings file format. Look JSON spec.")
        logger.Warning(err.Error())
        return nil, nil
    }
	sitemap := s.Sitemaps
    if len(sitemap) == 0 {
        sitemap = nil
        logger.Info("No sitemaps in \"urls.cfg\"")
    }
	pages := s.Pages
    if len(pages) == 0 {
	    pages = nil
        logger.Info("No direct urls in \"urls.cfg\"")
    }
    return sitemap, pages
}

type XMLSTRUCT struct {
    XMLName xml.Name
    Urls []string `xml:"url>loc"`
}

func GetAndParseXML(_xml_url string) (queue []string, err error) {

    // Get .xml file (sitemap) from server
    xmlfile, err := http.Get(_xml_url)
    if err != nil {
        logger.Warning("Get \"" + _xml_url + "\": " + err.Error())
        return
    }
    logger.Trace("Get \"" + _xml_url + "\": OK")
    xmltext, err := ioutil.ReadAll(xmlfile.Body)

    // tokenize .xml file
    var xmldoc XMLSTRUCT
    err = xml.Unmarshal([]byte(xmltext), &xmldoc)
    if err != nil {
        logger.Warning("Unmarshal \"" + _xml_url + "\" failed: " + err.Error())
        return
    }
    logger.Trace("Unmarshal \"" + _xml_url + "\": OK ")

    for _, link := range xmldoc.Urls {
        if ShouldLinkBeDownloaded(link) {
            queue = append(queue, link)
        } else {
	        logger.Trace(link + " already downloaded")
        }
    }
    logger.Trace(_xml_url + " OK")
    return
}

// Concatenate all lists with links to map with queues
func CreateQueues() (queuesOfUrls map[string][]string, count int) {
	queuesOfUrls = make(map[string][]string)
	// Init queue
	list_with_sitemaps, list_with_urls := MakeFirstList() // open urls.cfg for targets
	if list_with_urls == nil && list_with_sitemaps == nil {
		logger.Info("Nothing to download")
		return
	}

	if list_with_sitemaps != nil {
		for _, elem := range list_with_sitemaps{
			urlstruct, err := url.Parse(elem)
			if err != nil {
				logger.Warning("\tCan't parse url: " + err.Error())
				continue
			}
			r, err := GetAndParseXML(elem)
			if err != nil {
				logger.Warning("Can't download %s", elem)
				continue
			}
			if len(r) > 0 {
				count += len(r)
				queuesOfUrls[urlstruct.Host] = append(queuesOfUrls[urlstruct.Host], r...) // `...` is a magic for concat 2 slices
			}
		}
	}
	if list_with_urls != nil {
		for _, elem := range list_with_urls {
			urlstruct, err := url.Parse(elem)
			if err != nil {
				logger.Warning("\tCan't parse url: " + err.Error())

			} else {
				queuesOfUrls[urlstruct.Host] = append(queuesOfUrls[urlstruct.Host], elem)
			}
		}
		count += len(list_with_urls)
	}
	return
}

var AllHandlers map[string]handlers.HostHandlerInterface

func init() {
	AllHandlers = make(map[string]handlers.HostHandlerInterface)
	AllHandlers["ria.ru"] = &handlers.RiaHandler{}
}

func main()  {
	// Init setting and logger
	log_file_name := "log/" + time.Now().Format("02-01-06.15_04_05") + ".log"
	logger.Start(logger.LevelTrace, log_file_name)
    defer logger.Info("LAST LOG MESSAGE")
	defer logger.Stop()
	logger.Info("FIRST LOG MESSAGE")
    _, delay, _, _ := InitSettings() // open settings.cfg for settings

	queueOfUrls, urls_count := CreateQueues()
	host_count := len(queueOfUrls)

	if host_count == 0 {
		logger.Warning("Empty")
		return
	}
    dur := time.Duration(urls_count * delay * 2 / host_count) * time.Second
    logger.Trace("Queue: OK ")
    logger.Info("---------------------------------------------")
    logger.Info("In queue %d links.\n", urls_count)
    logger.Info("Time remaining ~ %v\n", dur)
    logger.Info("---------------------------------------------")

	uiprogress.Fill = '#'
	uiprogress.Empty = ' '
	uiprogress.Start()

	var wg sync.WaitGroup
    logger.Info("Start download:")
	for host, queue := range queueOfUrls {
		hostHandler, is_exist := AllHandlers[host]
		if is_exist {

			bar := uiprogress.AddBar(len(queue)).PrependFunc(func(b *uiprogress.Bar) string {
				return host
			}).PrependFunc(func(b *uiprogress.Bar) string {
				return strconv.Itoa(b.Current()) + "/" + strconv.Itoa(b.Total)
			}).PrependElapsed().AppendCompleted()

			wg.Add(1)
			go func() {
				defer wg.Done()
				hostHandler.Init(host, queue, time.Duration(delay) * time.Second)
				hostHandler.Start(bar)
			}()
		} else {
			logger.Warning("Handler for %s not found", host)
		}
	}
	wg.Wait()
}
