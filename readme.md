# ChronoLog
**Convenient, lightweight and understandable logging-tool for Golang.**

[![ChronoLog-Tests](https://github.com/wnderbin/ChronoLog/actions/workflows/chronolog_tests.yml/badge.svg)](https://github.com/wnderbin/ChronoLog/actions/workflows/chronolog_tests.yml)

ChronoLog is designed as part of a logging system, as it works well with files. It is not a complete solution, but only a component for working with logging and its management.

ChronoLog is a package for writing logs to files. The package is a lightweight and easy-to-use tool that writes data in json format and in plain text.

The package was created with an emphasis on the extensibility of its components, scalability and flexibility. You can change its functionality at your discretion, as well as extend it. It has a simple source code, which will allow anyone to easily change the functionality of this tool.

ChronoLog assumes that only one process will write. If logs are written by several processes with the same configuration, this may lead to incorrect behavior.

## ChronoLog Features
- [ ] `Writing logs to console`
- [X] `Writing logs in JSON format`
- [X] `Writing logs in plain text format`
- [X] `Compress large logs into .gz extension`
- [X] `Self-cleaning of old logs`
- [X] `Works well with multithreading`


### Installing a package into your project
You can install the package into your project using the command below.
```
go get github.com/wnderbin/ChronoLog
```
### Quick start
Once you install this package, you can use it immediately.
```
package main

import (
	"fmt"
	"time"

	chronolog "github.com/wnderbin/ChronoLog"
)

func main() {
	conf := chronolog.Config{
		FilePath:              "logs.json",
		MaxSize:               432,
		MaxAge:                time.Second * 5,
		CompressOldLogs:       true,
		JSONFormat:            true,
		TimestampFormat:       time.RFC3339,
		RotationCheckInterval: time.Second * 5,
	}
	chronologger, err := chronolog.New(conf)
	if err != nil {
		fmt.Println("Error from chronolog/New(): %w", err)
	}

	for {
		chronologger.Debug("debug message")
		chronologger.Error("error message")
		chronologger.Fatal("fatal message")
		chronologger.Info("info message")
		chronologger.Warning("warning message")
		time.Sleep(5 * time.Second)
	}
}
```

#### Let's take a closer look at the code
1. **Setting up the logger configuration.
The configuration has the following fields:**
```
type Config struct {
	FilePath              string
	MaxSize               int64         
	MaxAge                time.Duration
	CompressOldLogs       bool          
	JSONFormat            bool          
	TimestampFormat       string        
	RotationCheckInterval time.Duration 
}
```
* **FilePath** - Path to the file where logs should be written. If the file is missing, ChronoLog will initialize it automatically.
* **MaxSize** - Maximum log size, specified in bytes. The default log size is 50 megabytes. If the logs exceed this size, the file will be compressed depending on the settings you specify below. If you do not need to compress the log file, ChronoLog will simply save the file with the date at the time the file overflowed.
* **MaxAge** - Maximum storage time for overflow/compressed files.
* **CompressOldLogs** - Parameter, if true is specified - when the file is resized, it will start compressing it with the extension .gz. Otherwise, compression will not occur.
* **JSONFormat** - The parameter that is responsible for writing the file in JSON format. If true is specified, the file will be written in JSON format. Otherwise, it will be written in plain text format.
* **TimestampFormat** - Parameter that specifies the time format in which logs should be written.
* **RotationCheckInterval** - The period of checking the file overflow and performing its rotation.

2. **Creating a logger**
The logger is created using the New() function, which returns the logger object and the error, if any. Using the received object, you can write logs to the file specified in the configuration. The New() function also sets default settings if none have been specified.
```
chronologger, err := chronolog.New(conf)
```
```
func New(config Config) (*Logger, error) {
	if config.MaxSize == 0 {
		config.MaxSize = 50 * 1024 * 1024 // default size = 50MB
	}
	if config.MaxAge == 0 {
		config.MaxAge = 7 * 24 * time.Hour // default max age = 1 week
	}
	if config.TimestampFormat == "" {
		config.TimestampFormat = time.RFC3339 // default timestamp format = "2006-01-02T15:04:05Z07:00"
	}
	if config.RotationCheckInterval == 0 {
		config.RotationCheckInterval = time.Minute // check rotation every minute
	}

	l := &Logger{
		config:   config,
		quitChan: make(chan struct{}),
	}

	if err := l.openFile(); err != nil {
		return nil, err
	}

	go l.rotationChecker()

	return l, nil
}
```

3. **Writing logs. As mentioned above, writing logs will be done via the object reference you got earlier from the New() function.**
```
chronologger, err := chronolog.New(conf)
if err != nil {
	fmt.Println("Error from chronolog/New(): %w", err)
}
chronologger.Debug("debug message")
chronologger.Error("error message")
chronologger.Fatal("fatal message")
chronologger.Info("info message")
chronologger.Warning("warning message")
```

4. After using the logger, you can close it using the Close() function. This will stop it from consuming resources and close the file.
```
chronologger.Close()
```
```
func (l *Logger) Close() error {
	close(l.quitChan)
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
```

### Author
* wnderbin
