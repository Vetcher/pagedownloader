package mylogger

import (
  "time"
  "log"
  "os"
  "fmt"
)

type Logger struct {
  filewriter *log.Logger
  displayprinter *log.Logger
}

func (logger *Logger) Println(str string) {
  if logger == nil {
    fmt.Println("variable is nil")
    return
  }
  logger.filewriter.Println(str)
  logger.displayprinter.Println(/*time.Now().Format("03:04:05 ") + */str)
}

func (logger *Logger) Printf(format string, a ...interface{}) {
  logger.displayprinter.Printf(format, a...)
  logger.filewriter.Printf(format, a...)
}

func (logger *Logger) Init()  {
  if logger == nil {
    fmt.Println("logger variable is nil")
    return
  }
  err := os.Mkdir("log", 0777)
  err = os.Chdir("log")
  logfile, err := os.Create(time.Now().Format("02-01-06.15_04_05") + ".log")
  if err != nil {
    fmt.Println("Can't create .log file")
    return
  }
  err = os.Chdir("..")
  *logger = Logger {
    filewriter: log.New(logfile, "[log] ", log.Ltime),
    displayprinter: log.New(os.Stdout, "[log] ", log.Ltime),
  }
  logger.Println("logger was successfully initialized")
}
