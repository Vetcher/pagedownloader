package main

import (
    "encoding/xml"
    "encoding/json"
    "net/http"
    "net/url"
    "io/ioutil"
    "container/list"
    "time"
    "os"
    "path"
    "./pkg/mylogger"
    "./pkg/cleaner"
)

// variable for handle log
var logger mylogger.Logger

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
// 3. logmode: mode of logging (0/1/2), check mylogger doc for more information
// 4. logswitch: true for activate logger
func InitSettings() (int, int, int, bool)  {
    // open settings
    settingsfile, err := os.Open("settings.cfg")
    threads, delay, logmode, logswitch := default_threads, default_delay, default_logmode, default_logswitch

    if err != nil {
      settingsfile, err = os.Create("settings.cfg")
      if err != nil {
          logger.Alwaysln("No 'setting.cfg' file. Can not create it!. Using default settings.")
      } else {
          logger.Alwaysln("No 'setting.cfg' file. It was created. Using default settings.")
          settingsfile.Write([]byte(_setfile_defstr)) // default settings file
          settingsfile.Close()
          logger.Alwaysf("multi_thread: %d, delay: %d, logmode: %d, logswitch: %t", threads, delay, logmode, logswitch)
          return threads, delay, logmode, logswitch
      }
    }
    jtext, err := ioutil.ReadAll(settingsfile)
    if err != nil {
        logger.Alwaysln("Read error. Using default settings.")
        logger.Alwaysf("multi_thread: %d, delay: %d, logmode: %d, logswitch: %t", threads, delay, logmode, logswitch)
        return threads, delay, logmode, logswitch
    }
    settingsfile.Close()
    var s SettingsJSON
    err = json.Unmarshal(jtext, &s)
    if err != nil {
        logger.Alwaysln("Wrong settings file format. Look JSON spec. Using default settings.")
        logger.Alwaysf("multi_thread: %d, delay: %d, logmode: %d, logswitch: %t", threads, delay, logmode, logswitch)
        return threads, delay, logmode, logswitch
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

    logger.Alwaysf("multi_thread: %d, delay: %d, logmode: %d, logswitch: %t", threads, delay, logmode, logswitch)
    return threads, delay, logmode, logswitch
}

// split url to path and filename
func parceurl(_url string) (string, string)  { // directory, filename
    urlstruct, err := url.Parse(_url)
    if err != nil {
        logger.Alwaysln("\tCan't parse url: " + err.Error())
        return "./", ";errname"
    }
    return "./data/" + urlstruct.Host + path.Dir(urlstruct.Path), path.Base(urlstruct.Path)
}

// Check database for file
func ShouldItBeDownloaded(url string) (bool)  {
    _path, _name := parceurl(url) // make filename 'name' and directory for file 'dir'
    if _name == ";errname" {
        return false
    }

    // save current dir
    maindir, err := os.Getwd()
    defer os.Chdir(maindir)
    if err != nil {
        return false
    }
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

// create new Get request, save response to file, returns status of request (done/not)
func NewRequest(_url string) (bool)  {

    dir, name := parceurl(_url) // make filename 'name' and directory for file 'dir'
    if name == ";errname" {
        return false
    }

    urlstruct, err := url.Parse(_url) // need host name
    if err != nil {
        logger.Alwaysln("\tCan't parse url: " + err.Error())
        return false
    }

    resp, err := http.Get(_url)
    if err != nil {
        logger.Alwaysln("\tGet error: " + err.Error())
        return false
    }

    data, err := ioutil.ReadAll(resp.Body) // convert to []byte
    if err != nil {
        logger.Alwaysln("\tCan't read from request: " + err.Error())
        return false
    }
    // filesystem shamaning
    curdir, err := os.Getwd()
    err = os.MkdirAll(dir, 0777)
    err = os.Chdir(dir)
    defer os.Chdir(curdir) // go back to main dir after request
    newfile, err := os.Create(name)
    if err != nil {
        logger.Alwaysln("\tCan't create file: " + err.Error())
        return false
    }
    defer newfile.Close()
    // clear page
    switch urlstruct.Host {
    case "ria.ru":
        clean_data, isok := cleaner.ClearRIA(data)
        if isok {
            data = clean_data
            logger.Alwaysf("\tClear")
        } else {
            logger.Alwaysf("\tError in cleaning, Default")
        }
    case "www.mk.ru":
        clean_data, isok := cleaner.ClearMK(data)
        if isok {
            data = clean_data
            logger.Alwaysf("\tClear")
        } else {
            logger.Alwaysf("\tError in cleaning, Default")
        }
    default:
        logger.Alwaysf("\tDefault")
    }


    _, err = newfile.Write(data)

    if err != nil {
        fullpath, err1 := os.Getwd()
        if err1 != nil {
            logger.Alwaysln("\tCan't return directory to file: " + err1.Error() + "\n")
        }
        logger.Alwaysln("\tWrite data error in" + fullpath + "/" + newfile.Name() + " : " + err.Error() + "\n")
        return false
    }



    logger.Alwaysln("\tOK")
    return true
}

type UrlNode struct {
    Url string
}

type UrlsJSON struct {
    Lists []UrlNode
    Pages []UrlNode
}

// create queue with urls
// parce 'urls.cfg' for default urls
// list of sitemaps, list of urls
func MakeFirstList() (*list.List, *list.List) {
    urlsfile, err := os.Open("urls.cfg")
    if err != nil {
      urlsfile, err = os.Create("urls.cfg")
      if err != nil {
          logger.Alwaysln("No 'urls.cfg' file. Can not create it!.")
      } else {
          logger.Alwaysln("No 'urls.cfg' file. It was created.")
          urlsfile.Write([]byte(_urlfile_defstr))
          urlsfile.Close()
      }
      return nil, nil
    }
    jtext, err := ioutil.ReadAll(urlsfile)
    if err != nil {
        logger.Alwaysln("Read error.")
        return nil, nil
    }
    urlsfile.Close()
    var s UrlsJSON
    err = json.Unmarshal(jtext, &s)
    if err != nil {
        logger.Alwaysln("Wrong settings file format. Look JSON spec.")
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
    logger.Init(0)
    defer logger.Alwaysln("LAST LOG MESSAGE")
    _, delay, logmode, logswitch := InitSettings() // open settings.cfg for settings
    if logswitch {
        logger.ChangeMode(logmode)
    } else {
        logger.Deactivate()
    }
    list_with_sitemaps, list_with_urls := MakeFirstList() // open urls.cfg for targets
    if list_with_urls == nil && list_with_sitemaps == nil {
        logger.Alwaysln("Nothing to download")
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
    logger.Moreln("Queue: OK ")
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
