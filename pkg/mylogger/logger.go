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
    version int
}

func (logger *Logger) Println(str string) {
    if logger.filewriter != nil {
        logger.filewriter.Println(str)
    }
    if logger.displayprinter != nil {
        logger.displayprinter.Println(str)
    }
}

func (logger *Logger) Printf(format string, a ...interface{}) {
    if logger.filewriter != nil {
        logger.displayprinter.Printf(format, a...)
    }
    if logger.displayprinter != nil {
        logger.filewriter.Printf(format, a...)
    }
}

func (logger *Logger) Alwaysln(str string) {
    logger.Println("ALWAYS " + str)
}

func (logger *Logger) Alwaysf(format string, a ...interface{}) {
    logger.Printf("ALWAYS " + format, a...)
}

func (logger *Logger) Moreln(str string) {
    if logger.version >= 1 {
        logger.Println("MORE " + str)
    }
}

func (logger *Logger) Moref(format string, a ...interface{}) {
    if logger.version >= 1 {
        logger.Printf("MORE " + format, a...)
    }
}

func (logger *Logger) Debugln(str string) {
    if logger.version >= 2 {
        logger.Println("DEBUG " + str)
    }
}

func (logger *Logger) Debugf(format string, a ...interface{}) {
    if logger.version >= 2 {
        logger.Printf("DEBUG " + format, a...)
    }
}

// Init(0) - no log messages, activates methods Alwaysln, Alwaysf
// Init(1) - default log messages, activates methods Moreln, Moref
// Init(2) - all log messages, activates methods Debugln, Debugf
func (logger *Logger) Init(_version int)  {
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
        filewriter: log.New(logfile, "", log.Ltime),
        displayprinter: log.New(os.Stdout, "", log.Ltime),
        version: _version,
    }
    logger.Alwaysln("Logger was successfully initialized")
}

func (logger *Logger) Deactivate() {
    logger.Alwaysln("Logger was deactivated")
    logger.filewriter = nil
    logger.displayprinter = nil
}

func (logger *Logger) ChangeMode(mode int) {
    if logger != nil {
        if mode > 2 || mode < 0 {
            mode = 1
        }
        logger.version = mode
        logger.Alwaysf("Logger mode switched to %d\n", mode)
    }
}
