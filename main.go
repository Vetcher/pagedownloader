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
	"container/list"
)

const (
    default_threads int = 0
    default_delay int = 5
    default_logmode int = 0
    default_logswitch bool = false
)

const _setfile_defstr string = "{\n\"multi_thread\": 0,\n\"delay\": 5,\n\"logmode\": 0,\n\"logswitch\": false\n}"
const _urlfile_defstr string = "{\n\"Lists\": [],\n\"Urls\": []\n}"

type SettingsJSON struct {
    Multi_thread int
    Delay int
    LogMode int
    LogSwitch bool
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
          settingsfile.Write([]byte(_setfile_defstr)) // default settings file
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
func ShouldItBeDownloaded(url string) (bool)  {
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
    Lists []string
    Pages []string
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
			urlsfile.Write([]byte(_urlfile_defstr))
			urlsfile.Close()
		}
		return nil, error("Can't open" + urls_config)
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

    var s UrlsJSON
    err = json.Unmarshal(jtext, &s)
    if err != nil {
        panic(err.Error())
        logger.Alwaysln("Wrong settings file format. Look JSON spec.")
        logger.Alwaysln(err.Error())
        return nil, nil
    }
    sitemap := list.New()
    for _, elem := range s.Lists {
        sitemap.PushBack(elem.Url)
    }
    if sitemap.Len() == 0 {
        sitemap = nil
        logger.Alwaysln("No sitemaps in \"urls.cfg\"")
    }
    listofurls := list.New()
    for _, elem := range s.Pages {
        listofurls.PushBack(elem.Url)
    }
    if listofurls.Len() == 0 {
        listofurls = nil
        logger.Alwaysln("No direct urls in \"urls.cfg\"")
    }
    return sitemap, listofurls
}

type XMLSTRUCT struct {
    XMLName xml.Name
    Urls []string `xml:"url>loc"`
}

func GetAndParseXML(_xml_url string, _queue *list.List) (int) {
    if _queue == nil {
        logger.Alwaysln("Error: queue is nil")
        return 0
    }
    // Get .xml file from server
    xmlfile, err := http.Get(_xml_url)
    if err != nil {
        logger.Alwaysln("Get \"" + _xml_url + "\": " + err.Error())
        return 0
    }
    logger.Moreln("Get \"" + _xml_url + "\": OK")
    xmltext, err := ioutil.ReadAll(xmlfile.Body)

    // tokenize .xml file
    var xmldoc XMLSTRUCT
    err = xml.Unmarshal([]byte(xmltext), &xmldoc)
    if err != nil {
        logger.Alwaysln("Unmarshal \"" + _xml_url + "\" failed: " + err.Error())
        return 0
    }
    logger.Moreln("Unmarshal \"" + _xml_url + "\": OK ")

    count := len(xmldoc.Urls)
    for index := 0; index < count; index++ {
        if !ShouldItBeDownloaded(xmldoc.Urls[index]) {
            logger.Debugln(xmldoc.Urls[index] + " already downloaded")
            continue
        }
        _queue.PushBack(xmldoc.Urls[index])
    }
    logger.Alwaysln(_xml_url + " OK")
    return count
}

func main()  {
	// Init setting and logger
    defer logger.Info("LAST LOG MESSAGE")
	defer logger.Stop()
    _, delay, logmode, _ := InitSettings() // open settings.cfg for settings
	log_file_name := time.Now().Format("02-01-06.15_04_05") + ".log"
	logger.Start(logger.LogLevel(logmode), log_file_name)

	// Init queue
    list_with_sitemaps, list_with_urls := MakeFirstList() // open urls.cfg for targets
    if list_with_urls == nil && list_with_sitemaps == nil {
        logger.Info("Nothing to download")
        return
    }

    queueOfUrls := list.New()
    count := 0
    if list_with_sitemaps != nil {
        for elem := list_with_sitemaps.Front(); elem != nil; elem = elem.Next() {
            count += GetAndParseXML(elem.Value.(string), queueOfUrls)
        }
    }
    if list_with_urls != nil {
        for elem := list_with_urls.Front(); elem != nil; elem = elem.Next() {
            queueOfUrls.PushBack(elem.Value.(string))
            count++
        }
    }

    dur := time.Duration(queueOfUrls.Len() * delay) * time.Second
    logger.Trace("Queue: OK ")
    logger.Alwaysln("---------------------------------------------")
    logger.Alwaysf("Founded %d links,\n", count)
    logger.Alwaysf("Already downloaded %d,\n", count - queueOfUrls.Len())
    logger.Alwaysf("In queue %d links.\n", queueOfUrls.Len())
    logger.Alwaysf("Time remaining about %v\n", dur)
    logger.Alwaysln("---------------------------------------------")
    i := 0
    count = 0
    queuetimer := time.NewTicker(time.Second * time.Duration(delay))
    logger.Alwaysln("Start taker:")
	   for {
           select {
            case <- queuetimer.C:
                if queueOfUrls.Front() != nil {
                    temp := queueOfUrls.Front()
                    requesturl := temp.Value.(string)
                    queueOfUrls.Remove(temp)
                    logger.Alwaysf("%d: %s", i, requesturl)
		            boo := NewRequest(temp.Value.(string))
                    if boo {
                        count++
                    }
                    i++
                } else {
                    logger.Alwaysf("Head of queue, %d documents downloaded.", count)
                    return
                }
            }
        }
}
