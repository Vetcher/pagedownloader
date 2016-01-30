package main

import (
    "encoding/xml"
    "encoding/json"
    "net/http"
    "net/url"
    "io/ioutil"
    "container/list"
    "fmt"
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
    default_delay int = 15
)

type SettingsJSON struct {
    Multi_thread int
    Delay int
}

// return vars about init setups
// 1. multi_thread: 0 if should use one thread, else 1
// 2. delay: delay in seconds between requests
func InitSettings() (int, int)  {
    // open settings
    settingsfile, err := os.Open("settings.cfg")
    if err != nil {
      settingsfile, err = os.Create("settings.cfg")
      if err != nil {
          logger.Println("No 'setting.cfg' file. Can not create it!. Using default settings.")
      } else {
          logger.Println("No 'setting.cfg' file. It was created. Using default settings.")
          settingsfile.Close()
      }
      logger.Printf("multi_thread: %d, delay: %d", default_threads, default_delay)
      return default_threads, default_delay
    }
    jtext, err := ioutil.ReadAll(settingsfile)
    if err != nil {
        logger.Println("Read error. Using default settings.")
        logger.Printf("multi_thread: %d, delay: %d", default_threads, default_delay)
        return default_threads, default_delay
    }
    settingsfile.Close()
    var s SettingsJSON
    err = json.Unmarshal(jtext, &s)
    if err != nil {
        logger.Println("Wrong settings file format. Look JSON spec. Using default settings.")
        logger.Printf("multi_thread: %d, delay: %d", default_threads, default_delay)
        return default_threads, default_delay
    }
    var threads, delay int
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

    logger.Printf("multi_thread: %d, delay: %d", threads, delay)
    return threads, delay
}

// split url to path and filename
func parceurl(_url string) (string, string)  { // directory, filename
    urlstruct, err := url.Parse(_url)
    if err != nil {
        logger.Println(_url + " | Can't parse url: " + err.Error())
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

// create new Get request, save response to file
func NewRequest(_url string)  {

    dir, name := parceurl(_url) // make filename 'name' and directory for file 'dir'
    if name == ";errname" {
        return
    }

    urlstruct, err := url.Parse(_url) // need host name
    if err != nil {
        logger.Println(dir + name + " | Can't parse url: " + err.Error())
        return
    }

    resp, err := http.Get(_url)
    if err != nil {
        logger.Println(dir + name + " | " + err.Error())
    }

    data, err := ioutil.ReadAll(resp.Body) // convert to []byte

    // filesystem shamaning
    curdir, err := os.Getwd()
    err = os.MkdirAll(dir, 0777)
    err = os.Chdir(dir)
    newfile, err := os.Create(name)

    // clear page
    switch urlstruct.Host {
    case "ria.ru":
        clean_data, isok := cleaner.ClearRIA(data)
        if isok {
            data = clean_data
            logger.Println(dir + name + " | OK, Clear")
        } else {
            logger.Println(dir + name + " | Error in cleaning, Default")
        }
    case "www.mk.ru":
        clean_data, isok := cleaner.ClearMK(data)
        if isok {
            data = clean_data
            logger.Println(dir + name + " | OK, Clear")
        } else {
            logger.Println(dir + name + " | Error in cleaning, Default")
        }
    default:
        logger.Println(dir + name + " | OK, Default")
    }

    _, err = newfile.Write(data)

    if err != nil {
        logger.Println(dir + name + " | " + err.Error() + "\n")
        return
    }

    newfile.Close()
    err = os.Chdir(curdir)
}

type UrlNode struct {
    Url string
}

type UrlsJSON struct {
    Lists []UrlNode
    Pages []UrlNode
}

// parce 'urls.cfg' for default urls
// list of sitemaps, list of urls
func MakeFirstList() (*list.List, *list.List) {
    urlsfile, err := os.Open("urls.cfg")
    if err != nil {
      urlsfile, err = os.Create("urls.cfg")
      if err != nil {
          logger.Println("No 'urls.cfg' file. Can not create it!.")
      } else {
          logger.Println("No 'urls.cfg' file. It was created.")
          urlsfile.Close()
      }
      return nil, nil
    }
    jtext, err := ioutil.ReadAll(urlsfile)
    if err != nil {
        logger.Println("Read error.")
        return nil, nil
    }
    urlsfile.Close()
    var s UrlsJSON
    err = json.Unmarshal(jtext, &s)
    if err != nil {
        logger.Println("Wrong settings file format. Look JSON spec.")
        return nil, nil
    }
    sitemap := list.New()
    for _, elem := range s.Lists {
        sitemap.PushBack(elem.Url)
    }
    if sitemap.Len() == 0 {
        sitemap = nil
        logger.Println("no sitemaps")
    }
    listofurls := list.New()
    for _, elem := range s.Pages {
        listofurls.PushBack(elem.Url)
    }
    if listofurls.Len() == 0 {
        listofurls = nil
        logger.Println("no direct urls")
    }
    return sitemap, listofurls
}

type XMLSTRUCT struct {
    XMLName xml.Name
    Urls []string `xml:"url>loc"`
}

func GetAndParseXML(_xml_url string, _queue *list.List) (int) {
    if _queue == nil {
        logger.Println("Error: queue is nil")
        return 0
    }
    // Get .xml file from server
    xmlfile, err := http.Get(_xml_url)
    if err != nil {
        logger.Println("Get \"" + _xml_url + "\": " + err.Error())
        return 0
    }
    logger.Println("Get \"" + _xml_url + "\": OK")
    xmltext, err := ioutil.ReadAll(xmlfile.Body)

    // tokenize .xml file
    var xmldoc XMLSTRUCT
    err = xml.Unmarshal([]byte(xmltext), &xmldoc)
    if err != nil {
        logger.Println("Unmarshal \"" + _xml_url + "\" failed: " + err.Error())
        return 0
    }
    logger.Println("Unmarshal \"" + _xml_url + "\": OK ")

    count := len(xmldoc.Urls)
    for index := 0; index < count; index++ {
        if !ShouldItBeDownloaded(xmldoc.Urls[index]) {
            logger.Println(xmldoc.Urls[index] + " already downloaded")
            continue
        }
        _queue.PushBack(xmldoc.Urls[index])
    }
    return count
}

func main()  {
    logger.Init()
    _, delay := InitSettings() // open settings.cfg for settings
    list_with_sitemaps, list_with_urls := MakeFirstList() // open urls.cfg for targets
    if list_with_urls == nil && list_with_sitemaps == nil {
        logger.Println("Nothing to download")
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

    logger.Println("Queue: OK ")
    logger.Printf("Founded %d links,\n", count)
    logger.Printf("Already downloaded %d,\n", count - queueOfUrls.Len())
    logger.Printf("In queue %d links.\n", queueOfUrls.Len())
    i := 0
    queuetimer := time.NewTicker(time.Second * time.Duration(delay))
    logger.Println("Start taker:")
	   for {
           select {
            case <- queuetimer.C:
                if queueOfUrls.Front() != nil {
                    fmt.Print(i)
                    fmt.Print(": ")
                    temp := queueOfUrls.Front()
		            NewRequest(temp.Value.(string))
                    queueOfUrls.Remove(temp)
                    i++
                } else {
                    return
                }
            }
        }
}
